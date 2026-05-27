// Package python implements the LanguageDriver for Python projects using pytest.
// AST parsing is performed via go-tree-sitter with the Python grammar.
// This file contains the top-level driver struct and interface wiring;
// implementation details live in the sibling files:
//   - detector.go   — project root detection + package map scan
//   - analyzer.go   — tree-sitter import + public API extraction
//   - classifier.go — stdlib detection + import classification
//   - generator.go  — test file, fixture, and conftest generation
package python

import (
	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for Python + pytest.
type Driver struct{}

// New returns a ready-to-use Python Driver.
func New() *Driver { return &Driver{} }

func (d *Driver) Language() string             { return "python" }
func (d *Driver) FileExtensions() []string     { return []string{".py"} }
func (d *Driver) BodyGenerationPrompt() string { return pythonBodyPrompt }
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "python"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "pytest",
		TestFilePrefix: "test_",
		TestFileSuffix: ".py",
		FixtureDir:     "tests/fixtures/",
		FixtureSuffix:  "_fixture.py",
		BootstrapFile:  "conftest.py",
		TestFuncPrefix: "test_",
	}
}

// DetectProject, AnalyzeFile, ClassifyDependency, DeriveTestPath,
// DeriveModulePath, GenerateTestFile, GenerateFixture, GenerateBootstrap
// are implemented in detector.go, analyzer.go, classifier.go, and generator.go.
// Stubs below are replaced as each file is implemented.

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

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return generateFixture(dep, analysis, opts)
}

func (d *Driver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return generateBootstrap(plan, ctx)
}

const pythonBodyPrompt = `You are an expert Python testing assistant.
Write a comprehensive {{index .Extra "framework"}} test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in module ` + "`{{.ModulePath}}`" + `.

Source:
` + "```python\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this module:
` + "```python\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Available fixtures: {{range .FixtureNames}}{{.}} {{end}}
Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}

Requirements:
- Include a happy-path test.
- Include at least one edge-case or error test.
- Use the mock and assert styles shown above.
- Output ONLY valid Python code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return pyMigrators }
