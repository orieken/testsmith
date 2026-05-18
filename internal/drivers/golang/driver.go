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
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "go"
	return vocab
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
Write a comprehensive Go test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in package ` + "`{{.ModulePath}}`" + `.

Source:
` + "```go\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this package:
` + "```go\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}
Test function prefix: {{index .Extra "test_prefix"}}

Requirements:
- Use table-driven tests where appropriate.
- Cover the happy path and at least one error/edge case.
- Output ONLY valid Go code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return goMigrators }
