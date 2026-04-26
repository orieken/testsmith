// Package config handles loading and merging of TestSmith configuration.
package config

// Config is the fully-resolved configuration for a TestSmith invocation.
type Config struct {
	Language    string
	Root        string
	TestRoot    string
	FixtureDir  string
	ExcludeDirs []string
	LLM         LLMConfig
	Languages   map[string]LanguageConfig
	Workspaces  []WorkspaceConfig
}

// LLMConfig configures the optional LLM body-generation adapter.
type LLMConfig struct {
	Enabled              bool
	Provider             string
	Model                string
	MaxTokensPerFunction int
	Temperature          float64
	APIKeyEnvVar         string
	BaseURL              string
}

// LanguageConfig holds per-language overrides that are merged over the root Config.
type LanguageConfig struct {
	TestRoot      string
	FixtureDir    string
	FixtureSuffix string
	// Extra holds driver-specific keys not represented in the common fields.
	Extra map[string]string
}

// WorkspaceConfig describes one workspace in a monorepo.
type WorkspaceConfig struct {
	Path     string
	Language string
	LLM      *LLMConfig // nil means inherit from root Config
}
