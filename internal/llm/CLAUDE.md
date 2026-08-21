# llm package

## Structure

```
internal/llm/
├── generator.go      — LLMBodyGenerator: BodyGenerator + BatchBodyGenerator
├── retry.go          — RetryStrategy, retryProvider (WithRetry)
├── semaphore.go      — semaphoreProvider (WithSemaphore)
├── cache.go          — ResultCache, CacheKey (sha256 keyed)
├── anthropic/        — Anthropic Messages API provider (prompt caching enabled)
├── openai/           — OpenAI Chat Completions provider
├── ollama/           — Thin wrapper around openai/ for local Ollama instances
└── factory/          — Assembles the full middleware stack from LLMConfig
```

## Provider interface
```go
type Provider interface {
    Complete(ctx context.Context, req CompletionRequest) (CompletionResponse, error)
}
```
All three backends implement this. The factory wraps them with middleware before
handing to `LLMBodyGenerator` — do not call raw providers directly.

## Middleware stack (assembled by factory.Build)
```
retry → semaphore → raw provider
```
- **retry** (`WithRetry`): exponential backoff on 429/500/502/503/504; honours `Retry-After` header; non-retryable errors (401, 403, 404) returned immediately
- **semaphore** (`WithSemaphore`): counting semaphore capped at `cfg.MaxConcurrentCalls` (default 5); prevents burst traffic when many files process in parallel
- Order matters: retry wraps semaphore so each retry attempt gets its own slot independently

Adding a new provider:
1. Create `internal/llm/<name>/provider.go` implementing `Provider`
2. Use `sharedTransport` (connection pooling) + 90s `http.Client.Timeout` — see `openai/provider.go`
3. Register in `internal/llm/factory/factory.go`
4. Middleware is applied automatically by the factory — no changes needed in provider

## LLMBodyGenerator — single-member path
`GenerateBodies(ctx, req)` checks `ResultCache` first (keyed by `CacheKey(req)`).
On cache miss: renders prompt template → calls provider → parses code block → stores in cache.

System prompt construction:
```
<ProjectKnowledge>          ← prepended when non-empty (from ASSAY.md)
You are a strict code generation assistant. Output only valid code in a markdown code block.
```

## LLMBodyGenerator — batch path (preferred)
`GenerateBatchBodies(ctx, reqs)` sends all members of a file in one API call.
- Prompt lists members numerically, includes source once, requests labeled blocks: `<!-- TEST: MemberName -->`
- `parseBatchResponse` extracts blocks by name via regex; members not found get nil CodeLines
- Each parsed result is individually cached so watch-mode hits stay granular
- `maxTokens` is scaled by member count: `g.maxTokens * len(reqs)`

The pipeline type-asserts `p.llm` to `domain.BatchBodyGenerator` — if the assertion
succeeds the batch path runs; otherwise falls back to goroutine fan-out.

## ResultCache
Thread-safe in-process cache, reset each run. Key = sha256 of:
`language + memberKind + memberName + sourceCode + modulePath + depsSignatures + styleSnippet + frameworkName + projectKnowledge`
Null-byte-separated to prevent cross-field collisions.
`Stats()` returns (hits, misses, size) for verbose/debug logging.

## Anthropic provider — prompt caching
Uses structured content arrays with `cache_control: {type: "ephemeral"}` on both
the system prompt block and the user message block.
Header: `anthropic-beta: prompt-caching-2024-07-31`.
`TokensUsed` includes `cache_read_input_tokens` + `cache_creation_input_tokens`.
Cache TTL is 5 minutes on Anthropic's side — effective for watch mode and `--all` runs.

## RetryStrategy defaults (set in factory.Build)
```go
RetryStrategy{MaxAttempts: 3, BaseDelay: 500ms, MaxDelay: 30s, Multiplier: 2.0}
```
Configurable via `LLMConfig.MaxRetryAttempts`. `MaxAttempts=1` returns the inner
provider unwrapped (no overhead).

## factory.Build(cfg, driver)
1. Resolves API key from `cfg.APIKeyEnvVar`
2. Builds raw provider (`anthropic` / `openai` / `ollama`)
3. Wraps: `WithRetry(WithSemaphore(raw, cfg.MaxConcurrentCalls), strategy)`
4. Returns `llm.New(wrapped, prompts, model, maxTokens, temperature)`
