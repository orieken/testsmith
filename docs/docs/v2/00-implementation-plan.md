# TestSmith v2 — Go Implementation Plan

## Overview

TestSmith v2 rewrites the CLI in Go as a language-agnostic test scaffold generator. The Python tool (v1, available on PyPI) continues to work and receive bug fixes during the transition. v2 ships as a single static binary with no runtime dependencies.

**Status: All phases complete.** The binary is fully functional across all five languages with LLM integration, monorepo workspace support, and a complete CLI command set.

---

## Architecture Principles

1. **Clean Architecture layering** — domain has zero outward imports; only the CLI layer and adapters (drivers, LLM, config) import from internal layers.
2. **`LanguageDriver` as the only extension point** — adding a language means adding one `internal/drivers/<lang>/` package and one registration line. No core changes.
3. **`GenerationPlan` separates intent from I/O** — the entire pipeline is pure computation; `Executor` is the only function that touches the filesystem. `--dry-run` is free.
4. **LLM is injected, never assumed** — `BodyGenerator` is an optional nil-safe interface; the tool works offline by default.
5. **Backward compat for v1 users** — `pyproject.toml [tool.testsmith]` is honoured as a fallback config source.

---

## Directory Layout

```
v2/
├── cmd/testsmith/              # Cobra CLI entry point
│   ├── main.go                 # composition root: register drivers, wire dependencies
│   ├── root.go                 # root command + persistent flags (--config, --verbose, --dry-run)
│   ├── adapters.go             # adapters list subcommand
│   ├── cli_test.go             # black-box CLI integration tests (package main_test)
│   ├── completion.go           # shell completion subcommand (bash/zsh/fish/powershell)
│   ├── config.go               # config show subcommand
│   ├── gaps.go                 # gaps subcommand
│   ├── generate.go             # generate subcommand (--all, --workers, --workspace)
│   ├── graph.go                # graph subcommand
│   ├── init.go                 # init subcommand
│   ├── migrate.go              # migrate subcommand (--from, --to)
│   ├── prune.go                # prune subcommand
│   ├── testfiles.go            # shared file-discovery helpers (walkTestFiles, relPath)
│   ├── validate.go             # validate subcommand
│   ├── version.go              # version subcommand
│   └── watch.go                # watch subcommand
│
├── internal/
│   ├── domain/                 # pure types + interfaces (ZERO external imports)
│   │   ├── adapter.go          # TestAdapter, AdapterRegistry
│   │   ├── driver.go           # LanguageDriver + BodyGenerator interfaces
│   │   ├── migrator.go         # Migrator interface
│   │   ├── types.go            # ProjectContext, SourceAnalysis, GenerationPlan, etc.
│   │   └── validator.go        # ValidationIssue, Severity constants
│   │
│   ├── registry/
│   │   └── registry.go         # DriverRegistry: detect language, dispatch to driver
│   │
│   ├── analysis/
│   │   ├── pipeline.go         # AnalysisPipeline: AnalyzeFile, DiscoverUntested, DiscoverAndAnalyzeAll
│   │   └── graph.go            # DependencyGraph construction, metrics, Mermaid + table rendering
│   │
│   ├── generation/
│   │   ├── pipeline.go         # GenerationPipeline: build GenerationPlan
│   │   ├── executor.go         # write GenerationPlan to disk (only I/O layer)
│   │   ├── coverage.go         # gap detection, prioritisation, report rendering
│   │   ├── prune.go            # fixture pruning: scan, identify unused, delete
│   │   └── workers.go          # ClampWorkers helper for parallel generation
│   │
│   ├── migration/
│   │   └── text.go             # TextMigrator fluent builder (regex-based file rewriting)
│   │
│   ├── validation/
│   │   └── text.go             # TextValidator fluent builder (Require / Forbid rules)
│   │
│   ├── drivers/
│   │   ├── python/             # Python + pytest / unittest
│   │   │   ├── driver.go
│   │   │   ├── migrators.go    # pytestMockToUnittestMock, and reverse
│   │   │   └── validators.go   # pytestMockValidator, unittestMockValidator, ...
│   │   ├── typescript/         # TypeScript/JS + Jest/Vitest
│   │   │   ├── driver.go
│   │   │   ├── migrators.go    # jestToVitest, vitestToJest
│   │   │   └── validators.go   # jestValidator, vitestValidator
│   │   ├── golang/             # Go + testing / testify
│   │   │   ├── driver.go
│   │   │   ├── migrators.go    # (empty — AST-level rewrites too complex for regex)
│   │   │   └── validators.go   # testifyValidator, stdlibValidator
│   │   ├── java/               # Java + JUnit 4/5
│   │   │   ├── driver.go
│   │   │   ├── migrators.go    # junit4ToJunit5, junit5ToJunit4
│   │   │   └── validators.go   # junit5Validator, junit4Validator, testngValidator
│   │   └── csharp/             # C# + xUnit / NUnit / MSTest
│   │       ├── driver.go
│   │       ├── migrators.go    # nunitToXunit, xunitToNunit
│   │       └── validators.go   # xunitValidator, nunitValidator, mstestValidator
│   │
│   ├── llm/
│   │   ├── llm.go              # LLMBodyGenerator implements domain.BodyGenerator
│   │   ├── factory/            # Build() selects provider from LLMConfig
│   │   ├── anthropic/          # Anthropic Messages API (net/http)
│   │   ├── openai/             # OpenAI-compatible Chat Completions (net/http)
│   │   └── ollama/             # Ollama local REST (net/http)
│   │
│   ├── config/
│   │   ├── loader.go           # find + parse .testsmith.yaml / pyproject.toml
│   │   ├── schema.go           # Config, LLMConfig, LanguageConfig, WorkspaceConfig structs
│   │   ├── defaults.go         # per-driver default values + ApplyToContext
│   │   ├── loader_test.go      # snake_case YAML round-trip tests
│   │   └── workspace_test.go   # WorkspaceID, WorkspaceLLM, YAML round-trip tests
│   │
│   ├── integration/
│   │   └── integration_test.go # end-to-end pipeline tests against testdata/
│   │
│   └── watch/
│       ├── watcher.go          # fsnotify + debounce watcher
│       └── watcher_test.go
│
├── testdata/                   # per-language sample source files for integration tests
│   ├── python/
│   ├── typescript/
│   ├── golang/
│   ├── java/
│   ├── csharp/
│   └── workspace/              # monorepo fixture: Go api + TypeScript frontend
│
├── docs/v2/                    # feature documentation (this directory)
├── go.mod
├── go.sum
├── .testsmith.yaml             # dogfooded config
├── Makefile
└── .goreleaser.yaml            # cross-platform release builds
```

---

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| `github.com/spf13/cobra` | CLI framework + shell completion |
| `github.com/smacker/go-tree-sitter` | Polyglot AST parsing (Python, TypeScript, Java, C#) |
| `github.com/fsnotify/fsnotify` | Cross-platform file watching |
| `github.com/BurntSushi/toml` | TOML parsing (pyproject.toml fallback) |
| `gopkg.in/yaml.v3` | YAML parsing (.testsmith.yaml) |
| `go/ast`, `go/parser` | Go source analysis (stdlib, no extra dep) |

All LLM providers use plain `net/http` — no Anthropic or OpenAI SDK dependency.

---

## Build Plan — Completed Phases

### Phase 0 — Scaffold ✅
Project compiles; `testsmith version` works; all interfaces defined and satisfied.

### Phase 1 — Python Driver MVP ✅
`testsmith generate src/payment.py` produces pytest scaffolds identical to TestSmith v1.

### Phase 2 — Full Python Feature Parity ✅
`testsmith generate --all`, `graph`, `prune`, `gaps`, `watch`, `init` all work on real Python projects.

### Phase 3 — LLM Integration ✅
`testsmith generate src/payment.py --llm` calls the configured provider (Anthropic, OpenAI, or Ollama) and fills in test bodies. Falls back gracefully on API error.

### Phase 4 — TypeScript Driver ✅
`testsmith generate src/payment.ts` generates Jest or Vitest scaffolds based on `package.json` auto-detection.

### Phase 5 — Go Driver ✅
`testsmith generate internal/handlers/user.go` generates `user_test.go` using `go/ast`.

### Phase 6 — Java + C# Drivers + Monorepo Config ✅
Five-language support. `workspaces:` in `.testsmith.yaml` routes `generate --all`, `validate`, `gaps`, `graph`, `prune`, and `watch` per workspace.

### Phase 7 — Migrate + Validate + Workers ✅
- `testsmith migrate --from jest --to vitest` rewrites test files using regex `TextMigrator` builders.
- `testsmith validate` scans existing tests against the selected adapter's conventions using `TextValidator` builders.
- `testsmith generate --all --workers N` fans out generation across N goroutines.

### Phase 8 — Shell Completions + CLI Black-box Tests ✅
- `testsmith completion [bash|zsh|fish|powershell]` generates ready-to-install scripts via Cobra's built-in generator.
- `cmd/testsmith/cli_test.go` — 31 black-box integration tests that compile the binary in `TestMain` and exercise every command as a subprocess.

---

## Migration from v1 (Python)

| v1 Behaviour | v2 Behaviour |
|-------------|-------------|
| `testsmith src/payment.py` | `testsmith generate src/payment.py` |
| `testsmith --all` | `testsmith generate --all` |
| `testsmith --graph` | `testsmith graph` |
| `testsmith --prune` | `testsmith prune` |
| `testsmith --coverage-gaps` | `testsmith gaps` |
| `testsmith --watch` | `testsmith watch` |
| `pyproject.toml [tool.testsmith]` | `.testsmith.yaml` (pyproject.toml still read as fallback) |
| `pip install testsmith` / `pipx install testsmith` | `brew install testsmith` / binary download / `go install` |

No Python installation required for v2.

---

## Testing Strategy

- **Unit tests** — every domain function tested with in-memory data (no file I/O).
- **Driver tests** — each driver has a fixture set in `testdata/<lang>/`; migrator and validator tests use inline content.
- **Integration tests** — `internal/integration/` runs the full pipeline against `testdata/` in a temp directory.
- **Black-box CLI tests** — `cmd/testsmith/cli_test.go` compiles the binary once in `TestMain` and runs all 31 tests as subprocesses, exercising cobra routing, flag parsing, exit codes, and file side-effects.
- **Race detector** — all tests pass under `go test -race`.
- **Cross-platform tests** — GitHub Actions matrix covers Windows, macOS, and Linux.
