package llm

import (
	"fmt"
	"strings"
	"sync/atomic"
)

// TokenReport accumulates LLM token usage across a run. All fields are
// updated atomically so it is safe to share across goroutines.
type TokenReport struct {
	Calls        atomic.Int64
	InputTokens  atomic.Int64
	OutputTokens atomic.Int64
	TotalTokens  atomic.Int64
}

// add records the token counts from one CompletionResponse.
func (r *TokenReport) add(resp CompletionResponse) {
	r.Calls.Add(1)
	r.InputTokens.Add(int64(resp.InputTokens))
	r.OutputTokens.Add(int64(resp.OutputTokens))
	r.TotalTokens.Add(int64(resp.TokensUsed))
}

// EstimateCost returns a rough USD cost estimate based on published per-model
// pricing. Returns 0 for models not in the pricing table (e.g. local Ollama).
// Prices are per 1 million tokens as of mid-2025.
func (r *TokenReport) EstimateCost(model string) float64 {
	p, ok := modelPricing[normaliseModel(model)]
	if !ok {
		return 0
	}
	in := float64(r.InputTokens.Load()) / 1_000_000 * p.inputPerM
	out := float64(r.OutputTokens.Load()) / 1_000_000 * p.outputPerM
	return in + out
}

// Summary returns a human-readable one-liner for CLI output.
func (r *TokenReport) Summary(model string) string {
	calls := r.Calls.Load()
	total := r.TotalTokens.Load()
	in := r.InputTokens.Load()
	out := r.OutputTokens.Load()
	cost := r.EstimateCost(model)

	var sb strings.Builder
	fmt.Fprintf(&sb, "LLM usage — calls: %d  tokens: %s", calls, formatTokens(total))
	if in > 0 || out > 0 {
		fmt.Fprintf(&sb, " (in: %s  out: %s)", formatTokens(in), formatTokens(out))
	}
	if cost > 0 {
		fmt.Fprintf(&sb, "  est. cost: $%.4f", cost)
	}
	return sb.String()
}

// ── pricing table ─────────────────────────────────────────────────────────────

type modelPrice struct {
	inputPerM  float64 // USD per 1 million input tokens
	outputPerM float64 // USD per 1 million output tokens
}

// modelPricing maps normalised model identifiers to their published prices.
// Prices are approximate mid-2025 list rates; update when providers change them.
var modelPricing = map[string]modelPrice{
	// Anthropic — Claude 5 family
	"claude-opus-5":   {15.00, 75.00},
	"claude-sonnet-5": {3.00, 15.00},
	"claude-fable-5":  {3.00, 15.00},

	// Anthropic — Claude Sonnet 4.x
	"claude-sonnet-4-6":          {3.00, 15.00},
	"claude-sonnet-4-5":          {3.00, 15.00},
	"claude-sonnet-4-5-20251001": {3.00, 15.00},
	"claude-sonnet-4-6-20250514": {3.00, 15.00},

	// Anthropic — Haiku
	"claude-haiku-4-5":          {0.80, 4.00},
	"claude-haiku-4-5-20251001": {0.80, 4.00},
	"claude-3-5-haiku-20241022": {0.80, 4.00},

	// OpenAI — GPT-4o family
	"gpt-4o":            {2.50, 10.00},
	"gpt-4o-mini":       {0.15, 0.60},
	"gpt-4o-2024-11-20": {2.50, 10.00},

	// OpenAI — o-series reasoning models
	"o1":      {15.00, 60.00},
	"o1-mini": {3.00, 12.00},
	"o3":      {10.00, 40.00},
	"o3-mini": {1.10, 4.40},
	"o4-mini": {1.10, 4.40},

	// OpenAI — GPT-4 legacy
	"gpt-4-turbo":      {10.00, 30.00},
	"gpt-4-turbo-2024": {10.00, 30.00},
}

// normaliseModel strips date suffixes from model IDs so that
// "claude-sonnet-4-6-20250514" matches "claude-sonnet-4-6".
func normaliseModel(model string) string {
	if _, ok := modelPricing[model]; ok {
		return model
	}
	// Try progressively shorter dash-separated prefixes.
	parts := strings.Split(model, "-")
	for i := len(parts) - 1; i >= 2; i-- {
		candidate := strings.Join(parts[:i], "-")
		if _, ok := modelPricing[candidate]; ok {
			return candidate
		}
	}
	return model
}

func formatTokens(n int64) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.2fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fk", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
