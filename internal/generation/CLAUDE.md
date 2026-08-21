# generation package

## Files

| File | Responsibility |
|---|---|
| `pipeline.go` | `Pipeline` — `Plan()` builds `GenerationPlan`; `fetchBodies()` calls LLM |
| `executor.go` | `Executor.Execute()` — writes files; `verifyFile()` runs compile check post-write |
| `verify.go` | `Verifier` interface; `GoVerifier`, `TypeScriptVerifier`, `PythonVerifier`; `VerifierFor(lang)` |
| `coverage.go` | `DetectCoverage()` — classifies test coverage status per source file |
| `prune.go` | `ScanExistingFixtures()` — finds fixture files for the prune command |

## Pipeline data flow

```
Plan(ctx, analysis, opts)
├─ fetchBodies()          ← only when llm != nil && !opts.DryRun
│   ├─ LLMContext()             ← adapter vocabulary (framework, mock_style, ...)
│   ├─ LoadForFile()            ← ASSAY.md (package-level merge)
│   ├─ buildDepsSignatures()    ← internal dep public API from depIndex
│   ├─ mineConventions()        ← up to 5 test files in source dir (80-line cap)
│   ├─ TrimToBudget()           ← drops low-priority tiers at promptTokenBudget
│   ├─ [batch]   BatchBodyGenerator.GenerateBatchBodies(reqs) ← type asserted
│   └─ [fallback] goroutine fan-out GenerateBodies() per member
├─ GenerateFixture()      ← per external dep root module
├─ GenerateTestFile()     → sets GeneratedFile.Language
└─ GenerateBootstrap()    ← conftest.py / jest.setup.ts / etc.

executor.Execute()
├─ writeFile()            ← atomic rename via .tmp
└─ verifyFile()           ← only for RoleTestFile after successful write
     └─ VerifierFor(language) → GoVerifier / TypeScriptVerifier / PythonVerifier / nil
```

## Executor: verification

`NewVerifiedExecutor(language)` is the standard constructor — call this from the CLI,
not `&Executor{}` directly. It pre-loads the correct `Verifier` for the project language.

Verification errors are surfaced in `Result.Err` and printed as `✗` by the CLI.
The generated file is kept on disk so the user can inspect and fix it.

Languages with compile checks: `go` (`go vet`), `typescript` (`tsc --noEmit`), `python` (`py_compile`).
Java and C# return `nil` — their build toolchains are not guaranteed to be present.
`tsc` and Python interpreters are skipped silently when not installed.

## resolveAction(path, overwrite, knownNew)
- `knownNew=true` → skip `os.Stat`, return `ActionCreate` immediately
- Set `knownNew=true` only for files from `DiscoverUntested` / `DiscoverInPath`
- Fixture files always use `knownNew=false`

## Token budget tiers (priority order, in TrimToBudget)
| Priority | Tier | Dropped when? |
|---|---|---|
| 1 (keep) | Source code | Never |
| 2 | Dep signatures | Budget exceeded after source |
| 3 | Style snippet | Budget exceeded after deps |

`ProjectKnowledge` (ASSAY.md) goes into the **system prompt**, not the user prompt,
so it is never subject to the `PromptTokenBudget` trim.

## Batch vs fan-out path
`fetchBodies` type-asserts `p.llm` to `domain.BatchBodyGenerator`. If the assertion
succeeds (i.e. the generator is `*llm.LLMBodyGenerator`), one API call handles all
members. If not (e.g. test stubs), each member gets its own goroutine. The fan-out
path is the fallback — do not remove it.

## Pipeline builder pattern
```go
gen := generation.NewPipeline(driver, bodyGen).
    WithDepIndex(idx).
    WithPromptTokenBudget(cfg.LLM.PromptTokenBudget)
```
