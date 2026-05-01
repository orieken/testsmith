// Package config handles loading and merging of TestSmith configuration.
package config

// Config is the fully-resolved configuration for a TestSmith invocation.
type Config struct {
	Language    string            `yaml:"language,omitempty"`
	Root        string            `yaml:"root,omitempty"`
	TestRoot    string            `yaml:"test_root,omitempty"`
	FixtureDir  string            `yaml:"fixture_dir,omitempty"`
	ExcludeDirs []string          `yaml:"exclude_dirs,omitempty"`
	LLM         LLMConfig         `yaml:"llm,omitempty"`
	Languages   map[string]LanguageConfig `yaml:"languages,omitempty"`
	Workspaces  []WorkspaceConfig `yaml:"workspaces,omitempty"`

	// ConfigPath is set by the loader to the absolute path of the file that
	// was read. Empty when only defaults are in effect (no file found).
	ConfigPath string `yaml:"-"`
}

// LLMConfig configures the optional LLM body-generation adapter.
type LLMConfig struct {
	Enabled              bool    `yaml:"enabled,omitempty"`
	Provider             string  `yaml:"provider,omitempty"`
	Model                string  `yaml:"model,omitempty"`
	MaxTokensPerFunction int     `yaml:"max_tokens_per_function,omitempty"`
	Temperature          float64 `yaml:"temperature,omitempty"`
	APIKeyEnvVar         string  `yaml:"api_key_env_var,omitempty"`
	BaseURL              string  `yaml:"base_url,omitempty"`
}

// LanguageConfig holds per-language overrides that are merged over the root Config.
type LanguageConfig struct {
	TestRoot      string            `yaml:"test_root,omitempty"`
	FixtureDir    string            `yaml:"fixture_dir,omitempty"`
	FixtureSuffix string            `yaml:"fixture_suffix,omitempty"`
	Framework     string            `yaml:"framework,omitempty"`
	MockLibrary   string            `yaml:"mock_library,omitempty"`
	Extra         map[string]string `yaml:"extra,omitempty"`
}

// WorkspaceConfig describes one workspace in a monorepo.
type WorkspaceConfig struct {
	Name     string     `yaml:"name,omitempty"`
	Path     string     `yaml:"path,omitempty"`
	Language string     `yaml:"language,omitempty"`
	LLM      *LLMConfig `yaml:"llm,omitempty"`
}

// WorkspaceLLM returns the effective LLM config for a workspace: the
// workspace-level override when set, otherwise the root config's LLM.
func WorkspaceLLM(root LLMConfig, ws *WorkspaceConfig) LLMConfig {
	if ws.LLM != nil {
		return *ws.LLM
	}
	return root
}

// WorkspaceID returns the display name for a workspace (Name if set, else Path).
func WorkspaceID(ws *WorkspaceConfig) string {
	if ws.Name != "" {
		return ws.Name
	}
	return ws.Path
}
