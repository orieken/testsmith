# Feature: Configuration

## What It Does

TestSmith v2 works with **zero configuration** — it auto-detects the project language, root, and conventions. When the defaults don't match your project layout, a `.testsmith.yaml` file at the project root provides overrides.

For Python projects that already use TestSmith v1, the `[tool.testsmith]` section in `pyproject.toml` is read as a fallback, preserving backward compatibility.

---

## `.testsmith.yaml` Full Schema

```yaml
# .testsmith.yaml

# Override auto-detected language. Leave empty for auto-detection.
language: ""   # python | typescript | go | java | ruby

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
  - target        # Maven/Gradle
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
# Each key must match a driver's Language() value.
languages:
  python:
    test_root: "tests/"
    fixture_dir: "tests/fixtures/"
    fixture_suffix: "_fixture.py"
    conftest_path: "conftest.py"

  typescript:
    test_root: "src/"
    fixture_dir: "__mocks__/"
    test_file_suffix: ".test.ts"

  go:
    # Go tests are co-located; test_root is ignored.
    fixture_dir: ""

  java:
    test_root: "src/test/java/"

# --- Monorepo / workspace support ---
# Each workspace entry is resolved relative to the config file's directory.
# Settings inherit from root-level config and can be overridden per workspace.
workspaces:
  - path: "packages/api"
    language: "typescript"
  - path: "services/worker"
    language: "go"
```

---

## Loading Precedence

```
CLI flags  >  env vars  >  .testsmith.yaml  >  pyproject.toml [tool.testsmith]  >  driver defaults
```

---

## Go Implementation

```go
// internal/config/schema.go
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

type LLMConfig struct {
    Enabled                bool
    Provider               string
    Model                  string
    MaxTokensPerFunction   int
    Temperature            float64
    APIKeyEnvVar           string
    BaseURL                string
}

type LanguageConfig struct {
    TestRoot       string
    FixtureDir     string
    FixtureSuffix  string
    // Driver-specific keys are stored in a freeform map.
    Extra          map[string]string
}

type WorkspaceConfig struct {
    Path        string
    Language    string
    LLM         *LLMConfig       // pointer; nil means inherit from root
}
```

```go
// internal/config/loader.go
func Load(startDir string) (*Config, error)
func LoadFromFile(path string) (*Config, error)
```

`Load` walks up from `startDir` looking for `.testsmith.yaml` first, then `pyproject.toml`.

### Files Involved

| File | Role |
|------|------|
| `internal/config/schema.go` | Config struct definitions |
| `internal/config/loader.go` | File discovery + YAML/TOML parsing + merge |
| `internal/config/defaults.go` | Per-driver default values |
