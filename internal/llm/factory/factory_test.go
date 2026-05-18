package factory_test

import (
	"os"
	"testing"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/llm/factory"
)

func TestBuild_DisabledReturnsNil(t *testing.T) {
	gen, err := factory.Build(config.LLMConfig{Enabled: false}, &stubDriver{})
	if err != nil || gen != nil {
		t.Errorf("disabled LLM should return nil, nil; got gen=%v err=%v", gen, err)
	}
}

func TestBuild_MissingAPIKey_ReturnsError(t *testing.T) {
	os.Unsetenv("TEST_API_KEY")
	cfg := config.LLMConfig{Enabled: true, Provider: "anthropic", APIKeyEnvVar: "TEST_API_KEY"}
	_, err := factory.Build(cfg, &stubDriver{})
	if err == nil {
		t.Error("expected error when API key env var is missing")
	}
}

func TestBuild_OllamaRequiresNoKey(t *testing.T) {
	cfg := config.LLMConfig{
		Enabled:              true,
		Provider:             "ollama",
		Model:                "llama3",
		MaxTokensPerFunction: 500,
	}
	gen, err := factory.Build(cfg, &stubDriver{})
	if err != nil {
		t.Fatalf("ollama should not require an API key: %v", err)
	}
	if gen == nil {
		t.Error("expected non-nil generator for ollama")
	}
}

func TestBuild_UnknownProvider_ReturnsError(t *testing.T) {
	t.Setenv("SOME_KEY", "value")
	cfg := config.LLMConfig{Enabled: true, Provider: "unknown-llm", APIKeyEnvVar: "SOME_KEY"}
	_, err := factory.Build(cfg, &stubDriver{})
	if err == nil {
		t.Error("expected error for unknown provider")
	}
}

func TestBuild_AnthropicWithKey_ReturnsGenerator(t *testing.T) {
	t.Setenv("ANTHROPIC_API_KEY", "test-key")
	cfg := config.LLMConfig{
		Enabled:              true,
		Provider:             "anthropic",
		Model:                "claude-sonnet-4-6",
		MaxTokensPerFunction: 1500,
		APIKeyEnvVar:         "ANTHROPIC_API_KEY",
	}
	gen, err := factory.Build(cfg, &stubDriver{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gen == nil {
		t.Error("expected non-nil generator")
	}
}

// stubDriver satisfies domain.LanguageDriver minimally for factory tests.
type stubDriver struct{}

func (s *stubDriver) Language() string                    { return "python" }
func (s *stubDriver) BodyGenerationPrompt() string        { return "generate tests for {{.MemberName}}" }
func (s *stubDriver) FileExtensions() []string            { return []string{".py"} }
func (s *stubDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig { return domain.TestFrameworkConfig{} }
func (s *stubDriver) LLMContext(_ *domain.ProjectContext) map[string]string       { return nil }
func (s *stubDriver) DetectProject(dir string) (*domain.ProjectContext, error) { return nil, nil }
func (s *stubDriver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, nil
}
func (s *stubDriver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}
func (s *stubDriver) DeriveTestPath(src string, ctx *domain.ProjectContext) (string, error) {
	return "", nil
}
func (s *stubDriver) DeriveModulePath(src string, ctx *domain.ProjectContext) (string, error) {
	return "", nil
}
func (s *stubDriver) GenerateTestFile(a *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) GenerateFixture(dep string, a *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) ListAdapters(_ *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return nil, nil
}
func (s *stubDriver) ListMigrators() []domain.Migrator { return nil }
func (s *stubDriver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }
