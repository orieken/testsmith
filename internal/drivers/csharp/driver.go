// Package csharp implements the LanguageDriver for C# projects.
// Supports MSBuild project structures (.csproj/.sln).
// xUnit + Moq test generation (framework auto-detected).
package csharp

import (
	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for C# + xUnit/NUnit/MSTest.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string             { return "csharp" }
func (d *Driver) FileExtensions() []string     { return []string{".cs"} }
func (d *Driver) BodyGenerationPrompt() string { return csharpBodyPrompt }
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "csharp"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "xunit",
		TestFileSuffix: "Tests.cs",
		FixtureDir:     "",
		BootstrapFile:  "",
		TestFuncPrefix: "[Fact]",
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
	return nil, nil // C# uses Moq constructor injection, not shared fixture files.
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil // C# test discovery is assembly-based; no bootstrap file needed.
}

const csharpBodyPrompt = `You are an expert C# testing assistant.
Write a comprehensive {{index .Extra "framework"}} test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + `.

Source:
` + "```csharp\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this class:
` + "```csharp\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Test attribute: {{index .Extra "test_attribute"}}
Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}

Requirements:
- Include a happy-path test and at least one edge-case.
- Follow Arrange / Act / Assert comment structure.
- Use the mock and assert styles shown above.
- Output ONLY valid C# code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return csMigrators }
