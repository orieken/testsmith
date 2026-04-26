// Package csharp implements the LanguageDriver for C# projects.
// Supports MSBuild project structures (csproj/sln) and auto-detects
// between xUnit (default), NUnit, and MSTest test frameworks.
// AST parsing is performed via go-tree-sitter with the C# grammar.
// Phase 6 implementation target (alongside Java).
package csharp

import (
	"errors"

	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for C# + xUnit/NUnit/MSTest.
type Driver struct{}

// New returns a ready-to-use C# Driver.
func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "csharp" }
func (d *Driver) FileExtensions() []string { return []string{".cs"} }
func (d *Driver) BodyGenerationPrompt() string { return csharpBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "Assert.Equal / Assert.True (xUnit) or Assert.That (NUnit)",
		"framework":      "xUnit",
		"mock_library":   "Moq (Mock<T>, .Setup(), .Returns())",
		"test_attribute": "[Fact] / [Theory]",
	}
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "xunit",
		TestFilePrefix: "",
		TestFileSuffix: "Tests.cs",
		FixtureDir:     "",   // C# uses constructor injection, not shared fixture files
		FixtureSuffix:  "",
		BootstrapFile:  "",   // no conftest equivalent; test discovery is assembly-based
		TestFuncPrefix: "[Fact]",
	}
}

// DetectProject, AnalyzeFile, ClassifyDependency, DeriveTestPath,
// DeriveModulePath, GenerateTestFile, GenerateFixture, GenerateBootstrap
// are implemented in detector.go, analyzer.go, classifier.go, and generator.go.
// Stubs below are replaced as each file is implemented in Phase 6.

func (d *Driver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return nil, errors.New("csharp: not yet implemented — Phase 6")
}

func (d *Driver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, errors.New("csharp: not yet implemented — Phase 6")
}

func (d *Driver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}

func (d *Driver) DeriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("csharp: not yet implemented — Phase 6")
}

func (d *Driver) DeriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("csharp: not yet implemented — Phase 6")
}

func (d *Driver) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, errors.New("csharp: not yet implemented — Phase 6")
}

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	// C# uses Moq constructor injection rather than shared fixture files.
	// Mocks are declared per test class, not in a shared directory.
	return nil, nil
}

func (d *Driver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	// C# test discovery is assembly-based; no bootstrap file needed.
	return nil, nil
}

const csharpBodyPrompt = `You are an expert C# testing assistant.
Write xUnit test methods for the class or method named ` + "`{{.MemberName}}`" + `.

Source:
` + "```csharp\n{{.SourceCode}}\n```" + `

Requirements:
- Use xUnit [Fact] for single-case tests and [Theory] + [InlineData] for parameterised tests.
- Use Moq for mocking dependencies: ` + "`new Mock<IDependency>()`" + `, ` + "`.Setup()`, `.Returns()`" + `.
- Use ` + "`Assert.Equal`, `Assert.True`, `Assert.Throws<T>`" + ` for assertions.
- Follow Arrange / Act / Assert comment structure.
- Output ONLY valid C# code in a single markdown code block.`
