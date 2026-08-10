package validation_test

import (
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

func TestRequire_PassesWhenPatternPresent(t *testing.T) {
	v := validation.New("jest", "jest").
		Require("needs-import", `import.*jest`, "jest not imported", domain.SeverityError)

	issues := v.Validate(`import { describe } from 'jest'`)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestRequire_FiresWhenPatternAbsent(t *testing.T) {
	v := validation.New("jest", "jest").
		Require("needs-import", `import.*jest`, "jest not imported", domain.SeverityWarning)

	issues := v.Validate(`describe('foo', () => {})`)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != domain.SeverityWarning {
		t.Errorf("severity: got %q, want warning", issues[0].Severity)
	}
	if issues[0].Rule != "needs-import" {
		t.Errorf("rule: got %q, want needs-import", issues[0].Rule)
	}
}

func TestForbid_PassesWhenPatternAbsent(t *testing.T) {
	v := validation.New("vitest", "vitest").
		Forbid("no-jest-api", `\bjest\.`, "jest API in vitest file", domain.SeverityError)

	issues := v.Validate(`vi.fn()`)
	if len(issues) != 0 {
		t.Errorf("expected no issues, got %v", issues)
	}
}

func TestForbid_FiresWhenPatternPresent(t *testing.T) {
	v := validation.New("vitest", "vitest").
		Forbid("no-jest-api", `\bjest\.`, "jest API in vitest file", domain.SeverityError)

	issues := v.Validate(`jest.fn()`)
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Severity != domain.SeverityError {
		t.Errorf("severity: got %q, want error", issues[0].Severity)
	}
}

func TestMultipleRules_AllChecked(t *testing.T) {
	v := validation.New("vitest", "vitest").
		Require("has-vi", `\bvi\.`, "missing vi usage", domain.SeverityInfo).
		Forbid("no-jest", `\bjest\.`, "jest found", domain.SeverityError)

	// Content has jest but no vi → two issues.
	issues := v.Validate(`jest.fn()`)
	if len(issues) != 2 {
		t.Errorf("expected 2 issues (missing vi + forbidden jest), got %d: %v", len(issues), issues)
	}
}
