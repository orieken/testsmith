# Assay — Agent Context

## What this project is
Assay is a Go CLI that generates test scaffolds for source files in any language.
It is language-agnostic: a driver plugin model decouples file analysis, code generation,
and LLM prompting from any particular language or framework.

## Package map — read this before touching any file

| Package | Purpose | Read first |
|---|---|---|
| `internal/domain` | All interfaces and shared types. Nothing else imports from outer layers into this package. | Always |
| `internal/generation` | `Pipeline` builds `GenerationPlan`; `Executor` writes and verifies files | When changing generation logic |
| `internal/analysis` | File discovery (`DiscoverUntested`, `DiscoverInPath`) and `SourceAnalysis` construction | When changing discovery |
| `internal/drivers/{lang}` | Language-specific analysis + code generation; one package per language | When adding/changing a language |
| `internal/llm` | `LLMBodyGenerator`, provider middleware (retry/semaphore/cache), batch generation | When changing LLM integration |
| `internal/projectknowledge` | Reads `ASSAY.md`, token budget management, convention mining | When changing context loading |
| `internal/config` | Config schema + defaults — `LLMConfig` fields live here | When changing config |
| `cmd/assay` | Cobra CLI — one file per subcommand (`generate.go`, `init.go`, etc.) | When changing CLI |

## Dependency direction (hard constraint)
```
cmd → internal/generation → internal/domain ← internal/drivers
                          ← internal/llm
                          ← internal/projectknowledge
```
`internal/domain` NEVER imports from any other internal package.
`internal/drivers/*` NEVER imports from `internal/generation` or `internal/llm`.

## Key data flow for `generate --llm`
```
DetectProject()           → ProjectContext  (ProjectKnowledge loaded from ASSAY.md)
DiscoverUntested()        → []string        (opts.TestFileKnownNew=true set here)
AnalyzeFile()             → SourceAnalysis
genPipeline.Plan()
  └─ fetchBodies()
       ├─ projectknowledge.LoadForFile()    (per-file ASSAY.md merge)
       ├─ buildDepsSignatures()             (internal dep public API)
       ├─ mineConventions()                (up to 5 test files in same dir)
       ├─ TrimToBudget()                   (drops low-priority tiers at token limit)
       ├─ [batch path]  BatchBodyGenerator.GenerateBatchBodies()  ← 1 API call for all members
       └─ [fallback]    goroutine fan-out  GenerateBodies()       ← 1 call per member
            ↑ both paths go through: retry → semaphore → raw provider
            ↑ and both check ResultCache before hitting the provider
driver.GenerateTestFile() → GeneratedFile  (Language field set here)
executor.Execute()        → writes file
  └─ verifyFile()         → go vet / tsc --noEmit / py_compile (language-specific)
```

## LLMConfig fields (internal/config/schema.go)
| Field | Default | Purpose |
|---|---|---|
| `PromptTokenBudget` | 6000 | Max user-prompt tokens; lower-priority tiers dropped first |
| `MaxConcurrentCalls` | 5 | Semaphore cap on in-flight API calls |
| `MaxRetryAttempts` | 3 | Attempts per call (1 = no retry); exponential backoff |

## Invariants agents must not break
- `internal/domain` has no internal imports
- Every `LanguageDriver` must implement ALL methods in `domain.LanguageDriver`
- Every `TestAdapter` must implement `LLMVocabulary() map[string]string`
- `resolveAction` third param `knownNew` must be `false` for fixture files, `true` when caller used `DiscoverUntested`
- `GeneratedFile.Language` must be set in `Plan()` — `Executor` uses it to select the right `Verifier`
- Adding fields to `BodyGenRequest` requires mirroring in `BodyPromptData` and `generator.go`'s `data` construction
- Retry middleware must not be bypassed — all providers are wrapped in `factory.Build`

## Middleware stack (factory-assembled, all providers)
```
retry → semaphore → raw provider
```
Each retry attempt acquires its own semaphore slot to prevent slot starvation.

## Adding a new language driver
Copy `internal/drivers/golang/` as a template. Required files:
`driver.go`, `detector.go`, `analyzer.go`, `generator.go`, `adapters.go`, `migrators.go`
Register in `internal/registry/registry.go`.
Consider adding a `Verifier` in `internal/generation/verify.go` for compile checking.

## Common mistakes
- Forgetting to guard nil `extra` map after `LLMContext()` — always `if extra == nil { extra = make(...) }`
- Calling `resolveAction` without `knownNew` correctly — fixture files always `false`
- Forgetting `GeneratedFile.Language` — Executor silently skips verification when it's empty
- Implementing `BodyGenerator` but not `BatchBodyGenerator` — fine, but the pipeline falls back to per-member goroutines
