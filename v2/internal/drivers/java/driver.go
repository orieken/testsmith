// Package java implements the LanguageDriver for Java projects.
// Supports Maven (pom.xml) and Gradle (build.gradle) build systems.
// JUnit 5 + Mockito test generation.
// Phase 6 implementation target.
package java

import (
	"errors"

	"github.com/orieken/testsmith/internal/domain"
)

// Driver implements domain.LanguageDriver for Java + JUnit 5.
type Driver struct{}

func New() *Driver { return &Driver{} }

func (d *Driver) Language() string         { return "java" }
func (d *Driver) FileExtensions() []string { return []string{".java"} }
func (d *Driver) BodyGenerationPrompt() string { return javaBodyPrompt }
func (d *Driver) LLMContext() map[string]string {
	return map[string]string{
		"assert_keyword": "Assertions.assertEquals / assertThat",
		"framework":      "JUnit 5",
		"mock_library":   "Mockito",
	}
}

func (d *Driver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit5",
		TestFilePrefix: "Test",
		TestFileSuffix: ".java",
		FixtureDir:     "",
		BootstrapFile:  "",
		TestFuncPrefix: "@Test",
	}
}

func (d *Driver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return nil, errors.New("java: not yet implemented — Phase 6")
}

func (d *Driver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, errors.New("java: not yet implemented — Phase 6")
}

func (d *Driver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}

func (d *Driver) DeriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("java: not yet implemented — Phase 6")
}

func (d *Driver) DeriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	return "", errors.New("java: not yet implemented — Phase 6")
}

func (d *Driver) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, errors.New("java: not yet implemented — Phase 6")
}

func (d *Driver) GenerateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}

func (d *Driver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}

const javaBodyPrompt = `You are an expert Java testing assistant.
Write JUnit 5 test methods for the class or method named ` + "`{{.MemberName}}`" + `.

Source:
` + "```java\n{{.SourceCode}}\n```" + `

Output ONLY valid Java code in a single markdown code block.`
