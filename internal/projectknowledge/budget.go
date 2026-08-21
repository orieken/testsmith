package projectknowledge

import (
	"sort"
	"strings"
)

// Tier is a named, prioritised block of content destined for an LLM prompt.
// Lower Priority values are more important and will be kept when trimming.
type Tier struct {
	Name     string
	Content  string
	Priority int // 1 = must-keep, higher = drop first
}

// EstimateTokens returns a rough token count using the common 4-chars-per-token
// heuristic. Accurate enough for budget decisions; not a billing estimate.
func EstimateTokens(s string) int {
	if s == "" {
		return 0
	}
	return (len(s) + 3) / 4 // ceiling division
}

// TrimToBudget returns the Content of each tier that fits within budgetTokens,
// selecting highest-priority tiers first. Tiers that do not fit are silently
// dropped. The returned slice preserves the original tier order for prompt
// assembly; omitted tiers produce an empty string at their position.
//
// budgetTokens should be the estimated prompt token ceiling, e.g.
// (modelContextWindow - maxResponseTokens - safetyMargin).
func TrimToBudget(tiers []Tier, budgetTokens int) []string {
	if budgetTokens <= 0 {
		// No budget information — return everything.
		out := make([]string, len(tiers))
		for i, t := range tiers {
			out[i] = t.Content
		}
		return out
	}

	// Sort a working copy by priority so we fill high-priority tiers first.
	type indexed struct {
		idx  int
		tier Tier
	}
	work := make([]indexed, len(tiers))
	for i, t := range tiers {
		work[i] = indexed{i, t}
	}
	sort.SliceStable(work, func(a, b int) bool {
		return work[a].tier.Priority < work[b].tier.Priority
	})

	selected := make([]bool, len(tiers))
	remaining := budgetTokens

	for _, w := range work {
		cost := EstimateTokens(w.tier.Content)
		if cost == 0 {
			selected[w.idx] = true // empty tiers are free
			continue
		}
		if cost <= remaining {
			selected[w.idx] = true
			remaining -= cost
		}
		// If a tier doesn't fit, skip it — do not truncate mid-content.
	}

	out := make([]string, len(tiers))
	for i, t := range tiers {
		if selected[i] {
			out[i] = t.Content
		}
	}
	return out
}

// JoinTiers concatenates non-empty tier contents with the given separator.
// Useful for assembling prompt sections after TrimToBudget.
func JoinTiers(contents []string, sep string) string {
	var parts []string
	for _, c := range contents {
		if strings.TrimSpace(c) != "" {
			parts = append(parts, c)
		}
	}
	return strings.Join(parts, sep)
}
