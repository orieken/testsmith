package generation_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/generation"
)

// ── GenerateReportJSON ────────────────────────────────────────────────────────

func TestGenerateReportJSON_NoGaps(t *testing.T) {
	out := generation.GenerateReportJSON(nil, 5)

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if got := doc["gap_count"]; got != float64(0) {
		t.Errorf("gap_count: got %v, want 0", got)
	}
	if got := doc["coverage_pct"]; got != float64(100) {
		t.Errorf("coverage_pct: got %v, want 100", got)
	}
}

func TestGenerateReportJSON_WithGaps(t *testing.T) {
	gaps := []domain.CoverageGap{
		{SourcePath: "src/foo.go", Status: domain.CoverageNoTest, PriorityScore: 9.5, ExternalDeps: 2},
	}
	out := generation.GenerateReportJSON(gaps, 3)

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if got := doc["gap_count"]; got != float64(1) {
		t.Errorf("gap_count: got %v, want 1", got)
	}
	gapsSlice, ok := doc["gaps"].([]any)
	if !ok || len(gapsSlice) != 1 {
		t.Fatalf("gaps array: got %v", doc["gaps"])
	}
	entry, _ := gapsSlice[0].(map[string]any)
	if entry["source_path"] != "src/foo.go" {
		t.Errorf("source_path: got %v", entry["source_path"])
	}
}

// ── GenerateReportJUnit ───────────────────────────────────────────────────────

func TestGenerateReportJUnit_NoGaps(t *testing.T) {
	out := generation.GenerateReportJUnit(nil, 3)

	if !strings.HasPrefix(out, "<?xml") {
		t.Fatalf("expected XML preamble, got: %s", out[:min(60, len(out))])
	}
	if strings.Contains(out, "<failure") {
		t.Errorf("no failures expected when gaps is nil")
	}
	if !strings.Contains(out, `failures="0"`) {
		t.Errorf("expected failures=0 in output")
	}
}

func TestGenerateReportJUnit_WithGaps(t *testing.T) {
	gaps := []domain.CoverageGap{
		{SourcePath: "src/bar.go", Status: domain.CoverageNoTest, SuggestedCommand: "assay generate src/bar.go"},
	}
	out := generation.GenerateReportJUnit(gaps, 5)

	if !strings.Contains(out, "<failure") {
		t.Errorf("expected failure element for gap")
	}
	if !strings.Contains(out, "src/bar.go") {
		t.Errorf("expected source path in output")
	}
	if !strings.Contains(out, `failures="1"`) {
		t.Errorf("expected failures=1 in output")
	}
}

// ── GenerateValidateJSON ──────────────────────────────────────────────────────

func TestGenerateValidateJSON_Clean(t *testing.T) {
	results := []generation.ValidateFileResult{
		{FilePath: "foo_test.go", Issues: nil},
	}
	out := generation.GenerateValidateJSON(results)

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got := doc["errors"]; got != float64(0) {
		t.Errorf("errors: got %v, want 0", got)
	}
}

func TestGenerateValidateJSON_WithErrors(t *testing.T) {
	results := []generation.ValidateFileResult{
		{
			FilePath: "foo_test.go",
			Issues: []domain.ValidationIssue{
				{Rule: "rule-a", Severity: domain.SeverityError, Message: "bad import"},
				{Rule: "rule-b", Severity: domain.SeverityWarning, Message: "style issue"},
				{Rule: "rule-c", Severity: domain.SeverityInfo, Message: "tip"},
			},
		},
	}
	out := generation.GenerateValidateJSON(results)

	var doc map[string]any
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if got := doc["errors"]; got != float64(1) {
		t.Errorf("errors: got %v, want 1", got)
	}
	if got := doc["warnings"]; got != float64(1) {
		t.Errorf("warnings: got %v, want 1", got)
	}
}

// ── GenerateValidateJUnit ─────────────────────────────────────────────────────

func TestGenerateValidateJUnit_Clean(t *testing.T) {
	results := []generation.ValidateFileResult{
		{FilePath: "clean_test.go", Issues: nil},
	}
	out := generation.GenerateValidateJUnit(results)

	if !strings.HasPrefix(out, "<?xml") {
		t.Fatalf("expected XML preamble")
	}
	if strings.Contains(out, "<failure") {
		t.Errorf("no failures expected for clean file")
	}
}

func TestGenerateValidateJUnit_WithErrors(t *testing.T) {
	results := []generation.ValidateFileResult{
		{
			FilePath: "bad_test.go",
			Issues: []domain.ValidationIssue{
				{Rule: "junit4-import", Severity: domain.SeverityError, Message: "wrong import"},
			},
		},
	}
	out := generation.GenerateValidateJUnit(results)

	if !strings.Contains(out, "<failure") {
		t.Errorf("expected failure element")
	}
	if !strings.Contains(out, "bad_test.go") {
		t.Errorf("expected file path in output")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
