# TestSmith v2 — Go Implementation Plan

## Overview

TestSmith v2 rewrites the CLI in Go as a language-agnostic test scaffold generator. The Python tool (v1, available on PyPI) continues to work and receive bug fixes during the transition. v2 ships as a single static binary with no runtime dependencies.

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
├── cmd/testsmith/          # Cobra CLI entry point
│   ├── main.go             # composition root: register drivers, wire dependencies
│   ├── root.go             # root command + persistent flags (--config, --verbose, --dry-run)
│   ├── generate.go         # generate subcommand
│   ├── graph.go
│   ├── prune.go
│   ├── gaps.go
│   ├── watch.go
│   └── init.go
│
├── internal/
│   ├── domain/             # pure types + interfaces (ZERO external imports)
│   │   ├── driver.go       # LanguageDriver + BodyGenerator interfaces
│   │   ├── types.go        # ProjectContext, SourceAnalysis, GenerationPlan, etc.
│   │   └── errors.go       # sentinel errors
│   │
│   ├── registry/
│   │   └── registry.go     # DriverRegistry: detect language, dispatch to driver
│   │
│   ├── analysis/
│   │   ├── pipeline.go     # AnalysisPipeline: file analysis + discovery
│   │   ├── discovery.go    # find untested source files
│   │   └── graph.go        # DependencyGraph construction + Mermaid rendering
│   │
│   ├── generation/
│   │   ├── pipeline.go     # GenerationPipeline: build GenerationPlan
│   │   ├── executor.go     # write GenerationPlan to disk (only I/O layer)
│   │   ├── coverage.go     # gap detection + prioritisation + report
│   │   └── prune.go        # fixture pruning logic
│   │
│   ├── drivers/
│   │   ├── python/         # Python + pytest
│   │   ├── typescript/     # TypeScript/JS + Jest/Vitest
│   │   ├── golang/         # Go + testing package
│   │   └── java/           # Java + JUnit 5
│   │
│   ├── llm/
│   │   ├── generator.go    # LLMBodyGenerator implements domain.BodyGenerator
│   │   ├── anthropic/      # Anthropic Messages API (net/http)
│   │   ├── openai/         # OpenAI-compatible Chat Completions (net/http)
│   │   └── ollama/         # Ollama local REST (net/http)
│   │
│   ├── config/
│   │   ├── loader.go       # find + parse .testsmith.yaml / pyproject.toml
│   │   ├── schema.go       # Config, LLMConfig, LanguageConfig structs
│   │   └── defaults.go     # per-driver default values
│   │
│   ├── fsutil/
│   │   ├── write.go        # atomic safe-write, directory creation
│   │   └── walk.go         # file discovery helpers
│   │
│   └── watch/
│       └── watcher.go      # fsnotify + debounce
│
├── testdata/               # per-language sample source files for integration tests
│   ├── python/
│   ├── typescript/
│   ├── golang/
│   └── java/
│
├── docs/v2/                # feature documentation (this directory)
├── go.mod
├── go.sum
├── .testsmith.yaml         # dogfooded config
├── Makefile
└── .goreleaser.yaml        # cross-platform release builds
```

---

## Dependencies

| Dependency | Purpose |
|-----------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/smacker/go-tree-sitter` | Polyglot AST parsing (Python, TypeScript, Java) |
| `github.com/fsnotify/fsnotify` | Cross-platform file watching |
| `github.com/BurntSushi/toml` | TOML parsing (pyproject.toml fallback) |
| `gopkg.in/yaml.v3` | YAML parsing (.testsmith.yaml) |
| `go/ast`, `go/parser` | Go source analysis (stdlib, no extra dep) |

All LLM providers use plain `net/http` — no Anthropic or OpenAI SDK dependency.

---

## Phased Build Plan

### Phase 0 — Scaffold (Days 1–2)
**Goal**: Project compiles; `testsmith version` works; all interfaces are defined and satisfied by stubs.

- [x] `go.mod` initialised
- [x] `internal/domain/` — all types, interfaces, and errors
- [ ] `internal/registry/registry.go` — stub implementation
- [ ] `cmd/testsmith/main.go` + `root.go` — root Cobra command
- [ ] `cmd/testsmith/generate.go` — subcommand stub (returns "not implemented")
- [ ] All other subcommand stubs
- [ ] Empty driver structs satisfying `LanguageDriver` with `return nil, errors.New("not implemented")`

**Completion criterion**: `go build ./...` passes with no errors.

---

### Phase 1 — Python Driver MVP (Weeks 1–2)
**Goal**: `testsmith generate src/payment.py` produces the same output as TestSmith v1.

- [ ] `internal/drivers/python/detector.go` — root detection + package map scan
- [ ] `internal/drivers/python/analyzer.go` — tree-sitter import + public API extraction
- [ ] `internal/drivers/python/classifier.go` — stdlib map + classify
- [ ] `internal/drivers/python/stdlib_modules.go` — embedded Python 3.11 stdlib names
- [ ] `internal/drivers/python/generator.go` — `GenerateTestFile`, `GenerateFixture`, `GenerateBootstrap`
- [ ] `internal/drivers/python/conftest.go` — conftest.py read/update logic
- [ ] `internal/drivers/python/queries/imports.scm` — tree-sitter import query
- [ ] `internal/drivers/python/queries/public_api.scm` — tree-sitter API query
- [ ] `internal/analysis/pipeline.go` — `AnalyzeFile`, `DiscoverUntested`
- [ ] `internal/generation/pipeline.go` — `Plan()`
- [ ] `internal/generation/executor.go` — `Execute()`
- [ ] `internal/fsutil/write.go` + `walk.go`
- [ ] `internal/config/loader.go` + `schema.go` + `defaults.go`
- [ ] `internal/registry/registry.go` — full implementation
- [ ] `cmd/testsmith/generate.go` — wired end-to-end
- [ ] Integration tests using `testdata/python/` samples

**Completion criterion**: All existing Python v1 integration test cases pass against the Go binary.

---

### Phase 2 — Full Python Feature Parity (Week 3)
**Goal**: Every v1 command works for Python projects via the Go binary.

- [ ] `internal/analysis/discovery.go` — `--all` and `--path` batch discovery
- [ ] `internal/analysis/graph.go` — dependency graph + Mermaid rendering
- [ ] `internal/generation/coverage.go` — gap detection + prioritisation
- [ ] `internal/generation/prune.go` — fixture pruning
- [ ] `internal/watch/watcher.go` — fsnotify + debounce
- [ ] `cmd/testsmith/graph.go`, `prune.go`, `gaps.go`, `watch.go`, `init.go`

**Completion criterion**: `testsmith generate --all`, `graph`, `prune`, `gaps`, `watch` all work on a real Python project.

---

### Phase 3 — LLM Integration (Days 1–4 of Week 4)
**Goal**: `testsmith generate src/payment.py --llm` calls the LLM and fills test bodies.

- [ ] `internal/llm/anthropic/provider.go`
- [ ] `internal/llm/openai/provider.go`
- [ ] `internal/llm/ollama/provider.go`
- [ ] `internal/llm/generator.go` — `LLMBodyGenerator`
- [ ] `internal/drivers/python/prompts/generate_body.tmpl`
- [ ] Wire `--llm` flag in `generate` command

**Completion criterion**: `--llm` produces non-stub test bodies; falls back gracefully on API error.

---

### Phase 4 — TypeScript Driver (Week 5)
**Goal**: `testsmith generate src/payment.ts` generates Jest test scaffolds.

- [ ] `internal/drivers/typescript/` — full driver implementation
- [ ] `package.json` parsing for dependency classification
- [ ] Jest / Vitest detection and template selection
- [ ] Integration tests using `testdata/typescript/` samples

---

### Phase 5 — Go Driver (Days 1–4 of Week 6)
**Goal**: `testsmith generate internal/handlers/user.go` generates `user_test.go`.

- [ ] `internal/drivers/golang/` — full driver using `go/ast`
- [ ] `go.mod` parsing for module name
- [ ] Co-located `_test.go` generation with table-driven pattern
- [ ] Integration tests using `testdata/golang/` samples

---

### Phase 6 — Java + C# Drivers + Monorepo Config (Weeks 7–8)
**Goal**: Five-language support; monorepo workspaces in `.testsmith.yaml` work.

#### Java
- [ ] `internal/drivers/java/` — full driver
- [ ] `pom.xml` / `build.gradle` parsing
- [ ] JUnit 5 + Mockito test generation
- [ ] Integration tests using `testdata/java/` samples

#### C#
- [ ] `internal/drivers/csharp/detector.go` — `.sln` / `.csproj` root detection + namespace map
- [ ] `internal/drivers/csharp/analyzer.go` — tree-sitter `using_directive`, class/method extraction
- [ ] `internal/drivers/csharp/classifier.go` — `System.*`, `Microsoft.*` stdlib prefixes
- [ ] `internal/drivers/csharp/generator.go` — xUnit `[Fact]`/`[Theory]` test class + Moq setup
- [ ] `internal/drivers/csharp/queries/imports.scm` — tree-sitter `using_directive` query
- [ ] `internal/drivers/csharp/queries/public_api.scm` — class / method query
- [ ] `.Tests` project scaffold in `testsmith init`
- [ ] Integration tests using `testdata/csharp/` samples

#### Shared
- [ ] Workspace config loading in `internal/config/loader.go`

---

### Phase 7 — Release Pipeline (Week 8)
**Goal**: Single binary downloads available on GitHub Releases; Homebrew tap works.

- [ ] `.goreleaser.yaml` — linux/amd64, linux/arm64, darwin/arm64, darwin/amd64, windows/amd64
- [ ] GitHub Actions CI matrix (build + test per platform)
- [ ] Homebrew tap formula
- [ ] `CHANGELOG.md` v2.0.0 entry
- [ ] Update root `README.md` with v2 installation instructions
- [ ] Announce deprecation timeline for Python v1

---

## Migration from v1 (Python)

| v1 Behaviour | v2 Behaviour |
|-------------|-------------|
| `testsmith src/payment.py` | `testsmith generate src/payment.py` (`generate` is the default subcommand) |
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
- **Driver tests** — each driver has a fixture set in `testdata/<lang>/` with known-good expected outputs checked as golden files.
- **Integration tests** — `generate` command run against a real temp-dir project; output compared to expected files.
- **Cross-platform tests** — GitHub Actions matrix ensures path separators and binary execution work on Windows, macOS, and Linux.
- **Coverage target** — 85% line coverage, enforced in CI.
