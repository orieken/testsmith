package generation

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/projectknowledge"
)

// Pipeline builds GenerationPlans from SourceAnalyses.
type Pipeline struct {
	driver            domain.LanguageDriver
	llm               domain.BodyGenerator
	depIndex          map[string]*domain.SourceAnalysis // module path → analysis; nil = no cross-file context
	promptTokenBudget int                               // 0 = unlimited
}

// NewPipeline returns a Pipeline. Pass nil for llm to disable LLM body generation.
func NewPipeline(driver domain.LanguageDriver, llm domain.BodyGenerator) *Pipeline {
	return &Pipeline{driver: driver, llm: llm}
}

// WithDepIndex attaches a project-wide analysis index so the pipeline can include
// internal dependency signatures in the LLM prompt.
func (p *Pipeline) WithDepIndex(idx map[string]*domain.SourceAnalysis) *Pipeline {
	p.depIndex = idx
	return p
}

// WithPromptTokenBudget sets the maximum estimated token count for LLM user
// prompts. Lower-priority tiers (style snippet, dep signatures) are dropped when
// the total exceeds the budget. Pass 0 to disable the limit.
func (p *Pipeline) WithPromptTokenBudget(budget int) *Pipeline {
	p.promptTokenBudget = budget
	return p
}

// UpdateDepEntry refreshes a single entry in the dep index. It is a no-op when
// modulePath is empty or no dep index is attached (LLM disabled, single-file run).
// Called by the file watcher when a source file changes so subsequent generations
// of files that import it receive up-to-date public API signatures.
func (p *Pipeline) UpdateDepEntry(modulePath string, a *domain.SourceAnalysis) {
	if p.depIndex == nil || modulePath == "" {
		return
	}
	p.depIndex[modulePath] = a
}

// Plan builds the GenerationPlan for a single source file analysis.
// No I/O is performed here — the plan is pure data.
func (p *Pipeline) Plan(
	ctx context.Context,
	analysis *domain.SourceAnalysis,
	opts domain.GenerateOpts,
) (*domain.GenerationPlan, error) {
	plan := &domain.GenerationPlan{DryRun: opts.DryRun}

	if p.llm != nil && !opts.DryRun {
		opts.LLMBodies = p.fetchBodies(ctx, analysis)
	}

	extModules := uniqueRootModules(analysis.Imports.External)
	for _, dep := range extModules {
		fixture, err := p.driver.GenerateFixture(dep, analysis, opts)
		if err != nil || fixture == nil {
			continue
		}
		fixture.Action = resolveAction(fixture.AbsPath, opts.OverwriteExisting, false)
		plan.Files = append(plan.Files, *fixture)
		opts.FixtureImports = append(opts.FixtureImports, domain.FixtureImport{
			Module:   dep,
			FuncName: "mock_" + dep,
		})
	}

	testPath, err := p.driver.DeriveTestPath(analysis.SourcePath, analysis.Project)
	if err != nil {
		return nil, err
	}
	testAction := resolveAction(testPath, opts.OverwriteExisting, opts.TestFileKnownNew)
	if testAction != domain.ActionSkip {
		testFile, err := p.driver.GenerateTestFile(analysis, opts)
		if err != nil {
			return nil, err
		}
		testFile.Action = testAction
		testFile.Language = analysis.Project.Language
		plan.Files = append(plan.Files, *testFile)
	} else {
		plan.Files = append(plan.Files, domain.GeneratedFile{
			AbsPath:  testPath,
			Action:   domain.ActionSkip,
			Role:     domain.RoleTestFile,
			Language: analysis.Project.Language,
		})
	}

	bootstrap, err := p.driver.GenerateBootstrap(plan, analysis.Project)
	if err == nil && bootstrap != nil {
		plan.Files = append(plan.Files, *bootstrap)
	}

	return plan, nil
}

func (p *Pipeline) fetchBodies(ctx context.Context, analysis *domain.SourceAnalysis) map[string][]string {
	fwCfg := p.driver.GetTestFrameworkConfig()

	// Adapter-aware vocabulary for the specific project's chosen framework.
	extra := p.driver.LLMContext(analysis.Project)
	if extra == nil {
		extra = make(map[string]string)
	}

	// Project-derived facts, constant across all members in this file.
	extra["module_path"] = analysis.ModulePath
	extra["module_name"] = analysis.ModuleName()
	extra["language"] = analysis.Project.Language
	extra["external_deps"] = externalDepList(analysis.Imports.External)

	rawDeps := buildDepsSignatures(analysis, p.depIndex)
	rawSnippet := mineConventions(analysis.SourcePath, fwCfg)
	projectKnow := projectknowledge.LoadForFile(analysis.SourcePath, analysis.Project.Root)
	if projectKnow == "" {
		projectKnow = analysis.Project.ProjectKnowledge
	}

	// Apply token budget: trim lower-priority tiers when the combined prompt
	// would exceed p.promptTokenBudget. Priority 1 = must-keep.
	tiers := []projectknowledge.Tier{
		{Name: "source", Content: analysis.RawSource, Priority: 1},
		{Name: "deps", Content: rawDeps, Priority: 2},
		{Name: "snippet", Content: rawSnippet, Priority: 3},
	}
	trimmed := projectknowledge.TrimToBudget(tiers, p.promptTokenBudget)
	depsSignatures := trimmed[1]
	styleSnippet := trimmed[2]

	// Build one request per public member — shared fields are identical across all.
	reqs := make([]domain.BodyGenRequest, len(analysis.PublicAPI))
	for i, member := range analysis.PublicAPI {
		reqs[i] = domain.BodyGenRequest{
			Language:            analysis.Project.Language,
			MemberName:          member.Name,
			MemberKind:          member.Kind,
			SourceCode:          analysis.RawSource,
			Framework:           fwCfg,
			Extra:               extra,
			ModulePath:          analysis.ModulePath,
			DepsSignatures:      depsSignatures,
			ExistingTestSnippet: styleSnippet,
			ProjectKnowledge:    projectKnow,
		}
	}

	// Prefer the batch path (one API call for all members) when available.
	// Fall back to the parallel per-member fan-out for other implementations.
	var results []domain.BodyGenResult
	if batcher, ok := p.llm.(domain.BatchBodyGenerator); ok {
		results, _ = batcher.GenerateBatchBodies(ctx, reqs)
	} else {
		var mu sync.Mutex
		var wg sync.WaitGroup
		for _, req := range reqs {
			req := req
			wg.Add(1)
			go func() {
				defer wg.Done()
				partial, err := p.llm.GenerateBodies(ctx, req)
				if err == nil {
					mu.Lock()
					results = append(results, partial...)
					mu.Unlock()
				}
			}()
		}
		wg.Wait()
	}

	bodies := make(map[string][]string, len(results))
	for _, r := range results {
		bodies[r.MemberName] = r.CodeLines
	}
	return bodies
}

// mineConventions scans up to maxConventionFiles test files in the same
// directory as sourcePath and returns a consolidated style sample. Reading
// multiple files instead of a single 40-line snippet gives the LLM a far more
// representative picture of the project's test conventions.
//
// Each contributing file contributes its first maxLinesPerFile lines, separated
// by a file-name header. Total output is capped at maxTotalLines to stay within
// prompt token budgets.
func mineConventions(sourcePath string, fwCfg domain.TestFrameworkConfig) string {
	const (
		maxConventionFiles = 5
		maxLinesPerFile    = 20
		maxTotalLines      = 80
	)

	var dir string
	if idx := strings.LastIndexAny(sourcePath, "/\\"); idx >= 0 {
		dir = sourcePath[:idx]
	} else {
		dir = "."
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	suffix := fwCfg.TestFileSuffix
	prefix := fwCfg.TestFilePrefix
	if suffix == "" && prefix == "" {
		return ""
	}

	var sb strings.Builder
	totalLines := 0
	filesRead := 0

	for _, e := range entries {
		if filesRead >= maxConventionFiles || totalLines >= maxTotalLines {
			break
		}
		if e.IsDir() {
			continue
		}
		name := e.Name()
		// Skip the test file that corresponds to sourcePath itself —
		// we want style examples from other tests in the same package.
		if deriveTestPathFromConfig(sourcePath, fwCfg) == dir+"/"+name {
			continue
		}
		isTestFile := (suffix != "" && strings.HasSuffix(name, suffix)) ||
			(prefix != "" && strings.HasPrefix(name, prefix))
		if !isTestFile {
			continue
		}

		data, err := os.ReadFile(dir + "/" + name)
		if err != nil {
			continue
		}
		lines := strings.Split(string(data), "\n")
		if len(lines) > maxLinesPerFile {
			lines = lines[:maxLinesPerFile]
		}

		if sb.Len() > 0 {
			sb.WriteString("\n\n")
		}
		fmt.Fprintf(&sb, "// --- %s ---\n", name)
		sb.WriteString(strings.Join(lines, "\n"))
		totalLines += len(lines)
		filesRead++
	}
	return sb.String()
}

// deriveTestPathFromConfig does a best-effort test path derivation using only
// the TestFrameworkConfig, without needing the full driver. Used solely to
// locate existing test files for style context.
func deriveTestPathFromConfig(sourcePath string, fwCfg domain.TestFrameworkConfig) string {
	if fwCfg.TestFileSuffix == "" {
		return ""
	}
	dot := strings.LastIndex(sourcePath, ".")
	if dot < 0 {
		return sourcePath + fwCfg.TestFileSuffix
	}
	return sourcePath[:dot] + fwCfg.TestFileSuffix
}

// buildDepsSignatures produces a compact text block listing the public API of
// each internal dependency found in analysis.Imports.Internal.
// Returns empty string when no dep index is available or no internal deps exist.
func buildDepsSignatures(analysis *domain.SourceAnalysis, idx map[string]*domain.SourceAnalysis) string {
	if len(idx) == 0 || len(analysis.Imports.Internal) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, imp := range analysis.Imports.Internal {
		dep, ok := idx[imp.Module]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "// %s\n", imp.Module)
		for _, m := range dep.PublicAPI {
			fmt.Fprintf(&sb, "  %s %s", m.Kind, m.Name)
			if len(m.Parameters) > 0 {
				params := make([]string, 0, len(m.Parameters))
				for _, param := range m.Parameters {
					if param.TypeHint != "" {
						params = append(params, param.Name+" "+param.TypeHint)
					} else {
						params = append(params, param.Name)
					}
				}
				fmt.Fprintf(&sb, "(%s)", strings.Join(params, ", "))
			}
			sb.WriteByte('\n')
		}
	}
	return sb.String()
}

// externalDepList returns a comma-separated list of root external package names.
func externalDepList(imports []domain.ImportInfo) string {
	seen := make(map[string]bool)
	var names []string
	for _, imp := range imports {
		root := rootPackage(imp.Module)
		if !seen[root] {
			seen[root] = true
			names = append(names, root)
		}
	}
	return strings.Join(names, ", ")
}

// resolveAction determines whether a generated file should be created, updated,
// or skipped. Pass knownNew=true when the caller has already verified the file
// does not exist (e.g. after DiscoverUntested) to avoid a redundant os.Stat.
func resolveAction(path string, overwrite, knownNew bool) domain.FileAction {
	if knownNew {
		return domain.ActionCreate
	}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return domain.ActionCreate
	}
	if overwrite {
		return domain.ActionUpdate
	}
	return domain.ActionSkip
}

func uniqueRootModules(imports []domain.ImportInfo) []string {
	seen := make(map[string]bool)
	var out []string
	for _, imp := range imports {
		root := rootPackage(imp.Module)
		if !seen[root] {
			seen[root] = true
			out = append(out, root)
		}
	}
	return out
}

func rootPackage(module string) string {
	for i, c := range module {
		if c == '.' || c == '/' {
			return module[:i]
		}
	}
	return module
}
