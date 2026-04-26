// Package golang implements the LanguageDriver for Go projects.
// Uses the stdlib go/ast and go/parser packages — no tree-sitter dependency.
// Phase 5 implementation target.
package golang

import (
	"errors"

	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for Go + testing package.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "go" }
func (d *Driver) FileExtensions() []string { return []string{".go"} }
func (d *Driver) BodyGenerationPrompt() string { return goBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "t.Errorf / testify/assert",
		"framework":      "testing",
		"test_prefix":    "Test",
	}
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
	return nil, errors.New("go: not yet implemented — Phase 5")
}

func (d *Driver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, errors.New("go: not yet implemented — Phase 5")
}

func (d *Driver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}

func (d *Driver) DeriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("go: not yet implemented — Phase 5")
}

func (d *Driver) DeriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("go: not yet implemented — Phase 5")
}

func (d *Driver) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, errors.New("go: not yet implemented — Phase 5")
}

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil // Go uses interface mocks, not shared fixture files.
}

func (d *Driver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil // Go has no bootstrap file equivalent.
}

const goBodyPrompt = `You are an expert Go testing assistant.
Write table-driven tests for the function named ` + "`{{.MemberName}}`" + ` using the standard ` + "`testing`" + ` package.

Source:
` + "```go\n{{.SourceCode}}\n```" + `

Output ONLY valid Go code in a single markdown code block.`
