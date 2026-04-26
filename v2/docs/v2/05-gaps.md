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
  --output <file>   Output Markdown file (default: testsmith_coverage_report.md)
  --top <n>         Show only the top N gaps (default: show all)
```

---

## Example Report

```markdown
# TestSmith Coverage Gap Report

| Priority | File | Status | Ext Deps | Dependents | Command |
|----------|------|--------|----------|------------|---------|
| 0.95 | src/services/payment.py | no_test | 3 | 4 | testsmith generate src/services/payment.py |
| 0.82 | src/core/auth.py | skeleton_only | 2 | 6 | testsmith generate src/core/auth.py |
| 0.61 | src/utils/retry.py | partial | 1 | 3 | testsmith generate --overwrite src/utils/retry.py |
```

---

## Go Implementation

```go
// internal/generation/coverage.go
func DetectCoverage(
    analyses []*domain.SourceAnalysis,
    testRoot string,
    driver domain.LanguageDriver,
) []domain.CoverageGap

func PrioritizeGaps(gaps []domain.CoverageGap, metrics []domain.ModuleMetrics) []domain.CoverageGap

func GenerateReport(gaps []domain.CoverageGap) string
```

### Coverage Detection Heuristics

`skeleton_only` is detected by:
- All test functions in the file contain only `pass` / `// TODO` / empty body
- The heuristic is language-specific; each driver implements `IsSkeletonTest(content string) bool`

`partial` is detected by:
- Some test functions have real assertion bodies
- Some test functions are still stubs

### Priority Score Formula

```
score = (status_weight + 0.4 * normalised_fan_out + 0.3 * normalised_fan_in) / 1.7
```

Where `status_weight` is: `no_test=1.0`, `skeleton_only=0.6`, `partial=0.3`.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/gaps.go` | Cobra subcommand |
| `internal/generation/coverage.go` | Gap detection, prioritisation, report rendering |
| `internal/analysis/graph.go` | Provides metrics for fan-in/fan-out |
| `internal/analysis/pipeline.go` | `DiscoverAndAnalyzeAll` provides source analyses |
