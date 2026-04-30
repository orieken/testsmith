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
  --all               Discover and generate tests for every untested source file
  --path <dir>        Generate tests for all untested files under a specific directory
  --dry-run           Print the generation plan without writing any files
  --llm               Call the configured LLM provider to fill in test bodies
  --overwrite         Regenerate even when a test file already exists
  --lang <lang>       Override the auto-detected language (python|typescript|go|java|csharp)
  --workers <n>       Number of parallel workers for --all / --path (default: NumCPU)
  --workspace <name>  Process only this workspace when workspaces are configured
  --config <file>     Path to .testsmith.yaml (default: search up from cwd)
  --verbose, -v       Emit detailed analysis and classification output
```

### Parallel Workers

`--workers N` fans out file processing across N goroutines using a channel-based work queue. The actual concurrency is clamped to `min(N, len(files))` so small file sets never over-provision goroutines.

```go
// internal/generation/workers.go
func ClampWorkers(n, fileCount int) int
```

Output lines from multiple workers are serialised through a `sync.Mutex` so stdout is never interleaved.

### Workspace Mode

When `.testsmith.yaml` defines `workspaces:` and `--all` is given, `generate` iterates each workspace, resolves its driver independently, and prints a per-workspace header:

```
── workspace: api (services/api) ──
  ✓ created  handler_test.go

── workspace: frontend (services/frontend) ──
  ✓ created  src/utils.test.ts

Total — 2 file(s): 2 created/updated, 0 skipped, 0 failed
```

Use `--workspace <name>` to process a single workspace by name or path.

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
type Pipeline struct { driver domain.LanguageDriver }

func (p *Pipeline) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error)
func (p *Pipeline) DiscoverUntested(root string, ctx *domain.ProjectContext) ([]string, error)
func (p *Pipeline) DiscoverInPath(path string, ctx *domain.ProjectContext) ([]string, error)
func (p *Pipeline) DiscoverAndAnalyzeAll(root string, ctx *domain.ProjectContext) ([]*domain.SourceAnalysis, error)

// internal/generation/pipeline.go
type Pipeline struct {
    driver domain.LanguageDriver
    llm    domain.BodyGenerator // nil = stub bodies
}

func (p *Pipeline) Plan(ctx context.Context, a *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GenerationPlan, error)

// internal/generation/executor.go
type Executor struct{}

func (e *Executor) Execute(plan *domain.GenerationPlan) ([]generation.Result, error)

// internal/generation/workers.go
func ClampWorkers(n, fileCount int) int
```

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/generate.go` | Cobra subcommand, wires pipeline, workspace + workers routing |
| `internal/analysis/pipeline.go` | Orchestrates driver.AnalyzeFile + discovery |
| `internal/generation/pipeline.go` | Builds GenerationPlan |
| `internal/generation/executor.go` | Writes GenerationPlan to disk |
| `internal/generation/workers.go` | `ClampWorkers` helper |
| `internal/drivers/<lang>/generator.go` | Driver-specific `GenerateTestFile` + `GenerateFixture` |
| `internal/drivers/<lang>/analyzer.go` | Driver-specific `AnalyzeFile` |

### Idempotency Contract

- A source file whose test already exists is **skipped** unless `--overwrite` is set.
- Fixture files are **appended** (new sub-module mocks added to existing fixture file).
- Bootstrap files (`conftest.py`, `jest.setup.ts`) are **diff-updated** — only missing entries added.
- Running `generate --all` twice produces identical output on the second run.
