package projectknowledge_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/projectknowledge"
)

func TestEstimateTokens(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  int
	}{
		{name: "empty string returns 0", input: "", want: 0},
		{name: "exactly 4 chars is 1 token", input: "abcd", want: 1},
		{name: "5 chars rounds up to 2 tokens", input: "abcde", want: 2},
		{name: "3 chars rounds up to 1 token", input: "abc", want: 1},
		{name: "8 chars is 2 tokens", input: "abcdefgh", want: 2},
		{name: "100 chars is 25 tokens", input: strings.Repeat("x", 100), want: 25},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := projectknowledge.EstimateTokens(tt.input); got != tt.want {
				t.Errorf("EstimateTokens(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

func TestTrimToBudget(t *testing.T) {
	t.Parallel()

	// Each content string is 8 chars = 2 tokens.
	tierA := projectknowledge.Tier{Name: "source", Content: "aaaaaaaa", Priority: 1}
	tierB := projectknowledge.Tier{Name: "deps", Content: "bbbbbbbb", Priority: 2}
	tierC := projectknowledge.Tier{Name: "style", Content: "cccccccc", Priority: 3}
	emptyTier := projectknowledge.Tier{Name: "empty", Content: "", Priority: 4}

	tests := []struct {
		name         string
		tiers        []projectknowledge.Tier
		budgetTokens int
		want         []string
	}{
		{
			name:         "zero budget returns all tiers unchanged",
			tiers:        []projectknowledge.Tier{tierA, tierB, tierC},
			budgetTokens: 0,
			want:         []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"},
		},
		{
			name:         "budget fits all tiers",
			tiers:        []projectknowledge.Tier{tierA, tierB, tierC},
			budgetTokens: 6,
			want:         []string{"aaaaaaaa", "bbbbbbbb", "cccccccc"},
		},
		{
			name:         "budget fits only two highest-priority tiers",
			tiers:        []projectknowledge.Tier{tierA, tierB, tierC},
			budgetTokens: 4,
			want:         []string{"aaaaaaaa", "bbbbbbbb", ""},
		},
		{
			name:         "very tight budget keeps only priority-1 tier",
			tiers:        []projectknowledge.Tier{tierA, tierB, tierC},
			budgetTokens: 2,
			want:         []string{"aaaaaaaa", "", ""},
		},
		{
			name:         "budget of 1 token drops all non-empty tiers",
			tiers:        []projectknowledge.Tier{tierA, tierB, tierC},
			budgetTokens: 1,
			want:         []string{"", "", ""},
		},
		{
			name:         "empty-content tier is always kept at zero cost",
			tiers:        []projectknowledge.Tier{tierA, emptyTier},
			budgetTokens: 1,
			want:         []string{"", ""},
		},
		{
			name:         "nil input returns empty slice",
			tiers:        nil,
			budgetTokens: 100,
			want:         []string{},
		},
		{
			name:         "output preserves original order regardless of priority",
			tiers:        []projectknowledge.Tier{tierC, tierA, tierB},
			budgetTokens: 4, // fits A (p1) + B (p2); C (p3) dropped
			want:         []string{"", "aaaaaaaa", "bbbbbbbb"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := projectknowledge.TrimToBudget(tt.tiers, tt.budgetTokens)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("TrimToBudget() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJoinTiers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		contents []string
		sep      string
		want     string
	}{
		{
			name:     "joins non-empty contents with separator",
			contents: []string{"a", "b", "c"},
			sep:      "\n---\n",
			want:     "a\n---\nb\n---\nc",
		},
		{
			name:     "skips empty strings",
			contents: []string{"a", "", "c"},
			sep:      "\n",
			want:     "a\nc",
		},
		{
			name:     "skips whitespace-only strings",
			contents: []string{"a", "   ", "c"},
			sep:      "\n",
			want:     "a\nc",
		},
		{
			name:     "all empty returns empty string",
			contents: []string{"", "", ""},
			sep:      "\n",
			want:     "",
		},
		{
			name:     "nil input returns empty string",
			contents: nil,
			sep:      "\n",
			want:     "",
		},
		{
			name:     "single non-empty entry returns it without separator",
			contents: []string{"only"},
			sep:      "\n---\n",
			want:     "only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := projectknowledge.JoinTiers(tt.contents, tt.sep); got != tt.want {
				t.Errorf("JoinTiers() = %q, want %q", got, tt.want)
			}
		})
	}
}
