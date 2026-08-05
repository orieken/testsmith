package config_test

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/orieken/testsmith/internal/config"
)

// TestLoadFromFile_SnakeCaseFields verifies that snake_case YAML keys produced
// by `testsmith init` round-trip correctly into the Config struct.
func TestLoadFromFile_SnakeCaseFields(t *testing.T) {
	content := `
language: python
test_root: tests/
fixture_dir: tests/fixtures/
exclude_dirs:
  - node_modules
  - .venv
llm:
  enabled: false
  provider: anthropic
  model: claude-sonnet-4-6
  max_tokens_per_function: 1500
  api_key_env_var: ANTHROPIC_API_KEY
`
	tmp := t.TempDir() + "/init.yaml"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(tmp)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if cfg.Language != "python" {
		t.Errorf("Language = %q, want python", cfg.Language)
	}
	if cfg.TestRoot != "tests/" {
		t.Errorf("TestRoot = %q, want tests/", cfg.TestRoot)
	}
	if cfg.FixtureDir != "tests/fixtures/" {
		t.Errorf("FixtureDir = %q, want tests/fixtures/", cfg.FixtureDir)
	}
	if len(cfg.ExcludeDirs) < 2 {
		t.Errorf("ExcludeDirs = %v, want at least [node_modules .venv]", cfg.ExcludeDirs)
	}
	if cfg.LLM.Provider != "anthropic" {
		t.Errorf("LLM.Provider = %q, want anthropic", cfg.LLM.Provider)
	}
	if cfg.LLM.MaxTokensPerFunction != 1500 {
		t.Errorf("LLM.MaxTokensPerFunction = %d, want 1500", cfg.LLM.MaxTokensPerFunction)
	}
	if cfg.LLM.APIKeyEnvVar != "ANTHROPIC_API_KEY" {
		t.Errorf("LLM.APIKeyEnvVar = %q, want ANTHROPIC_API_KEY", cfg.LLM.APIKeyEnvVar)
	}
}

// TestLoadFromFile_LanguageConfig verifies that per-language overrides using
// snake_case keys parse correctly.
func TestLoadFromFile_LanguageConfig(t *testing.T) {
	content := `
languages:
  python:
    test_root: tests/
    fixture_dir: tests/fixtures/
    fixture_suffix: _fixture.py
    framework: pytest
    mock_library: pytest-mock
`
	tmp := t.TempDir() + "/lang.yaml"
	if err := os.WriteFile(tmp, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := config.LoadFromFile(tmp)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	py, ok := cfg.Languages["python"]
	if !ok {
		t.Fatal("no python language config")
	}
	if py.TestRoot != "tests/" {
		t.Errorf("py.TestRoot = %q, want tests/", py.TestRoot)
	}
	if py.MockLibrary != "pytest-mock" {
		t.Errorf("py.MockLibrary = %q, want pytest-mock", py.MockLibrary)
	}
	if py.FixtureSuffix != "_fixture.py" {
		t.Errorf("py.FixtureSuffix = %q, want _fixture.py", py.FixtureSuffix)
	}
}

func TestLoadFromFile_NonExistentPath_ReturnsError(t *testing.T) {
	_, err := config.LoadFromFile("/nonexistent/path/that/will/never/exist.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestLoadFromFile_InvalidYAML_ReturnsError(t *testing.T) {
	tmp := t.TempDir() + "/bad.yaml"
	if err := os.WriteFile(tmp, []byte(": this is not valid yaml: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := config.LoadFromFile(tmp)
	if err == nil {
		t.Error("expected error for invalid YAML, got nil")
	}
}

// TestInitConfigRoundTrip verifies that yaml.Marshal on a Config struct
// produces snake_case keys that LoadFromFile can read back.
func TestInitConfigRoundTrip(t *testing.T) {
	original := &config.Config{
		Language:    "go",
		TestRoot:    "tests/",
		FixtureDir:  "tests/fixtures/",
		ExcludeDirs: []string{"vendor", ".git"},
		LLM: config.LLMConfig{
			Provider:             "anthropic",
			Model:                "claude-sonnet-4-6",
			MaxTokensPerFunction: 800,
			APIKeyEnvVar:         "ANTHROPIC_API_KEY",
		},
	}

	data, err := yaml.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	tmp := t.TempDir() + "/roundtrip.yaml"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := config.LoadFromFile(tmp)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}

	if loaded.Language != original.Language {
		t.Errorf("Language = %q, want %q", loaded.Language, original.Language)
	}
	if loaded.TestRoot != original.TestRoot {
		t.Errorf("TestRoot = %q, want %q", loaded.TestRoot, original.TestRoot)
	}
	if loaded.LLM.MaxTokensPerFunction != original.LLM.MaxTokensPerFunction {
		t.Errorf("LLM.MaxTokensPerFunction = %d, want %d", loaded.LLM.MaxTokensPerFunction, original.LLM.MaxTokensPerFunction)
	}
	if loaded.LLM.APIKeyEnvVar != original.LLM.APIKeyEnvVar {
		t.Errorf("LLM.APIKeyEnvVar = %q, want %q", loaded.LLM.APIKeyEnvVar, original.LLM.APIKeyEnvVar)
	}
}
