// Package golang implements the LanguageDriver for Go projects.
// Uses stdlib go/ast and go/parser — no tree-sitter dependency.
package golang

import (
	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for Go + testing package.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string            { return "go" }
func (d *Driver) FileExtensions() []string    { return []string{".go"} }
func (d *Driver) BodyGenerationPrompt() string { return goBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "t.Errorf / testify/assert",
		"framework":      "testing",
		"test_prefix":    "Test",
	}
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "testing",
		TestFileSuffix: "_test.go",
		FixtureDir:     "",
		BootstrapFile:  "",
		TestFuncPrefix: "Test",
	}
}

func (d *Driver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return detectProject(dir)
}

func (d *Driver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return analyzeFile(path, ctx)
}

func (d *Driver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return classifyDependency(dep, ctx)
}

func (d *Driver) DeriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return deriveTestPath(sourcePath, ctx)
}

func (d *Driver) DeriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return deriveModulePath(sourcePath, ctx)
}

func (d *Driver) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return generateTestFile(analysis, opts)
}

func (d *Driver) GenerateFixture(_ string, _ *domain.SourceAnalysis, _ domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil // Go uses interface mocks, not shared fixture files.
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil // Go has no bootstrap file equivalent.
}

const goBodyPrompt = `You are an expert Go testing assistant.
Write table-driven tests for the function named ` + "`{{.MemberName}}`" + ` using the standard ` + "`testing`" + ` package.

Source:
` + "```go\n{{.SourceCode}}\n```" + `

Requirements:
- Use table-driven tests with a slice of structs.
- Cover the happy path and at least one error/edge case.
- Use t.Errorf for assertions (no external dependencies).
- Output ONLY valid Go code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return goMigrators }
