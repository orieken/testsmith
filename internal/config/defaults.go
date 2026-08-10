package config

import "github.com/orieken/assay/internal/domain"

// Default returns a Config pre-populated with sensible defaults.
func Default() *Config {
	return &Config{
		TestRoot:   "tests/",
		FixtureDir: "tests/fixtures/",
		ExcludeDirs: []string{
			"node_modules", ".venv", "venv", "__pycache__",
			".git", "build", "dist", ".tox", ".eggs", "vendor", "target",
		},
		LLM: LLMConfig{
			Enabled:              false,
			Provider:             "anthropic",
			Model:                "claude-sonnet-4-6",
			MaxTokensPerFunction: 1500,
			Temperature:          0.0,
			APIKeyEnvVar:         "ANTHROPIC_API_KEY",
			PromptTokenBudget:    6000, // conservative ceiling; fits most model context windows
			MaxConcurrentCalls:   5,    // safe default; raise for high-throughput runs
			MaxRetryAttempts:     3,    // first attempt + 2 retries with exponential backoff
		},
		Languages: map[string]LanguageConfig{
			"python": {
				TestRoot:      "tests/",
				FixtureDir:    "tests/fixtures/",
				FixtureSuffix: "_fixture.py",
				// Defaults; overridden by auto-detection or explicit config.
				Framework:   "pytest",
				MockLibrary: "pytest-mock",
				Extra: map[string]string{
					"conftest_path":    "conftest.py",
					"paths_to_add_var": "paths_to_add",
				},
			},
			"typescript": {
				TestRoot:    "src/",
				FixtureDir:  "__mocks__/",
				Framework:   "jest", // auto-detection may upgrade to "vitest"
				MockLibrary: "jest",
				Extra: map[string]string{
					"test_file_suffix": ".test.ts",
				},
			},
			"go": {
				FixtureDir:  "",
				Framework:   "testing",
				MockLibrary: "interfaces",
			},
			"java": {
				TestRoot:    "src/test/java/",
				Framework:   "junit5",
				MockLibrary: "mockito",
			},
			"csharp": {
				Framework:   "xunit",
				MockLibrary: "moq",
			},
		},
	}
}

// ApplyToContext copies framework/mock_library config overrides into ctx.Metadata.
// Driver auto-detection runs first (inside DetectProject); this runs after, so
// explicit config always wins over auto-detected values.
func ApplyToContext(cfg *Config, ctx *domain.ProjectContext) {
	if ctx == nil || ctx.Metadata == nil {
		return
	}
	langCfg, ok := cfg.Languages[ctx.Language]
	if !ok {
		return
	}
	if langCfg.Framework != "" {
		ctx.Metadata["framework"] = langCfg.Framework
	}
	if langCfg.MockLibrary != "" {
		ctx.Metadata["mock_library"] = langCfg.MockLibrary
	}
}
