# Feature: prune — Remove Unused Fixture Files

## What It Does

`testsmith prune` scans all source files to build a list of external dependencies currently in use, then compares that list against every fixture file in the fixture directory. Any fixture whose dependency is no longer imported anywhere in the project is flagged as unused.

By default prune runs as a **dry-run** — it prints what would be deleted without touching anything. Passing `--confirm` performs the deletion and updates any test files that imported the removed fixtures (commenting out the now-dead import lines).

---

## CLI

```
testsmith prune [flags]

Flags:
  --confirm     Actually delete unused fixtures (default: dry-run)
  --verbose     List the source files checked for each dependency
```

---

## Example Output

```
TestSmith Prune Summary
───────────────────────
Unused fixtures found: 2

  ✗ tests/fixtures/sendgrid_fixture.py — no source files import sendgrid
  ✗ tests/fixtures/boto3_fixture.py    — no source files import boto3

Run with --prune --confirm to delete these fixtures.
```

After `--confirm`:

```
Deleted fixtures:
  ✓ sendgrid_fixture.py
  ✓ boto3_fixture.py

Updated 3 test file(s) to comment out deleted fixture imports.
```

---

## Go Implementation

```go
// internal/generation/prune.go
func ScanUsedDependencies(analyses []*domain.SourceAnalysis) map[string]bool
func ScanExistingFixtures(fixtureDir string, cfg *config.Config) []FixtureFile
func IdentifyUnused(used map[string]bool, existing []FixtureFile) []FixtureFile
func PruneFixtures(unused []FixtureFile, dryRun bool) ([]PruneResult, error)
func UpdateTestImports(root string, deletedNames []string) ([]string, error)
```

### FixtureFile

```go
type FixtureFile struct {
    AbsPath string
    DepName string // root dependency name this fixture mocks
}
```

### Driver Integration

`ScanExistingFixtures` delegates to the language driver to determine the fixture directory and naming convention. The Go driver returns an empty list since it has no fixture directory.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/prune.go` | Cobra subcommand |
| `internal/generation/prune.go` | Prune logic |
| `internal/analysis/pipeline.go` | `DiscoverAndAnalyzeAll` provides dependency sets |
| `internal/fsutil/write.go` | Safe file deletion |
