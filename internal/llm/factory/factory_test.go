package factory_test

import (
	"os"
	"testing"

	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/llm/factory"
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

// ---- BuildProvider tests -------------------------------------------------------

// backfill / AC: BuildProvider returns a distinct error when LLM is disabled,
// unlike Build which returns nil,nil — callers of BuildProvider always need a live LLM.
func TestBuildProvider_DisabledReturnsError(t *testing.T) {
	t.Parallel()
	_, err := factory.BuildProvider(config.LLMConfig{Enabled: false})
	if err == nil {
		t.Error("expected error when LLM is disabled; got nil")
	}
}

// backfill / AC: BuildProvider returns error when non-ollama provider has no API key env var set.
// Uses a unique env var name (TESTSMITH_NO_SUCH_KEY_XYZ) that is guaranteed absent in any
// normal environment, so no explicit unset is required and the test is safe to run in parallel.
func TestBuildProvider_MissingAPIKey_ReturnsError(t *testing.T) {
	t.Parallel()
	cfg := config.LLMConfig{
		Enabled:      true,
		Provider:     "anthropic",
		APIKeyEnvVar: "TESTSMITH_NO_SUCH_KEY_XYZ",
	}
	_, err := factory.BuildProvider(cfg)
	if err == nil {
		t.Error("expected error when API key env var is unset; got nil")
	}
}

// backfill / AC: BuildProvider returns a non-nil provider for ollama, which needs no API key.
func TestBuildProvider_OllamaRequiresNoKey(t *testing.T) {
	t.Parallel()
	cfg := config.LLMConfig{
		Enabled:          true,
		Provider:         "ollama",
		MaxRetryAttempts: 1,
	}
	p, err := factory.BuildProvider(cfg)
	if err != nil {
		t.Fatalf("ollama should not require an API key: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider for ollama")
	}
}

// backfill / AC: BuildProvider returns a non-nil provider when the API key env var is set.
// t.Setenv restores the env automatically after the test; no t.Parallel() per project rules.
func TestBuildProvider_AnthropicWithKey_ReturnsProvider(t *testing.T) {
	t.Setenv("TESTSMITH_ANTHROPIC_PROVIDER_KEY", "test-key")
	cfg := config.LLMConfig{
		Enabled:          true,
		Provider:         "anthropic",
		APIKeyEnvVar:     "TESTSMITH_ANTHROPIC_PROVIDER_KEY",
		MaxRetryAttempts: 1,
	}
	p, err := factory.BuildProvider(cfg)
	if err != nil {
		t.Fatalf("unexpected error building anthropic provider: %v", err)
	}
	if p == nil {
		t.Error("expected non-nil provider")
	}
}

// backfill / AC: BuildProvider returns error for an unknown provider name,
// even when an API key env var is present (key presence check passes; switch hits default).
// t.Setenv restores the env automatically after the test; no t.Parallel() per project rules.
func TestBuildProvider_UnknownProvider_ReturnsError(t *testing.T) {
	t.Setenv("TESTSMITH_UNKNOWN_PROVIDER_KEY", "value")
	cfg := config.LLMConfig{
		Enabled:      true,
		Provider:     "unknown-llm",
		APIKeyEnvVar: "TESTSMITH_UNKNOWN_PROVIDER_KEY",
	}
	_, err := factory.BuildProvider(cfg)
	if err == nil {
		t.Error("expected error for unknown provider; got nil")
	}
}

// ---- stubDriver ---------------------------------------------------------------

// stubDriver satisfies domain.LanguageDriver minimally for factory tests.
type stubDriver struct{}

func (s *stubDriver) Language() string             { return "python" }
func (s *stubDriver) BodyGenerationPrompt() string { return "generate tests for {{.MemberName}}" }
func (s *stubDriver) FileExtensions() []string     { return []string{".py"} }
func (s *stubDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{}
}
func (s *stubDriver) LLMContext(_ *domain.ProjectContext) map[string]string    { return nil }
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
func (s *stubDriver) ListMigrators() []domain.Migrator                     { return nil }
func (s *stubDriver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }
