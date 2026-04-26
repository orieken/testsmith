package config

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
		},
		Languages: map[string]LanguageConfig{
			"python": {
				TestRoot:      "tests/",
				FixtureDir:    "tests/fixtures/",
				FixtureSuffix: "_fixture.py",
				Extra: map[string]string{
					"conftest_path":      "conftest.py",
					"paths_to_add_var":   "paths_to_add",
				},
			},
			"typescript": {
				TestRoot:   "src/",
				FixtureDir: "__mocks__/",
				Extra: map[string]string{
					"test_file_suffix": ".test.ts",
				},
			},
			"go": {
				FixtureDir: "",
			},
			"java": {
				TestRoot: "src/test/java/",
			},
		},
	}
}
