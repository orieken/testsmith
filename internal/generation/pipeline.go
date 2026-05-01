package generation

import (
	"context"
	"os"

	"github.com/orieken/testsmith/internal/domain"
)

// Pipeline builds GenerationPlans from SourceAnalyses.
type Pipeline struct {
	driver domain.LanguageDriver
	llm    domain.BodyGenerator // nil = stub bodies
}

// NewPipeline returns a Pipeline. Pass nil for llm to disable LLM body generation.
func NewPipeline(driver domain.LanguageDriver, llm domain.BodyGenerator) *Pipeline {
	return &Pipeline{driver: driver, llm: llm}
}

// Plan builds the GenerationPlan for a single source file analysis.
// No I/O is performed here — the plan is pure data.
func (p *Pipeline) Plan(
	ctx context.Context,
	analysis *domain.SourceAnalysis,
	opts domain.GenerateOpts,
) (*domain.GenerationPlan, error) {
	plan := &domain.GenerationPlan{DryRun: opts.DryRun}

	// Optionally enrich opts with LLM-generated bodies.
	if p.llm != nil && !opts.DryRun {
		bodies, err := p.fetchBodies(ctx, analysis)
		if err == nil {
			opts.LLMBodies = bodies
		}
		// Non-fatal: LLM failure falls back to stubs.
	}

	// Derive fixture imports from external dependencies.
	extModules := uniqueRootModules(analysis.Imports.External)
	for _, dep := range extModules {
		fixture, err := p.driver.GenerateFixture(dep, analysis, opts)
		if err != nil || fixture == nil {
			continue
		}
		fixture.Action = resolveAction(fixture.AbsPath, opts.OverwriteExisting)
		plan.Files = append(plan.Files, *fixture)
		opts.FixtureImports = append(opts.FixtureImports, domain.FixtureImport{
			Module:   dep,
			FuncName: "mock_" + dep,
		})
	}

	// Generate test file.
	testPath, err := p.driver.DeriveTestPath(analysis.SourcePath, analysis.Project)
	if err != nil {
		return nil, err
	}
	testAction := resolveAction(testPath, opts.OverwriteExisting)
	if testAction != domain.ActionSkip {
		testFile, err := p.driver.GenerateTestFile(analysis, opts)
		if err != nil {
			return nil, err
		}
		testFile.Action = testAction
		plan.Files = append(plan.Files, *testFile)
	} else {
		plan.Files = append(plan.Files, domain.GeneratedFile{
			AbsPath: testPath,
			Action:  domain.ActionSkip,
			Role:    domain.RoleTestFile,
		})
	}

	// Generate / update bootstrap file (conftest.py, jest.setup.ts, etc.).
	bootstrap, err := p.driver.GenerateBootstrap(plan, analysis.Project)
	if err == nil && bootstrap != nil {
		plan.Files = append(plan.Files, *bootstrap)
	}

	return plan, nil
}

func (p *Pipeline) fetchBodies(ctx context.Context, analysis *domain.SourceAnalysis) (map[string][]string, error) {
	var results []domain.BodyGenResult
	fwCfg := p.driver.GetTestFrameworkConfig()
	extra := p.driver.LLMContext()

	for _, member := range analysis.PublicAPI {
		req := domain.BodyGenRequest{
			Language:   analysis.Project.Language,
			MemberName: member.Name,
			MemberKind: member.Kind,
			SourceCode: analysis.RawSource,
			Framework:  fwCfg,
			Extra:      extra,
		}
		partial, err := p.llm.GenerateBodies(ctx, req)
		if err == nil {
			results = append(results, partial...)
		}
	}

	bodies := make(map[string][]string, len(results))
	for _, r := range results {
		bodies[r.MemberName] = r.CodeLines
	}
	return bodies, nil
}

// resolveAction returns Create if the file doesn't exist, Skip if it does
// (unless overwrite is true), or Update when overwrite is true and file exists.
func resolveAction(path string, overwrite bool) domain.FileAction {
	if _, err := os.Stat(path); os.IsNotExist(err) {
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
