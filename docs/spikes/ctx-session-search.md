# Spike: ctx session-search integration

**Branch**: `spike/ctx-session-search`
**Status**: open
**Contact**: Luca @ ctxrs — https://github.com/ctxrs/ctx

---

## Problem

Assay's per-file LLM context is assembled fresh every run:

```
ASSAY.md (project conventions)
+ source code
+ internal dep signatures
+ live style examples (up to 5 test files in the same dir)
```

What it lacks is **session memory** — earlier decisions about why a particular
test was structured a certain way, failed approaches the LLM should not repeat,
or naming rationale that never made it into `ASSAY.md`. That knowledge exists
in Claude Code (and other agent) transcript history, but nothing surfaces it
at generation time.

ctx (https://github.com/ctxrs/ctx) indexes local agent transcripts from 40+
tools — including Claude Code — into a Tantivy search index and exposes a Go
SDK. This spike evaluates whether pulling relevant past sessions into the
Assay LLM prompt produces measurably better test output.

---

## What ctx provides

| Feature | Detail |
|---|---|
| **Indexing** | `ctx setup` + `ctx sync` — scans local agent histories, no cloud |
| **Search** | Lexical (free); semantic (Pro, $20/mo) |
| **Go SDK** | `github.com/ctxrs/ctx/sdks/go` — zero external deps |
| **Key query axes** | free-text query, `--file`, `--workspace`, `--since`, `--provider`, session ID |
| **Primary client** | `ctxagenthistory.NewLocalClient(WithCLIPath(...), WithDataRoot(...))` |
| **Core call** | `client.Search(ctx, SearchOptions{Query: "...", Limit: 5})` |

The SDK shells out to the local `ctx` binary and parses JSON. If `ctx` is not
installed, the call returns a structured error — safe to handle gracefully.

---

## Hypothesis

Injecting a small block of prior-session context into the system prompt —
specifically decisions, constraints, and failed attempts from sessions that
touched the same file or package — reduces LLM hallucination and aligns
generated tests more closely with the project's actual conventions.

---

## Integration point

`internal/generation/pipeline.go` — `fetchBodies()`.

The natural insertion is **after** `LoadForFile` and **before** `TrimToBudget`,
as a new Priority 4 tier (first to drop when the budget is tight):

```
Tier 1 — source code         (never dropped)
Tier 2 — dep signatures      (dropped second)
Tier 3 — style snippet       (dropped third)
Tier 4 — ctx session excerpt  ← NEW, dropped first
```

ctx output goes into the **system prompt** alongside `ProjectKnowledge` so it
is budget-exempt in the same way ASSAY.md content is — OR it goes into the
user prompt as Tier 4. The spike should test both placements.

Plumbing sketch (not implementation, just shape):

```go
// internal/generation/pipeline.go

type Pipeline struct {
    driver            domain.LanguageDriver
    llm               domain.BodyGenerator
    depIndex          map[string]*domain.SourceAnalysis
    promptTokenBudget int
    ctxClient         CtxSearcher   // NEW: nil = feature disabled
}

type CtxSearcher interface {
    SearchForFile(ctx context.Context, sourcePath, projectRoot string) (string, error)
}
```

`SearchForFile` returns a short markdown block (≤300 tokens) summarising the
most relevant prior-session events. Returning an empty string disables the
feature gracefully — no code-path changes needed in the rest of the pipeline.

---

## Spike tasks

### 1. Install ctx and index the Assay project history

```bash
curl -fsSL https://ctx.rs/install | sh
ctx setup
ctx sync
ctx search "assay generate test body" --limit 5
ctx search --file internal/generation/pipeline.go --limit 5
```

Manually review the output. Does it surface useful rationale? Is the signal
worth the noise?

### 2. Wire the Go SDK in a throwaway binary

Write `cmd/assay/spike_ctx/main.go` (throwaway, gitignored) that:

1. Creates `ctxagenthistory.NewLocalClient()`.
2. Calls `Search` with a file-scoped query against a known source file.
3. Prints the raw result so we can see what the LLM would receive.

```go
package main

import (
    "context"
    "fmt"
    "log"

    ctxagenthistory "github.com/ctxrs/ctx/sdks/go"
)

func main() {
    client := ctxagenthistory.NewLocalClient()
    results, err := client.Search(context.Background(), ctxagenthistory.SearchOptions{
        Query: "test body generation pipeline",
        Limit: 5,
    })
    if err != nil {
        log.Fatalf("ctx search failed (is ctx installed?): %v", err)
    }
    for _, r := range results.Events {
        fmt.Printf("--- session %s / event %s ---\n%s\n\n", r.SessionID, r.EventID, r.Content)
    }
}
```

### 3. Measure prompt quality delta (qualitative)

Pick three source files in the Assay project. Generate tests:

- **Baseline**: current pipeline (no ctx)
- **With ctx system-prompt placement**: ctx excerpt prepended to system prompt
- **With ctx user-prompt placement**: ctx excerpt as Tier 4, budget-trimmed

Compare on three axes:
- Does the LLM avoid a naming pattern we already tried and rejected?
- Does it pick up conventions that are not yet in `ASSAY.md`?
- Does it increase or decrease prompt token cost?

### 4. Evaluate graceful-degradation path

Verify the integration is a no-op when ctx is not installed:

```go
func (c *localCtxSearcher) SearchForFile(ctx context.Context, sourcePath, root string) (string, error) {
    result, err := c.client.Search(ctx, ...)
    if err != nil {
        // ctx not installed or index empty — return empty, not an error
        return "", nil
    }
    return formatExcerpt(result), nil
}
```

Test: remove ctx binary, run `assay generate`, expect identical output to
baseline (no panic, no empty test file, no error surfaced to the user).

### 5. Token cost accounting

ctx excerpts add tokens. Measure:
- Average tokens per `SearchForFile` result (target: ≤300).
- How often Tier 4 gets trimmed at the default 6000-token budget.
- Impact on cost per file for Anthropic (claude-3-5-sonnet vs claude-3-haiku).

---

## Open questions

1. **Signal quality at scale**: Does ctx search return useful results after only
   a few sessions, or does it need a significant history to be worthwhile?

2. **System vs user prompt placement**: ProjectKnowledge is in the system prompt
   (budget-exempt, cache-friendly on Anthropic). Should ctx excerpts go there too,
   or is Tier 4 in the user prompt the right tradeoff?

3. **Semantic search requirement**: The free tier is lexical only. Will file-scoped
   lexical search be good enough, or is the $20/mo Pro semantic tier needed for
   signal quality?

4. **Watch mode benefit**: In watch mode, the ctx index is static between runs.
   Is there a meaningful benefit in watch mode, or only on cold `assay generate --all` runs?

5. **`assay learn` hook**: Could `assay learn` write a post-session ctx summary
   into `ASSAY.md` or `.assay/patterns/` automatically, making the integration
   more durable than ephemeral search?

---

## Go/No-go criteria

| Criterion | Pass |
|---|---|
| ctx SDK integrates without breaking the build | binary builds clean |
| Graceful degradation when ctx absent | `assay generate` output identical to baseline |
| Signal quality (qualitative) | ≥2 of 3 test files show a meaningful improvement |
| Token overhead | ≤300 tokens added per file on average |
| No regression in test suite | `go test ./...` green |

If all five pass, promote to a proper feature spec and wire into the pipeline.
If signal quality fails, park the spike and revisit after more session history
accumulates.

---

## Prompts for the spike sessions

Use these verbatim when running the spike so the ctx index captures the
rationale and can be searched in future sessions.

**Session A — install and index**

> Install ctx and index the Assay project's Claude Code session history.
> Run `ctx search "test body generation"` and `ctx search --file internal/generation/pipeline.go`.
> Summarise what sessions are indexed, whether they contain useful rationale,
> and whether file-scoped search returns on-topic results.
> Record your findings in docs/spikes/ctx-session-search.md under a new
> "## Session A findings" section.

**Session B — Go SDK wiring**

> In the spike/ctx-session-search branch of the Assay project, create
> cmd/assay/spike_ctx/main.go (add the path to .gitignore).
> Wire the ctxagenthistory Go SDK to search for sessions touching
> internal/generation/pipeline.go. Print the raw result.
> Then implement internal/generation/ctx_searcher.go with a CtxSearcher
> interface and a localCtxSearcher that returns empty string when ctx is
> unavailable. Do not modify pipeline.go yet.
> Record findings in docs/spikes/ctx-session-search.md under "## Session B findings".

**Session C — prompt quality experiment**

> Using the Assay project on the spike/ctx-session-search branch:
> Generate tests for three files (internal/generation/pipeline.go,
> internal/llm/generator.go, internal/projectknowledge/loader.go)
> under three conditions: baseline (no ctx), ctx in system prompt,
> ctx as user-prompt Tier 4.
> Compare naming consistency, convention adherence, and token cost.
> Record findings and a go/no-go recommendation in
> docs/spikes/ctx-session-search.md under "## Session C findings".

---

## Relationship to bundled agents

Assay ships three Claude Code agents embedded in the binary
(`internal/agents/files/`) and written into a consumer project's
`.claude/agents/` via `assay init --with-agents`:

| Agent | Role |
|---|---|
| `assay-test-author` | Runs `assay generate`, reviews scaffold, flags non-obvious patterns |
| `assay-pattern-curator` | Extracts reusable patterns into `.assay/patterns/` |
| `assay-migration-guide` | Coordinates `assay migrate` and fixes regex-rewrite edge cases |

These agents are **isolated from the ai-assistant-dot-files shared agent
library** — they ship with the binary so consumers need no extra setup, but
improvements in either repo don't flow automatically to the other.

ctx fits naturally as a pre-step for `assay-test-author`. If ctx is installed
in the consumer project, the agent could run:

```bash
ctx search --file <source-file> --limit 5
```

before calling `assay generate`, surfacing prior rationale and failed
approaches before the scaffold is even built. This is agent-side context
enrichment rather than pipeline-side — useful even if the Go SDK integration
(Session B) never ships.

**If the pipeline integration goes ahead**, the bundled agent instructions
should be updated to:

1. Check whether ctx is installed (`ctx status`).
2. If yes, run a file-scoped search and include the top result in the
   review step — not to add tokens to the LLM call, but to cross-check the
   scaffold against known prior decisions before editing.
3. If not installed, proceed as today (no change in behaviour).

Session D (optional) could evaluate this agent-side approach as an
alternative or complement to the pipeline-side Tier 4 approach.

---

## References

- ctx repo: https://github.com/ctxrs/ctx
- ctx Go SDK: https://github.com/ctxrs/ctx/tree/main/sdks/go
- ctx skill (agent instructions): https://github.com/ctxrs/ctx/blob/main/skills/ctx/SKILL.md
- Assay fetchBodies: internal/generation/pipeline.go:111
- Token budget: internal/projectknowledge/budget.go
