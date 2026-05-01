# Feature: prune — Remove Unused Fixture Files

## What It Does

`testsmith prune` scans all source files to build a list of external dependencies currently in use, then compares that list against every fixture file in the fixture directory. Any fixture whose dependency is no longer imported anywhere in the project is flagged as unused.

By default prune runs as a **dry-run** — it prints what would be deleted without touching anything. Passing `--confirm` performs the deletion and updates any test files that imported the removed fixtures (commenting out the now-dead import lines).

---

## CLI

```
testsmith prune [flags]

Flags:
  --confirm           Actually delete unused fixtures (default: dry-run / list only)
  --workspace <name>  Prune only this workspace (name or path)
  --verbose, -v       List the source files checked for each dependency
```

### Workspace Mode

When `workspaces:` are configured, each workspace is pruned independently with its own fixture directory resolved from the driver's `TestFrameworkConfig.FixtureDir`. Use `--workspace <name>` to limit to one workspace.

---

## Example Output

```
  · would delete  sendgrid_fixture
  · would delete  boto3_fixture

  2 unused fixture(s) found. Run with --confirm to delete.
```

After `--confirm`:

```
  ✓ deleted  sendgrid_fixture
  ✓ deleted  boto3_fixture
  · commented out stale imports in tests/test_payment.py

  Pruned 2 fixture(s).
```

---

## Go Implementation

```go
// internal/generation/prune.go
func ScanUsedDependencies(analyses []*domain.SourceAnalysis) map[string]bool
func ScanExistingFixtures(fixtureDir string, cfg domain.TestFrameworkConfig) ([]FixtureFile, error)
func IdentifyUnused(used map[string]bool, existing []FixtureFile) []FixtureFile
func PruneFixtures(unused []FixtureFile, dryRun bool) []PruneResult
func UpdateTestImports(root string, deletedNames []string) ([]string, error)
```

### FixtureFile / PruneResult

```go
type FixtureFile struct {
    AbsPath string
    DepName string // root dependency name this fixture mocks
}

type PruneResult struct {
    DepName string
    Action  string // "deleted" | "skipped" | "error"
    Err     error
}
```

### Fixture Directory Resolution

The fixture directory is resolved from the driver's `TestFrameworkConfig.FixtureDir` field, joined against the workspace or project root. Drivers that have no fixture directory (Go) return `""` and `ScanExistingFixtures` returns nil without error.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/prune.go` | Cobra subcommand, workspace routing |
| `internal/generation/prune.go` | `ScanUsedDependencies`, `ScanExistingFixtures`, prune + import cleanup |
| `internal/analysis/pipeline.go` | `DiscoverAndAnalyzeAll` provides dependency sets |
