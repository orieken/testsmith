// Package typescript implements the LanguageDriver for TypeScript/JavaScript projects.
// Supports Jest and Vitest test frameworks (auto-detected from package.json).
package typescript

import (
	"github.com/orieken/assay/internal/domain"
)

// Driver implements domain.LanguageDriver for TypeScript/JavaScript + Jest/Vitest.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string             { return "typescript" }
func (d *Driver) FileExtensions() []string     { return []string{".ts", ".tsx", ".js", ".jsx"} }
func (d *Driver) BodyGenerationPrompt() string { return typescriptBodyPrompt }
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
	vocab := registry.SelectFromContext(ctx).LLMVocabulary()
	vocab["language"] = "typescript"
	return vocab
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "jest",
		TestFileSuffix: ".test.ts",
		FixtureDir:     "__mocks__/",
		BootstrapFile:  "jest.setup.ts",
		TestFuncPrefix: "it(",
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

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return generateMock(dep, analysis, opts)
}

func (d *Driver) GenerateBootstrap(_ *domain.GenerationPlan, _ *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

const typescriptBodyPrompt = `You are an expert TypeScript testing assistant.
Write a comprehensive {{index .Extra "framework"}} test body for the {{.MemberKind}} ` + "`{{.MemberName}}`" + ` in module ` + "`{{.ModulePath}}`" + `.

Source:
` + "```typescript\n{{.SourceCode}}\n```" + `
{{- if .DepsSignatures}}

Internal dependency signatures:
{{.DepsSignatures}}
{{- end}}
{{- if .ExistingTestSnippet}}

Follow the style of existing tests in this module:
` + "```typescript\n{{.ExistingTestSnippet}}\n```" + `
{{- end}}

Import style: {{index .Extra "import_style"}}
Mock style: {{index .Extra "mock_style"}}
Assert style: {{index .Extra "assert_style"}}

Requirements:
- Include a happy-path test and at least one edge-case.
- Use the import and mock styles shown above.
- Output ONLY valid TypeScript code in a single markdown code block.`

func (d *Driver) ListAdapters(ctx *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return registry.All(), selectAdapter(ctx)
}

func (d *Driver) ListMigrators() []domain.Migrator { return tsMigrators }
