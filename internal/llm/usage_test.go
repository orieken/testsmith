package llm_test

import (
	"context"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/llm"
)

// inputOutputProvider returns responses with explicit input/output token counts.
// Unlike the stubProvider in generator_test.go, it sets InputTokens/OutputTokens.
type inputOutputProvider struct {
	input  int
	output int
}

func (p *inputOutputProvider) Complete(_ context.Context, _ llm.CompletionRequest) (llm.CompletionResponse, error) {
	return llm.CompletionResponse{
		Content:      "```go\nfunc TestX(t *testing.T){}\n```",
		InputTokens:  p.input,
		OutputTokens: p.output,
		TokensUsed:   p.input + p.output,
	}, nil
}

func newBodyReq(name string) domain.BodyGenRequest {
	return domain.BodyGenRequest{
		Language:   "go",
		MemberName: name,
		MemberKind: domain.KindFunction,
		SourceCode: "func " + name + "() {}",
	}
}

// ── TokenReport accumulation ──────────────────────────────────────────────────

func TestUsage_AccumulatesAcrossCalls(t *testing.T) {
	gen := llm.New(&stubProvider{content: "```go\nfunc TestFoo(t *testing.T){}\n```", tokensUsed: 30},
		nil, "claude-sonnet-4-6", 100, 0.0)

	// Three distinct requests so the cache doesn't suppress provider calls.
	for _, name := range []string{"Alpha", "Beta", "Gamma"} {
		if _, err := gen.GenerateBodies(context.Background(), newBodyReq(name)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	usage := gen.Usage()
	if got := usage.Calls.Load(); got != 3 {
		t.Errorf("Calls = %d, want 3", got)
	}
	if got := usage.TotalTokens.Load(); got != 90 {
		t.Errorf("TotalTokens = %d, want 90", got)
	}
}

func TestUsage_InputOutputTokensSeparate(t *testing.T) {
	gen := llm.New(&inputOutputProvider{input: 10, output: 5},
		nil, "claude-sonnet-4-6", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("Bar")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	usage := gen.Usage()
	if got := usage.InputTokens.Load(); got != 10 {
		t.Errorf("InputTokens = %d, want 10", got)
	}
	if got := usage.OutputTokens.Load(); got != 5 {
		t.Errorf("OutputTokens = %d, want 5", got)
	}
}

// ── EstimateCost ──────────────────────────────────────────────────────────────

func TestEstimateCost_KnownModel(t *testing.T) {
	gen := llm.New(&inputOutputProvider{input: 1_000_000, output: 1_000_000},
		nil, "claude-sonnet-4-6", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("Baz")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// claude-sonnet-4-6: $3.00/M in + $15.00/M out = $18.00 for 1M each
	cost := gen.Usage().EstimateCost("claude-sonnet-4-6")
	if cost < 17.9 || cost > 18.1 {
		t.Errorf("EstimateCost = %.4f, want ~18.00", cost)
	}
}

func TestEstimateCost_UnknownModel_ReturnsZero(t *testing.T) {
	gen := llm.New(&inputOutputProvider{input: 100, output: 50},
		nil, "my-local-model", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("Qux")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cost := gen.Usage().EstimateCost("my-local-model"); cost != 0 {
		t.Errorf("EstimateCost unknown = %f, want 0", cost)
	}
}

func TestEstimateCost_DateSuffixNormalised(t *testing.T) {
	// "claude-sonnet-4-6-20250514" should normalise to "claude-sonnet-4-6"
	gen := llm.New(&inputOutputProvider{input: 1_000_000, output: 1_000_000},
		nil, "claude-sonnet-4-6-20250514", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("DateModel")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	cost := gen.Usage().EstimateCost("claude-sonnet-4-6-20250514")
	if cost < 17.9 || cost > 18.1 {
		t.Errorf("EstimateCost with date suffix = %.4f, want ~18.00", cost)
	}
}

// ── Summary ───────────────────────────────────────────────────────────────────

func TestSummary_ContainsKeyFields(t *testing.T) {
	gen := llm.New(&inputOutputProvider{input: 500, output: 200},
		nil, "gpt-4o", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("Quux")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	s := gen.Usage().Summary("gpt-4o")
	for _, want := range []string{"calls:", "tokens:", "cost:"} {
		if !strings.Contains(s, want) {
			t.Errorf("Summary() missing %q: %q", want, s)
		}
	}
}

func TestSummary_ZeroCalls(t *testing.T) {
	gen := llm.New(&inputOutputProvider{}, nil, "claude-sonnet-4-6", 100, 0.0)
	s := gen.Usage().Summary("claude-sonnet-4-6")
	if !strings.Contains(s, "calls: 0") {
		t.Errorf("Summary() zero-call line missing 'calls: 0': %q", s)
	}
}

// ── UsageSummary method ───────────────────────────────────────────────────────

func TestUsageSummary_DelegatesToReport(t *testing.T) {
	gen := llm.New(&inputOutputProvider{input: 100, output: 50},
		nil, "gpt-4o-mini", 100, 0.0)

	if _, err := gen.GenerateBodies(context.Background(), newBodyReq("Delegate")); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if s := gen.UsageSummary("gpt-4o-mini"); s == "" {
		t.Error("UsageSummary() returned empty string")
	}
}
