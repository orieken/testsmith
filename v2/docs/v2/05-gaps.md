# Feature: gaps — Test Coverage Gap Analysis

## What It Does

`testsmith gaps` inspects every source file in the project to determine whether it has a corresponding test file and, if so, how complete that test file is. It then cross-references the dependency graph to produce a **prioritised list of coverage gaps** — files that need tests the most.

Priority is calculated from:
- `no_test` (source file has no test at all) → highest weight
- `skeleton_only` (test file exists but contains only TODO stubs) → medium weight
- `partial` (some members tested, others have stubs) → lower weight
- Fan-out (external dependency count) — more external deps = harder to test, higher priority
- Fan-in (dependents) — more dependents = higher blast radius if it breaks

The output is a Markdown report sorted by priority score, with a ready-to-copy `testsmith generate` command for each gap.

---

## CLI

```
testsmith gaps [flags]

Flags:
  --output <file>     Output Markdown file (default: testsmith_coverage_report.md)
  --top <n>           Show only the top N gaps (default: show all)
  --workspace <name>  Analyse only this workspace (name or path)
  --dry-run           Print report to stdout instead of writing a file
```

### Workspace Mode

When `workspaces:` are configured, each workspace is analysed independently and gaps are merged into a single prioritised report. Use `--workspace <name>` to limit to one workspace.

---

## Example Report

```markdown
# TestSmith Coverage Gap Report

**2 / 5 source files have test coverage (40%)**

| Priority | File | Status | Ext Deps | Command |
|----------|------|--------|----------|---------|
| 0.95 | `src/services/payment.py` | no_test | 3 | `testsmith generate src/services/payment.py` |
| 0.82 | `src/core/auth.py` | skeleton_only | 2 | `testsmith generate src/core/auth.py --overwrite` |
```

---

## Go Implementation

```go
// internal/generation/coverage.go
func DetectCoverage(analyses []*domain.SourceAnalysis, driver domain.LanguageDriver) []domain.CoverageGap
func PrioritizeGaps(gaps []domain.CoverageGap, metrics []domain.ModuleMetrics) []domain.CoverageGap
func GenerateReport(gaps []domain.CoverageGap, totalSources int) string
func GapsForAnalyses(analyses []*domain.SourceAnalysis, driver domain.LanguageDriver) ([]domain.CoverageGap, error)
```

`GapsForAnalyses` is the convenience wrapper used by the `gaps` command: it calls `DetectCoverage`, builds the dependency graph, computes metrics, and calls `PrioritizeGaps` in one shot.

### Coverage Detection Heuristics

`assessTestFile` classifies test content using language-agnostic string heuristics:

| Status | Detection |
|--------|-----------|
| `no_test` | Test file does not exist |
| `skeleton_only` | File exists but contains only `pass` / `// TODO` stubs with no assertions |
| `partial` | Mix of real assertions and stubs |
| `covered` | File has real assertions throughout |

Real assertions are detected by presence of `assert`, `Assert`, `expect(`, or `should.`.

### Priority Score Formula

```
score = (status_weight + 0.4 * normalised_fan_out + 0.3 * normalised_fan_in) / 1.7
```

Where `status_weight` is: `no_test=1.0`, `skeleton_only=0.6`, `partial=0.3`.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/gaps.go` | Cobra subcommand, workspace routing, dry-run |
| `internal/generation/coverage.go` | Gap detection, prioritisation, report rendering |
| `internal/analysis/graph.go` | Provides metrics for fan-in/fan-out scoring |
| `internal/analysis/pipeline.go` | `DiscoverAndAnalyzeAll` provides source analyses |
