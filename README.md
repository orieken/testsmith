# TestSmith v2

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
brew install orieken/tap/testsmith
```

### Download binary
Download the latest release from the [releases page](https://github.com/orieken/testsmith/releases), then make it executable:

```sh
chmod +x testsmith-darwin-arm64
sudo mv testsmith-darwin-arm64 /usr/local/bin/testsmith
```

### Build from source
Requires Go 1.22+.

```sh
git clone https://github.com/orieken/testsmith.git
cd testsmith
go build -o testsmith ./cmd/testsmith
```

---

## Quick start

```sh
# Initialise a project (creates .testsmith.yaml and .testsmith/patterns/)
testsmith init

# Also write Claude Code agents into .claude/agents/
testsmith init --with-agents

# Generate a test for one file
testsmith generate src/services/payment.py

# Generate tests for every untested file in the project
testsmith generate --all

# Preview what would be generated without writing
testsmith generate --all --dry-run

# Capture a testing pattern from an existing test file (requires llm.enabled: true)
testsmith learn src/services/payment_test.py

# Start watching for changes (auto-regenerates on save)
testsmith watch
```

---

## Commands

### `generate`
Generate test scaffolds for one file, a directory, or the whole project.

```
testsmith generate [file] [flags]

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
testsmith generate src/payment.py
testsmith generate --all
testsmith generate --all --workers 8
testsmith generate --path src/services/
testsmith generate src/payment.py --llm --overwrite
```

---

### `validate`
Scan existing test files and report mismatches against the configured adapter's conventions. Exits non-zero when errors are found (suitable for CI).

```
testsmith validate [flags]

Flags:
  --lang <name>       Override auto-detected language
  --path <dir>        Restrict to test files under this directory
  --workspace <name>  Validate only this workspace
  -v, --verbose       Print ✓ for each clean file
```

```sh
testsmith validate
testsmith validate --lang java
testsmith validate --workspace api
```

---

### `migrate`
Rewrite existing test files from one framework to another using ordered regex transformations.

```
testsmith migrate [flags]

Required:
  --from <framework>  Source framework (e.g. jest, junit4, pytest-mock, nunit)
  --to   <framework>  Target framework (e.g. vitest, junit5, unittest-mock, xunit)

Optional:
  --path <dir>        Restrict to test files under this directory
  --lang <name>       Override auto-detected language
  --dry-run           Print what would change without writing files
```

```sh
testsmith migrate --from jest --to vitest
testsmith migrate --from junit4 --to junit5 --path src/test/
testsmith migrate --from pytest-mock --to unittest-mock --dry-run
```

Available pairs: `jest↔vitest`, `junit4↔junit5`, `pytest-mock↔unittest-mock`, `nunit↔xunit`.

---

### `gaps`
Analyse all source files and produce a prioritised Markdown coverage report.

```
testsmith gaps [flags]

Flags:
  --output <file>     Output file (default: testsmith_coverage_report.md)
  --top <n>           Show only the top N gaps
  --workspace <name>  Analyse only this workspace
  --dry-run           Print the report to stdout
```

```sh
testsmith gaps
testsmith gaps --top 10 --dry-run
testsmith gaps --output coverage.md
```

---

### `graph`
Build a Mermaid dependency graph and coupling-score table for all source modules.

```
testsmith graph [flags]

Flags:
  --output <file>     Output Markdown file (default: testsmith_graph.md)
  --workspace <name>  Graph only this workspace
  --dry-run           Print the report to stdout
```

```sh
testsmith graph
testsmith graph --dry-run
testsmith graph --output deps.md
```

---

### `prune`
Find fixture files that no longer match any active external dependency.

```
testsmith prune [flags]

Flags:
  --confirm           Actually delete unused fixtures (default: dry-run)
  --workspace <name>  Prune only this workspace
```

```sh
testsmith prune             # preview what would be removed
testsmith prune --confirm   # delete unused fixtures
```

---

### `watch`
Monitor source files and automatically regenerate test scaffolds on save.

```
testsmith watch [flags]

Flags:
  --debounce <ms>     Debounce interval in milliseconds (default: 500)
  --llm               Enable LLM body generation on watched changes
  --workspace <name>  Watch only this workspace
  -v, --verbose       Log each file event
```

```sh
testsmith watch
testsmith watch --debounce 1000 --llm
```

---

### `init`
Scaffold a `.testsmith.yaml`, standard test directories, and a `.testsmith/patterns/` directory with a README.

```
testsmith init [flags]

Flags:
  --lang <name>      Force a specific language instead of auto-detecting
  --with-agents      Write bundled Claude Code agent files into .claude/agents/
  --dry-run          Print what would be created without writing files
```

```sh
testsmith init
testsmith init --lang python
testsmith init --with-agents   # also writes Claude Code agents into .claude/agents/
```

---

### `learn`
Read a test file, extract its non-obvious testing patterns using the configured LLM, and write the result to `.testsmith/patterns/<slug>.md` for future `generate` runs.

Requires `llm.enabled: true` in `.testsmith.yaml`. Skips writing if the target pattern file already exists.

```
testsmith learn <file> [flags]

Flags:
  --dry-run   Print the extracted pattern without writing a file
```

```sh
testsmith learn src/payment_test.py
testsmith learn internal/db/store_test.go --dry-run
```

---

### `adapters list`
List all available adapters for the detected (or specified) language.

```
testsmith adapters list [flags]

Flags:
  --lang <name>   Show adapters for this language
```

```sh
testsmith adapters list
testsmith adapters list --lang java
```

---

### `config show`
Print the resolved configuration (defaults merged with any `.testsmith.yaml`).

```sh
testsmith config show
```

---

### `completion`
Generate shell completion scripts.

```sh
# Bash (load for session)
source <(testsmith completion bash)

# Zsh
testsmith completion zsh > "${fpath[1]}/_testsmith"

# Fish
testsmith completion fish | source

# PowerShell
testsmith completion powershell | Out-String | Invoke-Expression
```

---

## Configuration (`.testsmith.yaml`)

Run `testsmith init` to generate a starter config. Full schema:

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

TestSmith works offline with TODO stubs. Pass `--llm` to have an LLM write the test bodies.

### Anthropic (default)
```sh
export ANTHROPIC_API_KEY=sk-ant-...
testsmith generate src/payment.py --llm
```

### OpenAI
```yaml
# .testsmith.yaml
llm:
  provider: openai
  model: gpt-4o
  api_key_env_var: OPENAI_API_KEY
```

### Ollama (local, no API key)
```yaml
# .testsmith.yaml
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
go test ./cmd/testsmith/... -v -timeout 120s

# Run end-to-end pipeline tests
go test ./internal/integration/... -v

# Build the binary
go build -o testsmith ./cmd/testsmith

# Lint
golangci-lint run
```

### Project layout
```
cmd/testsmith/       CLI commands (Cobra) + black-box integration tests
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
