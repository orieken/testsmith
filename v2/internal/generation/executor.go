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
type Executor struct{}

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
		case domain.ActionCreate:
			err = writeFile(f.AbsPath, f.Content)
		case domain.ActionUpdate:
			err = writeFile(f.AbsPath, f.Content)
		case domain.ActionSkip:
			// nothing to do
		}
		results = append(results, Result{AbsPath: f.AbsPath, Action: f.Action, Role: f.Role, Err: err})
	}
	return results, nil
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
