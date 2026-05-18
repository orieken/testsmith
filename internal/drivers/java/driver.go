// Package java implements the LanguageDriver for Java projects.
// Supports Maven (pom.xml) and Gradle (build.gradle) build systems.
// JUnit 5 + Mockito test generation.
package java

import (
	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for Java + JUnit 5.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string            { return "java" }
func (d *Driver) FileExtensions() []string    { return []string{".java"} }
func (d *Driver) BodyGenerationPrompt() string { return javaBodyPrompt }
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "java"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit5",
		TestFileSuffix: "Test.java",
		FixtureDir:     "",
		BootstrapFile:  "",
		TestFuncPrefix: "@Test",
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
	return nil, nil // Java uses Mockito constructor injection, not shared fixture files.
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

const javaBodyPrompt = `You are an expert Java testing assistant.
Write a comprehensive {{index .Extra "framework"}} test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + `.

Source:
` + "```java\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this class:
` + "```java\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Test annotation: {{index .Extra "test_annotation"}}
Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}

Requirements:
- Include a happy-path test and at least one edge-case.
- Follow Arrange / Act / Assert structure with comments.
- Use the mock and assert styles shown above.
- Output ONLY valid Java code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return javaMigrators }
