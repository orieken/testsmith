// Package typescript implements the LanguageDriver for TypeScript/JavaScript projects.
// Supports Jest and Vitest test frameworks (auto-detected from package.json).
// Phase 4 implementation target.
package typescript

import (
	"errors"

	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for TypeScript/JavaScript + Jest/Vitest.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "typescript" }
func (d *Driver) FileExtensions() []string { return []string{".ts", ".tsx", ".js", ".jsx"} }
func (d *Driver) BodyGenerationPrompt() string { return typescriptBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "expect(...).toBe / toEqual",
		"framework":      "Jest",
		"mock_library":   "jest.mock / vi.mock",
	}
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
	return nil, errors.New("typescript: not yet implemented — Phase 4")
}

func (d *Driver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, errors.New("typescript: not yet implemented — Phase 4")
}

func (d *Driver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}

func (d *Driver) DeriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("typescript: not yet implemented — Phase 4")
}

func (d *Driver) DeriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("typescript: not yet implemented — Phase 4")
}

func (d *Driver) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, errors.New("typescript: not yet implemented — Phase 4")
}

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}

func (d *Driver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

const typescriptBodyPrompt = `You are an expert TypeScript testing assistant.
Write Jest test bodies for the {{.MemberKind}} named ` + "`{{.MemberName}}`" + `.

Source:
` + "```typescript\n{{.SourceCode}}\n```" + `

Output ONLY valid TypeScript code in a single markdown code block.`
