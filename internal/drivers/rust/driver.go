// Package rust implements the LanguageDriver for Rust projects.
// Source analysis is regex-based; no tree-sitter dependency required.
package rust

import (
	"github.com/orieken/assay/internal/domain"
)

// Driver implements domain.LanguageDriver for Rust + built-in #[test].
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "rust" }
func (d *Driver) FileExtensions() []string { return []string{".rs"} }
func (d *Driver) BodyGenerationPrompt() string {
	return rustBodyPrompt
}

func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "rust"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "test",
		TestFileSuffix: "_test.rs",
		FixtureDir:     "tests",
		BootstrapFile:  "",
		TestFuncPrefix: "test_",
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
	return nil, nil
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return rustMigrators }

func (d *Driver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }

const rustBodyPrompt = `You are an expert Rust testing assistant.
Write a comprehensive Rust test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in ` + "`{{.ModulePath}}`" + `.

Source:
` + "```rust\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this crate:
` + "```rust\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Test style: {{index .Extra "assert_style"}}

Requirements:
- Use #[test] attribute for synchronous tests, #[tokio::test] for async.
- Cover the happy path and at least one error/edge case.
- Use assert_eq!, assert!, or the configured assertion style.
- Output ONLY valid Rust code in a single markdown code block.`
