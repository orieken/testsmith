// Package discount applies promotional discounts to prices.
package discount

// Rule describes a discount that reduces a price by a fixed percentage.
type Rule struct {
	Code       string
	Percentage int // 0–100
}

// Apply returns the discounted price in cents.
// A 0% discount returns the original price; 100% makes it free.
func (r Rule) Apply(priceCents int64) int64 {
	if r.Percentage <= 0 {
		return priceCents
	}
	if r.Percentage >= 100 {
		return 0
	}
	return priceCents - (priceCents*int64(r.Percentage))/100
}

// FindRule returns the matching Rule from a catalogue, and false when absent.
func FindRule(catalogue []Rule, code string) (Rule, bool) {
	for _, r := range catalogue {
		if r.Code == code {
			return r, true
		}
	}
	return Rule{}, false
}
