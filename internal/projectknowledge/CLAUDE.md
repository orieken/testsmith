# projectknowledge package

Reads and merges `TESTSMITH.md` knowledge files from a target project, manages
the LLM prompt token budget, and mines test-file conventions. This package has
no domain imports — it depends only on the standard library.

## Files

| File | Responsibility |
|---|---|
| `loader.go` | `Load`, `LoadForFile`, `LoadForDir`, `Template` |
| `budget.go` | `Tier`, `EstimateTokens`, `TrimToBudget`, `JoinTiers` |

## TESTSMITH.md hierarchy

```
<project-root>/TESTSMITH.md       ← always loaded; project-wide conventions
<source-dir>/TESTSMITH.md         ← merged below root when present (package overrides)
```

- `Load(root)` — reads root-level file only
- `LoadForFile(sourcePath, root)` — merges root + source directory
- `LoadForDir(dir, root)` — same but takes a directory path directly (used for workspaces)

When both files exist, directory content is appended under `## Package-level conventions`.
When neither exists, returns empty string — callers must handle this gracefully.

## Token budget

`TrimToBudget(tiers []Tier, budgetTokens int)` returns a `[]string` (one per tier)
where lower-priority tiers are replaced with `""` when the total estimated token
count would exceed `budgetTokens`. `budgetTokens=0` disables trimming.

Priority convention used in `fetchBodies`:
- 1 = source code (never dropped)
- 2 = dep signatures (dropped second)
- 3 = style snippet (dropped first)

`EstimateTokens(s string)` uses the 4-chars-per-token heuristic — good enough for
budget decisions, not a billing estimate.

## Template(language)
Returns a language-appropriate starter `TESTSMITH.md` as a string.
Languages with specific templates: `go`, `python`, `typescript`, `java`, `csharp`.
Falls back to `"default"` for unknown languages.
Called by `testsmith init` to scaffold the file alongside `.testsmith.yaml`.

## Usage pattern in pipeline.go
```go
projectKnow := projectknowledge.LoadForFile(analysis.SourcePath, analysis.Project.Root)
if projectKnow == "" {
    projectKnow = analysis.Project.ProjectKnowledge // fall back to run-level load
}
```
`analysis.Project.ProjectKnowledge` is loaded once in `runGenerate` via `Load(ctx.Root)`
and acts as the run-level cache. Per-file loading allows package-level overrides without
re-reading the root file on every call.
