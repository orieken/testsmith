// Package validation provides TextValidator, a rule-based checker that
// identifies framework mismatches and convention violations in test files.
package validation

import (
	"regexp"

	"github.com/orieken/testsmith/internal/domain"
)

type rule struct {
	id       string
	severity domain.Severity
	message  string
	// required: issue when pattern is NOT found in the content.
	// When nil the positive check is skipped.
	required *regexp.Regexp
	// forbidden: issue when pattern IS found in the content.
	// When nil the negative check is skipped.
	forbidden *regexp.Regexp
}

// TextValidator applies an ordered list of regex-based rules to test file content.
type TextValidator struct {
	framework, mockLib string
	rules              []rule
}

// New returns a TextValidator for the given (framework, mockLib) pair.
func New(framework, mockLib string) *TextValidator {
	return &TextValidator{framework: framework, mockLib: mockLib}
}

// Require adds a rule that fires when pattern is NOT present.
func (v *TextValidator) Require(id, pattern, message string, sev domain.Severity) *TextValidator {
	v.rules = append(v.rules, rule{
		id:       id,
		severity: sev,
		message:  message,
		required: regexp.MustCompile(pattern),
	})
	return v
}

// Forbid adds a rule that fires when pattern IS present.
func (v *TextValidator) Forbid(id, pattern, message string, sev domain.Severity) *TextValidator {
	v.rules = append(v.rules, rule{
		id:       id,
		severity: sev,
		message:  message,
		forbidden: regexp.MustCompile(pattern),
	})
	return v
}

func (v *TextValidator) Framework() string  { return v.framework }
func (v *TextValidator) MockLibrary() string { return v.mockLib }

// Validate runs all rules against content and returns any issues found.
func (v *TextValidator) Validate(content string) []domain.ValidationIssue {
	var issues []domain.ValidationIssue
	for _, r := range v.rules {
		if r.required != nil && !r.required.MatchString(content) {
			issues = append(issues, domain.ValidationIssue{
				Rule: r.id, Severity: r.severity, Message: r.message,
			})
		}
		if r.forbidden != nil && r.forbidden.MatchString(content) {
			issues = append(issues, domain.ValidationIssue{
				Rule: r.id, Severity: r.severity, Message: r.message,
			})
		}
	}
	return issues
}
