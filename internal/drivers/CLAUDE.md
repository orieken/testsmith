# drivers — language plugin layer

Each sub-package (`python`, `typescript`, `golang`, `java`, `csharp`) implements
`domain.LanguageDriver`. All five follow the same file layout:

| File | Responsibility |
|---|---|
| `driver.go` | Struct, interface delegation, body prompt constant, `LLMContext()` |
| `detector.go` | `DetectProject()` — finds project root, reads build files |
| `analyzer.go` | `AnalyzeFile()` — AST parsing → `SourceAnalysis` |
| `generator.go` | `GenerateTestFile()` — renders test file template |
| `adapters.go` | `TestAdapter` implementations + `registry` + `selectAdapter()` |
| `migrators.go` | `Migrator` implementations (framework migration support) |

## Adding a new adapter (e.g. a new test framework variant)

1. Add a struct in `adapters.go` implementing `domain.TestAdapter`:
   ```go
   type myAdapter struct{}
   func (a myAdapter) Name() string { return "myframework" }
   func (a myAdapter) LLMVocabulary() map[string]string {
       return map[string]string{
           "framework":    "myframework",
           "mock_style":   "...",
           "assert_style": "...",
       }
   }
   // ... other interface methods
   ```
2. Register it in the language's `registry` (at bottom of `adapters.go`)
3. `SelectFromContext()` auto-selects based on `ProjectContext.Metadata["framework"]`

## LLMContext() contract
```go
func (d *Driver) LLMContext(ctx *domain.ProjectContext) map[string]string {
    vocab := registry.SelectFromContext(ctx).LLMVocabulary()
    vocab["language"] = "go" // always add language
    return vocab
}
```
**Never return nil** — the caller guards for nil but returning a real map is cleaner.

## Body prompt template
The constant in `driver.go` is a `text/template` string rendered against `domain.BodyPromptData`.
Available fields: `{{.MemberName}}`, `{{.MemberKind}}`, `{{.SourceCode}}`, `{{.ModulePath}}`,
`{{.DepsSignatures}}`, `{{.ExistingTestSnippet}}`, `{{index .Extra "framework"}}`, etc.
Use `{{- if .DepsSignatures}}...{{- end}}` guards for optional sections.

## Adding a new language driver
Copy `golang/` as a template — it is the simplest driver.
Register in `internal/registry/registry.go` by adding to the `drivers` slice.

Consider also adding a `Verifier` case in `internal/generation/verify.go`
`VerifierFor()` so generated test files are compile-checked after writing.
