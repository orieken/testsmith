# domain package — read this before any other internal package

This package is the dependency inversion boundary. Every interface lives here.
No other internal package is imported from here — only the standard library.

## Key interfaces

| Interface | Implemented by | Used by |
|---|---|---|
| `LanguageDriver` | `internal/drivers/{lang}/driver.go` | analysis pipeline, generation pipeline, CLI |
| `BodyGenerator` | `internal/llm/generator.go` (`LLMBodyGenerator`) | `generation.Pipeline.fetchBodies` |
| `BatchBodyGenerator` | `internal/llm/generator.go` (`LLMBodyGenerator`) | `fetchBodies` — preferred path; falls back to `BodyGenerator` fan-out when absent |
| `TestAdapter` | `internal/drivers/{lang}/adapters.go` | drivers internally |

`BatchBodyGenerator` embeds `BodyGenerator`. Implement only `BodyGenerator` for test stubs — `fetchBodies` handles the fallback automatically via a type assertion.

## Key types and what fills them

| Type | Filled by | Consumed by |
|---|---|---|
| `ProjectContext` | `driver.DetectProject()` | everything downstream |
| `ProjectContext.ProjectKnowledge` | `projectknowledge.Load()` in `cmd/assay/generate.go` | `generation.Pipeline.fetchBodies` → LLM system prompt |
| `SourceAnalysis` | `driver.AnalyzeFile()` | `generation.Pipeline.Plan()` |
| `GenerationPlan` | `generation.Pipeline.Plan()` | `generation.Executor.Execute()` |
| `GeneratedFile.Language` | `Plan()` — set on every `RoleTestFile` | `Executor.verifyFile()` → selects the right `Verifier` |
| `GenerateOpts.TestFileKnownNew` | CLI when files come from `DiscoverUntested` | `resolveAction` in generation pipeline — skips redundant `os.Stat` |

## Adding a field to a struct
1. Add to the struct here in `types.go`
2. If it belongs in LLM prompts: mirror in both `BodyGenRequest` AND `BodyPromptData`
3. Update `internal/llm/generator.go` `data` construction to pass it through
4. If it affects cache keys: update `CacheKey()` in `internal/llm/cache.go`
5. Update `internal/generation/pipeline.go` `fetchBodies` to populate it

## MemberKind values
`KindFunction`, `KindMethod`, `KindClass`, `KindStruct`, `KindInterface`
Drivers set these; prompt templates reference them via `{{.MemberKind}}`.

## GeneratedFile.Language
Must be set whenever `Role == RoleTestFile`. The Executor uses it to look up a
`Verifier` — if empty, compile verification is silently skipped.
