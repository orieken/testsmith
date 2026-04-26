# Feature: generate — Test Scaffold Generation

## What It Does

`testsmith generate` is the primary command. Given one or more source files it:

1. Auto-detects the project language by probing registered `LanguageDriver` implementations against the project root markers it finds walking up from the target path.
2. Parses each source file through the driver's AST engine to extract every import and every exported public member (functions, classes, structs, interfaces, methods).
3. Classifies each import as **stdlib** (skip), **internal** (note path for bootstrap), or **external** (generate a mock fixture).
4. Builds a `GenerationPlan` — a pure in-memory list of files to create or update — without touching the disk.
5. Passes the plan to the `Executor` which writes files, creates missing directories, and inits any required package stub files.

Supports three scopes:

| Scope | Flag | Behaviour |
|-------|------|-----------|
| Single file | `testsmith generate src/payment.go` | Analyse one file |
| Directory | `--path src/services/` | All untested files under a directory |
| Whole project | `--all` | Every source file with no corresponding test |

---

## CLI

```
testsmith generate [file] [flags]

Flags:
  --all             Discover and generate tests for every untested source file
  --path <dir>      Generate tests for all untested files under a specific directory
  --dry-run         Print the generation plan without writing any files
  --llm             Call the configured LLM provider to fill in test bodies
  --overwrite       Regenerate even when a test file already exists
  --lang <lang>     Override the auto-detected language (python|typescript|go|java|ruby)
  --config <file>   Path to .testsmith.yaml (default: search up from cwd)
  --verbose, -v     Emit detailed analysis and classification output
```

---

## Outputs (Python / pytest example)

Given `src/services/payment.py` that imports `stripe` and `requests`:

```
tests/
├── src/
│   └── services/
│       ├── __init__.py                  ← created if missing
│       └── test_payment.py              ← test scaffold
├── fixtures/
│   ├── stripe_fixture.py                ← shared mock fixture
│   └── requests_fixture.py
└── conftest.py                          ← updated with sys.path entries
```

Given `internal/handlers/user.go` (Go):

```
internal/handlers/
└── user_test.go                         ← co-located, same package
```

---

## Go Implementation

### Key Types

```go
// internal/analysis/pipeline.go
type AnalysisPipeline struct {
    driver domain.LanguageDriver
}

func (p *AnalysisPipeline) AnalyzeFile(path string) (*domain.SourceAnalysis, error)
func (p *AnalysisPipeline) DiscoverUntested(root string, testRoot string) ([]string, error)

// internal/generation/pipeline.go
type GenerationPipeline struct {
    driver domain.LanguageDriver
    llm    domain.BodyGenerator // nil = stub bodies
}

func (p *GenerationPipeline) Plan(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GenerationPlan, error)

// internal/generation/executor.go
type Executor struct{}

func (e *Executor) Execute(plan *domain.GenerationPlan) ([]Result, error)
```

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/generate.go` | Cobra subcommand, wires pipeline |
| `internal/analysis/pipeline.go` | Orchestrates driver.AnalyzeFile + discovery |
| `internal/analysis/discovery.go` | Finds untested source files |
| `internal/generation/pipeline.go` | Builds GenerationPlan |
| `internal/generation/executor.go` | Writes GenerationPlan to disk |
| `internal/drivers/<lang>/generator.go` | Driver-specific GenerateTestFile + GenerateFixture |
| `internal/drivers/<lang>/analyzer.go` | Driver-specific AnalyzeFile |
| `internal/fsutil/write.go` | Atomic safe-write, directory creation |

### Idempotency Contract

- A source file whose test already exists is **skipped** unless `--overwrite` is set.
- Fixture files are **appended** (new sub-module mocks added to existing fixture file).
- Bootstrap files (`conftest.py`, `jest.setup.ts`) are **diff-updated** — only missing entries added.
- Running `generate --all` twice produces identical output on the second run.
