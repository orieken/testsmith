# Feature: Configuration

## What It Does

TestSmith v2 works with **zero configuration** — it auto-detects the project language, root, and conventions. When the defaults don't match your project layout, a `.testsmith.yaml` file at the project root provides overrides.

For Python projects that already use TestSmith v1, the `[tool.testsmith]` section in `pyproject.toml` is read as a fallback, preserving backward compatibility.

---

## `.testsmith.yaml` Full Schema

```yaml
# .testsmith.yaml

# Override auto-detected language. Leave empty for auto-detection.
language: ""   # python | typescript | go | java | csharp

# Override project root. Default: directory containing this file.
root: ""

# --- Language-agnostic defaults ---
test_root: "tests/"
fixture_dir: "tests/fixtures/"
exclude_dirs:
  - node_modules
  - .venv
  - venv
  - __pycache__
  - .git
  - build
  - dist
  - .tox
  - vendor
  - target
  - .eggs

# --- LLM configuration ---
llm:
  enabled: false
  provider: "anthropic"       # anthropic | openai | ollama
  model: "claude-sonnet-4-6"
  max_tokens_per_function: 1500
  temperature: 0.0
  api_key_env_var: "ANTHROPIC_API_KEY"
  base_url: ""                # set for Ollama or OpenAI-compatible endpoints

# --- Per-language overrides ---
languages:
  python:
    test_root: "tests/"
    fixture_dir: "tests/fixtures/"
    fixture_suffix: "_fixture.py"
    framework: "pytest"
    mock_library: "pytest-mock"

  typescript:
    test_root: "src/"
    fixture_dir: "__mocks__/"
    framework: "jest"         # jest | vitest (auto-detected if omitted)
    mock_library: "jest"

  go:
    framework: "testing"
    mock_library: "interfaces"

  java:
    test_root: "src/test/java/"
    framework: "junit5"
    mock_library: "mockito"

  csharp:
    framework: "xunit"
    mock_library: "moq"

# --- Monorepo / workspace support ---
# Each workspace entry is resolved relative to the config file's directory.
# All commands that support --workspace use these entries.
workspaces:
  - name: api           # human-readable identifier for --workspace flag
    path: services/api
    language: go
  - name: frontend
    path: services/frontend
    language: typescript
    llm:                # optional per-workspace LLM override
      provider: openai
      model: gpt-4o
```

---

## Loading Precedence

```
CLI flags  >  .testsmith.yaml  >  pyproject.toml [tool.testsmith]  >  driver defaults
```

---

## YAML Key Conventions

All multi-word keys use `snake_case`. The Go structs carry explicit `yaml:"snake_case"` tags so marshalling a `*config.Config` struct (e.g. from `testsmith init`) always produces keys that `config.LoadFromFile` can round-trip:

```go
data, _ := yaml.Marshal(cfg)            // writes  test_root: tests/
loaded, _ := config.LoadFromFile(path)  // loaded.TestRoot == "tests/"  ✓
```

---

## Go Implementation

```go
// internal/config/schema.go
type Config struct {
    Language    string                     `yaml:"language,omitempty"`
    Root        string                     `yaml:"root,omitempty"`
    TestRoot    string                     `yaml:"test_root,omitempty"`
    FixtureDir  string                     `yaml:"fixture_dir,omitempty"`
    ExcludeDirs []string                   `yaml:"exclude_dirs,omitempty"`
    LLM         LLMConfig                  `yaml:"llm,omitempty"`
    Languages   map[string]LanguageConfig  `yaml:"languages,omitempty"`
    Workspaces  []WorkspaceConfig          `yaml:"workspaces,omitempty"`

    ConfigPath  string `yaml:"-"` // set by loader; empty when no file found
}

type LLMConfig struct {
    Enabled              bool    `yaml:"enabled,omitempty"`
    Provider             string  `yaml:"provider,omitempty"`
    Model                string  `yaml:"model,omitempty"`
    MaxTokensPerFunction int     `yaml:"max_tokens_per_function,omitempty"`
    Temperature          float64 `yaml:"temperature,omitempty"`
    APIKeyEnvVar         string  `yaml:"api_key_env_var,omitempty"`
    BaseURL              string  `yaml:"base_url,omitempty"`
}

type LanguageConfig struct {
    TestRoot      string            `yaml:"test_root,omitempty"`
    FixtureDir    string            `yaml:"fixture_dir,omitempty"`
    FixtureSuffix string            `yaml:"fixture_suffix,omitempty"`
    Framework     string            `yaml:"framework,omitempty"`
    MockLibrary   string            `yaml:"mock_library,omitempty"`
    Extra         map[string]string `yaml:"extra,omitempty"`
}

type WorkspaceConfig struct {
    Name     string     `yaml:"name,omitempty"`     // identifier for --workspace flag
    Path     string     `yaml:"path,omitempty"`     // relative to config file
    Language string     `yaml:"language,omitempty"`
    LLM      *LLMConfig `yaml:"llm,omitempty"`      // nil = inherit root LLM
}
```

### Workspace helpers

```go
// WorkspaceID returns ws.Name if set, otherwise ws.Path.
func WorkspaceID(ws *WorkspaceConfig) string

// WorkspaceLLM returns the workspace-level LLM config when set, otherwise root.
func WorkspaceLLM(root LLMConfig, ws *WorkspaceConfig) LLMConfig
```

### Loader

```go
// internal/config/loader.go
func Load(startDir string) (*Config, error)       // walks up to find .testsmith.yaml
func LoadFromFile(path string) (*Config, error)   // loads explicit path
```

`Load` walks upward from `startDir` looking for `.testsmith.yaml`, then falls back to `pyproject.toml [tool.testsmith]`. If neither is found it returns `config.Default()` with `ConfigPath == ""`.

The loader merges over defaults using a double-unmarshal approach: the raw YAML is first decoded into a `map[string]any` to capture only the keys that are actually present, then re-marshalled and decoded into the `Config` struct — so unset keys keep their default values rather than being zeroed.

### Files Involved

| File | Role |
|------|------|
| `internal/config/schema.go` | Struct definitions with yaml tags |
| `internal/config/loader.go` | File discovery, YAML/TOML parsing, merge-over-defaults |
| `internal/config/defaults.go` | `Default()` and `ApplyToContext()` |
| `internal/config/loader_test.go` | Snake_case round-trip tests |
| `internal/config/workspace_test.go` | `WorkspaceID`, `WorkspaceLLM`, YAML round-trip |
