# Assay

> *assay (v.)* — to test the quality or composition of something; from Old French *assai*, "trial, test." In metallurgy, an assay determines the purity of a metal sample. Here, it determines the test coverage of your code.

**Language-agnostic test scaffold generator.** Point it at any source file and it writes the boilerplate so you can write the assertions.

| Language | Frameworks | Mocking |
|----------|-----------|---------|
| Python | pytest, unittest | pytest-mock, unittest.mock |
| TypeScript / JavaScript | Jest, Vitest, Mocha | jest.fn(), vi.fn(), sinon |
| Go | testing (stdlib) | interfaces, testify, gomock |
| Java | JUnit 5, JUnit 4, TestNG, Spring Boot | Mockito |
| C# | xUnit, NUnit, MSTest | Moq, NSubstitute |

---

## Install

### Homebrew (macOS / Linux)
```sh
brew install orieken/tap/assay
```

### Download binary
Download the latest release from the [releases page](https://github.com/orieken/assay/releases), then make it executable:

```sh
chmod +x assay-darwin-arm64
sudo mv assay-darwin-arm64 /usr/local/bin/assay
```

### Build from source
Requires Go 1.22+.

```sh
git clone https://github.com/orieken/assay.git
cd assay
go build -o assay ./cmd/assay
```

---

## Quick start

```sh
# Initialise a project (creates .assay.yaml and .assay/patterns/)
assay init

# Also write Claude Code agents into .claude/agents/
assay init --with-agents

# Generate a test for one file
assay generate src/services/payment.py

# Generate tests for every untested file in the project
assay generate --all

# Preview what would be generated without writing
assay generate --all --dry-run

# Capture a testing pattern from an existing test file (requires llm.enabled: true)
assay learn src/services/payment_test.py

# Start watching for changes (auto-regenerates on save)
assay watch
```

---

## Commands

### `generate`
Generate test scaffolds for one file, a directory, or the whole project.

```
assay generate [file] [flags]

Flags:
  --all               Generate tests for every untested source file
  --path <dir>        Generate tests for untested files under this directory
  --overwrite         Regenerate even if a test file already exists
  --llm               Use an LLM to write test bodies (see LLM setup)
  --lang <name>       Override auto-detected language
  --workers <n>       Parallel workers for --all / --path (default: NumCPU)
  --workspace <name>  Process only this workspace (monorepo support)
  --dry-run           Print what would be created without writing any files
  -v, --verbose       Show detailed per-file analysis output
```

```sh
assay generate src/payment.py
assay generate --all
assay generate --all --workers 8
assay generate --path src/services/
assay generate src/payment.py --llm --overwrite
```

---

### `validate`
Scan existing test files and report mismatches against the configured adapter's conventions. Exits non-zero when errors are found (suitable for CI).

```
assay validate [flags]

Flags:
  --lang <name>       Override auto-detected language
  --path <dir>        Restrict to test files under this directory
  --workspace <name>  Validate only this workspace
  -v, --verbose       Print ✓ for each clean file
```

```sh
assay validate
assay validate --lang java
assay validate --workspace api
```

---

### `migrate`
Rewrite existing test files from one framework to another using ordered regex transformations.

```
assay migrate [flags]

Required:
  --from <framework>  Source framework (e.g. jest, junit4, pytest-mock, nunit)
  --to   <framework>  Target framework (e.g. vitest, junit5, unittest-mock, xunit)

Optional:
  --path <dir>        Restrict to test files under this directory
  --lang <name>       Override auto-detected language
  --dry-run           Print what would change without writing files
```

```sh
assay migrate --from jest --to vitest
assay migrate --from junit4 --to junit5 --path src/test/
assay migrate --from pytest-mock --to unittest-mock --dry-run
```

Available pairs: `jest↔vitest`, `junit4↔junit5`, `pytest-mock↔unittest-mock`, `nunit↔xunit`.

---

### `gaps`
Analyse all source files and produce a prioritised Markdown coverage report.

```
assay gaps [flags]

Flags:
  --output <file>     Output file (default: assay_coverage_report.md)
  --top <n>           Show only the top N gaps
  --workspace <name>  Analyse only this workspace
  --dry-run           Print the report to stdout
```

```sh
assay gaps
assay gaps --top 10 --dry-run
assay gaps --output coverage.md
```

---

### `graph`
Build a Mermaid dependency graph and coupling-score table for all source modules.

```
assay graph [flags]

Flags:
  --output <file>     Output Markdown file (default: assay_graph.md)
  --workspace <name>  Graph only this workspace
  --dry-run           Print the report to stdout
```

```sh
assay graph
assay graph --dry-run
assay graph --output deps.md
```

---

### `prune`
Find fixture files that no longer match any active external dependency.

```
assay prune [flags]

Flags:
  --confirm           Actually delete unused fixtures (default: dry-run)
  --workspace <name>  Prune only this workspace
```

```sh
assay prune             # preview what would be removed
assay prune --confirm   # delete unused fixtures
```

---

### `watch`
Monitor source files and automatically regenerate test scaffolds on save.

```
assay watch [flags]

Flags:
  --debounce <ms>     Debounce interval in milliseconds (default: 500)
  --llm               Enable LLM body generation on watched changes
  --workspace <name>  Watch only this workspace
  -v, --verbose       Log each file event
```

```sh
assay watch
assay watch --debounce 1000 --llm
```

---

### `init`
Scaffold a `.assay.yaml`, standard test directories, and a `.assay/patterns/` directory with a README.

```
assay init [flags]

Flags:
  --lang <name>      Force a specific language instead of auto-detecting
  --with-agents      Write bundled Claude Code agent files into .claude/agents/
  --dry-run          Print what would be created without writing files
```

```sh
assay init
assay init --lang python
assay init --with-agents   # also writes Claude Code agents into .claude/agents/
```

---

### `learn`
Read a test file, extract its non-obvious testing patterns using the configured LLM, and write the result to `.assay/patterns/<slug>.md` for future `generate` runs.

Requires `llm.enabled: true` in `.assay.yaml`. Skips writing if the target pattern file already exists.

```
assay learn <file> [flags]

Flags:
  --dry-run   Print the extracted pattern without writing a file
```

```sh
assay learn src/payment_test.py
assay learn internal/db/store_test.go --dry-run
```

---

### `adapters list`
List all available adapters for the detected (or specified) language.

```
assay adapters list [flags]

Flags:
  --lang <name>   Show adapters for this language
```

```sh
assay adapters list
assay adapters list --lang java
```

---

### `config show`
Print the resolved configuration (defaults merged with any `.assay.yaml`).

```sh
assay config show
```

---

### `completion`
Generate shell completion scripts.

```sh
# Bash (load for session)
source <(assay completion bash)

# Zsh
assay completion zsh > "${fpath[1]}/_assay"

# Fish
assay completion fish | source

# PowerShell
assay completion powershell | Out-String | Invoke-Expression
```

---

## Configuration (`.assay.yaml`)

Run `assay init` to generate a starter config. Full schema:

```yaml
language: python          # override auto-detection

test_root: tests/
fixture_dir: tests/fixtures/

exclude_dirs:
  - node_modules
  - .venv
  - vendor
  - build
  - dist

# LLM body generation (optional — works offline without this)
llm:
  enabled: false
  provider: anthropic          # anthropic | openai | ollama
  model: claude-sonnet-4-6
  max_tokens_per_function: 1500
  temperature: 0.0
  api_key_env_var: ANTHROPIC_API_KEY
  # base_url: http://localhost:11434/v1   # Ollama or OpenAI-compatible

# Per-language overrides
languages:
  python:
    test_root: tests/
    fixture_dir: tests/fixtures/
    framework: pytest
    mock_library: pytest-mock
  typescript:
    test_root: src/
    fixture_dir: __mocks__/
  java:
    test_root: src/test/java/

# Monorepo workspace support
# All commands that accept --workspace use these entries.
workspaces:
  - name: api
    path: services/api
    language: go
  - name: frontend
    path: services/frontend
    language: typescript
    llm:
      provider: openai
      model: gpt-4o
```

---

## LLM setup

Assay works offline with TODO stubs. Pass `--llm` to have an LLM write the test bodies.

### Anthropic (default)
```sh
export ANTHROPIC_API_KEY=sk-ant-...
assay generate src/payment.py --llm
```

### OpenAI
```yaml
# .assay.yaml
llm:
  provider: openai
  model: gpt-4o
  api_key_env_var: OPENAI_API_KEY
```

### Ollama (local, no API key)
```yaml
# .assay.yaml
llm:
  provider: ollama
  model: llama3
  base_url: http://localhost:11434/v1
```

---

## How it works

```
Source file
    │
    ▼
LanguageDriver.AnalyzeFile()      → SourceAnalysis (imports, public API)
    │
    ▼
generation.Pipeline.Plan()        → GenerationPlan (pure data, no I/O)
    │   optionally: BodyGenerator via LLM
    ▼
generation.Executor.Execute()     → writes files to disk
```

Each language driver is isolated behind the `LanguageDriver` interface. Adding a new language means implementing that interface — no changes to the pipeline or CLI are required.

---

## Development

```sh
# Run all tests (with race detector)
go test -race ./...

# Run a specific package
go test ./internal/drivers/python/... -v

# Run black-box CLI integration tests
go test ./cmd/assay/... -v -timeout 120s

# Run end-to-end pipeline tests
go test ./internal/integration/... -v

# Build the binary
go build -o assay ./cmd/assay

# Lint
golangci-lint run
```

### Project layout
```
cmd/assay/       CLI commands (Cobra) + black-box integration tests
internal/
  analysis/          Source discovery and analysis pipeline
  config/            Config loading, defaults, yaml tags
  domain/            Pure types and interfaces (no dependencies)
  drivers/           One package per language
    python/
    typescript/
    golang/
    java/
    csharp/
  generation/        GenerationPlan builder, Executor, gap analysis, prune
  integration/       End-to-end pipeline tests against testdata/
  llm/               LLM adapter (Anthropic, OpenAI, Ollama)
  migration/         TextMigrator fluent builder (regex-based rewrites)
  registry/          Language driver registry
  validation/        TextValidator fluent builder (Require / Forbid rules)
  watch/             Debounced file-system watcher
testdata/            Fixture projects for driver tests
  python/
  typescript/
  golang/
  java/
  csharp/
  workspace/         Monorepo fixture (Go api + TypeScript frontend)
```

---

## License

MIT
