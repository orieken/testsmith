// Package kotlin implements the LanguageDriver for Kotlin projects (JVM and KMP).
// Source analysis is regex-based; no tree-sitter dependency required.
package kotlin

import (
	"github.com/orieken/assay/internal/domain"
)

// Driver implements domain.LanguageDriver for Kotlin + JUnit 5.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "kotlin" }
func (d *Driver) FileExtensions() []string { return []string{".kt", ".kts"} }
func (d *Driver) BodyGenerationPrompt() string {
	return kotlinBodyPrompt
}

func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "kotlin"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit5",
		TestFileSuffix: "Test.kt",
		FixtureDir:     "test",
		BootstrapFile:  "",
		TestFuncPrefix: "",
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
	return nil, nil // Kotlin uses MockK/Mockito mocks, not shared fixture files.
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return kotlinMigrators }

func (d *Driver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }

const kotlinBodyPrompt = `You are an expert Kotlin testing assistant.
Write a comprehensive JVM test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in ` + "`{{.ModulePath}}`" + `.

Source:
` + "```kotlin\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this module:
` + "```kotlin\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}
Framework: {{index .Extra "framework"}}

Requirements:
- Use @Test from JUnit 5 (org.junit.jupiter.api.Test).
- Use MockK for mocking: mockk<T>(), every { }, verify { }.
- Cover the happy path and at least one error/edge case.
- Prefer @ParameterizedTest with @MethodSource for table-driven tests.
- Output ONLY valid Kotlin code in a single markdown code block.`
