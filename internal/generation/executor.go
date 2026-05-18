// Package generation contains the GenerationPipeline and the Executor that
// writes GenerationPlans to disk. The Executor is the ONLY place in the
// codebase that performs filesystem writes.
package generation

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orieken/testsmith/internal/domain"
)

// Result describes the outcome of executing a single GeneratedFile entry.
type Result struct {
	AbsPath string
	Action  domain.FileAction
	Role    domain.FileRole
	Err     error
}

// Executor writes a GenerationPlan to the filesystem.
// Attach language-specific verifiers via WithVerifiers to check generated test
// files compile after they are written.
type Executor struct {
	verifiers map[string]Verifier // language → verifier; nil = no compile checks
}

// WithVerifiers returns a copy of the Executor configured to run compile
// verification on written test files. Pass the result of VerifierFor for each
// language in scope, or use NewVerifiedExecutor for the common case.
func (e *Executor) WithVerifiers(v map[string]Verifier) *Executor {
	return &Executor{verifiers: v}
}

// NewVerifiedExecutor returns an Executor pre-loaded with verifiers for all
// languages that have a supported compile check.
func NewVerifiedExecutor(language string) *Executor {
	v := VerifierFor(language)
	if v == nil {
		return &Executor{}
	}
	return &Executor{verifiers: map[string]Verifier{language: v}}
}

// Execute writes each file in the plan. When plan.DryRun is true it returns
// the same Results but performs no writes.
func (e *Executor) Execute(plan *domain.GenerationPlan) ([]Result, error) {
	var results []Result
	for _, f := range plan.Files {
		if plan.DryRun {
			results = append(results, Result{
				AbsPath: f.AbsPath,
				Action:  f.Action,
				Role:    f.Role,
			})
			continue
		}

		var err error
		switch f.Action {
		case domain.ActionCreate, domain.ActionUpdate:
			err = writeFile(f.AbsPath, f.Content)
			if err == nil && f.Role == domain.RoleTestFile {
				err = e.verifyFile(f.AbsPath, f.Language)
			}
		case domain.ActionSkip:
			// nothing to do
		}
		results = append(results, Result{AbsPath: f.AbsPath, Action: f.Action, Role: f.Role, Err: err})
	}
	return results, nil
}

// verifyFile runs the language-appropriate compile check after writing a test
// file. Returns nil when no verifier is registered for the language.
func (e *Executor) verifyFile(path, language string) error {
	if e.verifiers == nil {
		return nil
	}
	v, ok := e.verifiers[language]
	if !ok {
		return nil
	}
	return v.Verify(path)
}

func writeFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create directories for %s: %w", path, err)
	}
	// Write to a temp file then rename for atomicity.
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write temp file %s: %w", tmp, err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("rename %s -> %s: %w", tmp, path, err)
	}
	return nil
}
