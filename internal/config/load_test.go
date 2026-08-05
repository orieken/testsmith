package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
)

// ── Load ──────────────────────────────────────────────────────────────────────

func TestLoad_NoConfigFile_ReturnsDefaults(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg == nil {
		t.Fatal("Load returned nil config")
	}
	// Verify a default value is intact.
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("default LLM.Provider = %q, want anthropic", cfg.LLM.Provider)
	}
	if cfg.ConfigPath != "" {
		t.Errorf("ConfigPath should be empty when no file found, got %q", cfg.ConfigPath)
	}
}

func TestLoad_FindsFileInCurrentDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	content := "language: python\ntest_root: mytests/\n"
	cfgPath := filepath.Join(dir, ".testsmith.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Language != "python" {
		t.Errorf("Language = %q, want python", cfg.Language)
	}
	if cfg.ConfigPath != cfgPath {
		t.Errorf("ConfigPath = %q, want %q", cfg.ConfigPath, cfgPath)
	}
}

func TestLoad_FindsFileInParentDir(t *testing.T) {
	t.Parallel()
	parent := t.TempDir()
	child := filepath.Join(parent, "sub", "pkg")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(parent, ".testsmith.yaml")
	if err := os.WriteFile(cfgPath, []byte("language: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.Load(child)
	if err != nil {
		t.Fatalf("Load from child: %v", err)
	}
	if cfg.Language != "go" {
		t.Errorf("Language = %q, want go (loaded from parent)", cfg.Language)
	}
	if cfg.ConfigPath != cfgPath {
		t.Errorf("ConfigPath = %q, want parent config %q", cfg.ConfigPath, cfgPath)
	}
}

func TestLoad_InvalidYAML_ReturnsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".testsmith.yaml"), []byte(":\tinvalid:\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := config.Load(dir); err == nil {
		t.Error("Load with invalid YAML should return error")
	}
}

// ── ApplyToContext ────────────────────────────────────────────────────────────

func TestApplyToContext_AppliesFrameworkAndMockLibrary(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Languages: map[string]config.LanguageConfig{
			"python": {Framework: "pytest", MockLibrary: "pytest-mock"},
		},
	}
	ctx := &domain.ProjectContext{
		Language: "python",
		Metadata: map[string]any{},
	}
	config.ApplyToContext(cfg, ctx)

	if got, _ := ctx.Metadata["framework"].(string); got != "pytest" {
		t.Errorf("framework = %q, want pytest", got)
	}
	if got, _ := ctx.Metadata["mock_library"].(string); got != "pytest-mock" {
		t.Errorf("mock_library = %q, want pytest-mock", got)
	}
}

func TestApplyToContext_NilContext_DoesNotPanic(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Languages: map[string]config.LanguageConfig{"go": {Framework: "testing"}}}
	config.ApplyToContext(cfg, nil) // must not panic
}

func TestApplyToContext_NilMetadata_DoesNotPanic(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Languages: map[string]config.LanguageConfig{"go": {Framework: "testing"}}}
	ctx := &domain.ProjectContext{Language: "go", Metadata: nil}
	config.ApplyToContext(cfg, ctx) // must not panic
}

func TestApplyToContext_UnknownLanguage_NoChange(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{Languages: map[string]config.LanguageConfig{}}
	ctx := &domain.ProjectContext{
		Language: "rust",
		Metadata: map[string]any{"framework": "original"},
	}
	config.ApplyToContext(cfg, ctx)
	if got, _ := ctx.Metadata["framework"].(string); got != "original" {
		t.Errorf("framework should be unchanged for unknown language, got %q", got)
	}
}

func TestApplyToContext_EmptyFramework_DoesNotOverride(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{
		Languages: map[string]config.LanguageConfig{
			"go": {Framework: "", MockLibrary: ""},
		},
	}
	ctx := &domain.ProjectContext{
		Language: "go",
		Metadata: map[string]any{"framework": "testing"},
	}
	config.ApplyToContext(cfg, ctx)
	// Empty framework in config must not wipe the existing value.
	if got, _ := ctx.Metadata["framework"].(string); got != "testing" {
		t.Errorf("framework overridden to empty — want testing, got %q", got)
	}
}
