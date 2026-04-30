# Changelog

All notable changes to TestSmith v2 are documented here.
Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

---

## [Unreleased]

---

## [2.0.0] — 2026-04-30

### Added

#### Language drivers
- **Python** driver — pytest + pytest-mock / unittest.mock; class-based and function-based test generation; `conftest.py` bootstrap
- **TypeScript** driver — Jest, Vitest, Mocha+Sinon; auto-detected from `package.json` devDependencies; `.test.ts` scaffold generation
- **Go** driver — `go/ast`-based analysis; co-located `_test.go` generation; table-driven test pattern; testify and gomock adapter variants
- **Java** driver — JUnit 5, JUnit 4, TestNG, Spring Boot; Mockito `@Mock` scaffolds; Maven/Gradle project detection
- **C#** driver — xUnit, NUnit, MSTest; Moq and NSubstitute variants; `.csproj`-based root detection

#### CLI commands
- `generate` — scaffold tests for a single file, a directory (`--path`), or the whole project (`--all`)
- `generate --workers N` — parallel fan-out generation across N goroutines
- `generate --workspace <name>` — process a single workspace in a monorepo
- `validate` — scan existing test files for framework convention mismatches; exits non-zero on errors (CI-safe)
- `migrate --from X --to Y` — regex-based bulk rewrite between framework pairs (jest↔vitest, junit4↔junit5, pytest-mock↔unittest-mock, nunit↔xunit)
- `gaps` — prioritised Markdown coverage gap report with coupling-score ranking
- `graph` — Mermaid dependency graph + coupling-score table
- `prune` — identify and optionally delete unused fixture files; comments out stale imports in test files
- `watch` — debounced `fsnotify` watcher; regenerates on source file save; per-workspace goroutines in monorepo mode
- `init` — scaffold `.testsmith.yaml` and test directories for the detected language
- `adapters list` — list all adapters for the current or specified language with selection reason
- `config show` — print the fully-resolved configuration (defaults merged with `.testsmith.yaml`)
- `completion [bash|zsh|fish|powershell]` — generate shell completion scripts via Cobra's built-in generator
- `version` — print the binary version

#### Monorepo workspace support
- `workspaces:` in `.testsmith.yaml` — named workspace entries with per-workspace `language` and `llm` overrides
- `--workspace <name>` flag on `generate`, `validate`, `gaps`, `graph`, `prune`, and `watch`
- `WorkspaceID()` / `WorkspaceLLM()` helpers in `internal/config`

#### LLM body generation
- Anthropic (Claude), OpenAI-compatible, and Ollama providers via plain `net/http`
- `--llm` flag on `generate` and `watch`
- Per-workspace LLM config overrides

#### Migration system
- `migration.TextMigrator` — fluent regex pipeline builder (`Add`, `InjectImport`)
- `LanguageDriver.ListMigrators()` — each driver exposes its available migration pairs

#### Validation system
- `validation.TextValidator` — fluent `Require` / `Forbid` rule builder
- `domain.ValidationIssue` with `error` / `warning` / `info` severity
- `LanguageDriver.ValidateFile()` — per-framework validators for all five languages

#### Configuration
- `.testsmith.yaml` with full `yaml:"snake_case"` struct tags — files written by `init` are immediately loadable by `Load()`
- Per-language `framework` and `mock_library` overrides
- `exclude_dirs`, `test_root`, `fixture_dir` at root and per-language level
- `pyproject.toml [tool.testsmith]` fallback for Python v1 compatibility

#### Testing
- 31 black-box CLI integration tests in `cmd/testsmith/cli_test.go` (`package main_test`) — compile binary once in `TestMain`, exercise every command as a subprocess
- Race detector (`-race`) enforced in CI on all three platforms
- Unit tests for all internal packages
- End-to-end pipeline tests in `internal/integration/`

#### CI / Release
- GitHub Actions CI matrix: Ubuntu, macOS, Windows — build + `go test -race`
- Release workflow: native CGo builds for linux/amd64, darwin/amd64, darwin/arm64, windows/amd64; SHA-256 checksums; GitHub Release auto-notes

### Changed
- Rewritten from Python (v1) to Go — single static binary, no runtime dependencies
- `testsmith <file>` (v1 default) → `testsmith generate <file>` (explicit subcommand)
- `testsmith --all` → `testsmith generate --all`
- `testsmith --graph` → `testsmith graph`
- `testsmith --prune` → `testsmith prune`
- `testsmith --coverage-gaps` → `testsmith gaps`
- `testsmith --watch` → `testsmith watch`

### Migration from v1

See the [v1 → v2 migration table](docs/v2/00-implementation-plan.md#migration-from-v1-python) for the full flag mapping. The v1 Python package (`pip install testsmith`) continues to receive bug fixes during the transition period.

---

[Unreleased]: https://github.com/orieken/testsmith/compare/v2.0.0...HEAD
[2.0.0]: https://github.com/orieken/testsmith/releases/tag/v2.0.0
