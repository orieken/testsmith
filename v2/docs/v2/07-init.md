# Feature: init — Project Initialisation

## What It Does

`testsmith init` bootstraps the test infrastructure for a project that has not yet used TestSmith. It:

1. Detects (or confirms with `--lang`) the project language.
2. Creates the test root directory (`tests/` for Python/TypeScript, co-located for Go).
3. Creates the fixture directory and any required stub files (`__init__.py`, etc.).
4. Writes or updates the framework bootstrap file (`conftest.py`, `jest.setup.ts`, etc.) with the minimum required content.
5. Writes a starter `.testsmith.yaml` with commented-out options so developers can discover available configuration.

Running `init` on an already-initialised project is safe — it only creates files that do not yet exist.

---

## CLI

```
testsmith init [flags]

Flags:
  --lang <lang>   Hint the primary language if auto-detection fails
  --dry-run       Print what would be created without touching the filesystem
```

---

## Example Output

```
TestSmith Init
──────────────
Language detected: python

Created:
  ✓ tests/
  ✓ tests/__init__.py
  ✓ tests/fixtures/
  ✓ tests/fixtures/__init__.py
  ✓ conftest.py
  ✓ .testsmith.yaml

Run 'testsmith generate <file>' to scaffold your first test.
```

---

## Go Implementation

`init` is implemented as a specialised `GenerationPlan` — it produces a fixed set of `GeneratedFile` values representing the skeleton infrastructure and passes them to the same `Executor` used by `generate`.

```go
// cmd/testsmith/init.go
func buildInitPlan(driver domain.LanguageDriver, ctx *domain.ProjectContext, cfg *config.Config) *domain.GenerationPlan
```

No new infrastructure is required beyond what `generate` already uses.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/init.go` | Cobra subcommand + plan construction |
| `internal/generation/executor.go` | Shared file writer |
| `internal/config/schema.go` | Starter YAML template |
