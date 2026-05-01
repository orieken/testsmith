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

func (d *Driver) Language() string            { return "csharp" }
func (d *Driver) FileExtensions() []string    { return []string{".cs"} }
func (d *Driver) BodyGenerationPrompt() string { return csharpBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "Assert.Equal / Assert.True (xUnit)",
		"framework":      "xUnit",
		"mock_library":   "Moq (Mock<T>, .Setup(), .Returns())",
		"test_attribute": "[Fact] / [Theory]",
	}
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
Write xUnit test methods for the class or method named ` + "`{{.MemberName}}`" + `.

Source:
` + "```csharp\n{{.SourceCode}}\n```" + `

Requirements:
- Use xUnit [Fact] for single-case tests and [Theory] + [InlineData] for parameterised tests.
- Use Moq for mocking: ` + "`new Mock<IDependency>()`" + `, ` + "`.Setup()`, `.Returns()`" + `.
- Use ` + "`Assert.Equal`, `Assert.True`, `Assert.Throws<T>`" + ` for assertions.
- Follow Arrange / Act / Assert comment structure.
- Output ONLY valid C# code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return csMigrators }
