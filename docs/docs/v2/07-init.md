# Feature: init — Project Initialisation

## What It Does

`testsmith init` bootstraps the test infrastructure for a project that has not yet used TestSmith. It:

1. Detects (or confirms with `--lang`) the project language.
2. Creates the test root directory and fixture directory for the detected language.
3. Writes a `.testsmith.yaml` pre-populated with language-appropriate defaults.

Running `init` on an already-initialised project is safe — it skips the `.testsmith.yaml` write and prints `already exists`.

---

## CLI

```
testsmith init [flags]

Flags:
  --lang <lang>   Hint the primary language if auto-detection fails
  --dry-run       Print what would be created and the config content without writing files
```

---

## Example Output

```
$ testsmith init
  ✓ created  tests/
  ✓ created  tests/fixtures/
  ✓ created  .testsmith.yaml

Initialised python project. Run 'testsmith generate --all' to start.
```

Dry-run output:

```
$ testsmith init --dry-run
  ✓ created  tests/
  ✓ created  tests/fixtures/

-- .testsmith.yaml (dry-run) --
language: python
test_root: tests/
fixture_dir: tests/fixtures/
exclude_dirs:
    - node_modules
    - .venv
    ...
llm:
    provider: anthropic
    model: claude-sonnet-4-6
    api_key_env_var: ANTHROPIC_API_KEY
```

---

## Go Implementation

`init` detects the language via `reg.Detect(cwd)`, looks up per-language defaults from `config.Default()`, creates directories with `os.MkdirAll`, and marshals a typed `*config.Config` struct directly to YAML using `gopkg.in/yaml.v3`. All config fields have `yaml:"snake_case"` struct tags so the written file is immediately loadable by `config.LoadFromFile`.

```go
// cmd/testsmith/init.go
func runInit(langHint string) error
```

Key behaviour:
- If `.testsmith.yaml` already exists the function prints `already exists — skipping` and returns nil (idempotent).
- `--dry-run` prints the YAML to stdout via a `-- .testsmith.yaml (dry-run) --` banner instead of writing the file.
- Directory creation is never skipped — `os.MkdirAll` is a no-op when the directory already exists.

### YAML Round-trip Guarantee

The `*config.Config` struct carries `yaml:"snake_case,omitempty"` tags on every field (see `internal/config/schema.go`). This means:

```go
data, _ := yaml.Marshal(cfg)           // writes  test_root: tests/
loaded, _ := config.LoadFromFile(path) // reads   loaded.TestRoot == "tests/"
```

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/init.go` | Cobra subcommand, directory creation, YAML write |
| `internal/config/schema.go` | Config struct with yaml tags |
| `internal/config/defaults.go` | Per-language default values |
