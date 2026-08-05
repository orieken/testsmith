package generation

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// ── JSON ──────────────────────────────────────────────────────────────────────

type gapsJSONReport struct {
	TotalSources int             `json:"total_sources"`
	Covered      int             `json:"covered"`
	CoveragePC   int             `json:"coverage_pct"`
	GapCount     int             `json:"gap_count"`
	Gaps         []gapsJSONEntry `json:"gaps"`
}

type gapsJSONEntry struct {
	SourcePath       string  `json:"source_path"`
	Status           string  `json:"status"`
	PriorityScore    float64 `json:"priority_score"`
	ExternalDeps     int     `json:"external_deps"`
	Dependents       int     `json:"dependents"`
	SuggestedCommand string  `json:"suggested_command"`
}

// GenerateReportJSON renders gaps as a JSON document.
func GenerateReportJSON(gaps []domain.CoverageGap, totalSources int) string {
	covered := totalSources - len(gaps)
	pct := 0
	if totalSources > 0 {
		pct = covered * 100 / totalSources
	}

	entries := make([]gapsJSONEntry, len(gaps))
	for i, g := range gaps {
		entries[i] = gapsJSONEntry{
			SourcePath:       g.SourcePath,
			Status:           string(g.Status),
			PriorityScore:    g.PriorityScore,
			ExternalDeps:     g.ExternalDeps,
			Dependents:       g.Dependents,
			SuggestedCommand: g.SuggestedCommand,
		}
	}

	report := gapsJSONReport{
		TotalSources: totalSources,
		Covered:      covered,
		CoveragePC:   pct,
		GapCount:     len(gaps),
		Gaps:         entries,
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}

// ── JUnit XML ─────────────────────────────────────────────────────────────────

type junitTestSuites struct {
	XMLName    xml.Name     `xml:"testsuites"`
	Name       string       `xml:"name,attr"`
	TestSuites []junitSuite `xml:"testsuite"`
}

type junitSuite struct {
	Name     string          `xml:"name,attr"`
	Tests    int             `xml:"tests,attr"`
	Failures int             `xml:"failures,attr"`
	Cases    []junitTestCase `xml:"testcase"`
}

type junitTestCase struct {
	Name      string        `xml:"name,attr"`
	ClassName string        `xml:"classname,attr"`
	Failure   *junitFailure `xml:"failure,omitempty"`
}

type junitFailure struct {
	Message string `xml:"message,attr"`
	Type    string `xml:"type,attr"`
	Body    string `xml:",chardata"`
}

// GenerateReportJUnit renders gaps as a JUnit XML document suitable for CI ingestion.
// Each source file with a gap becomes a failing test case.
func GenerateReportJUnit(gaps []domain.CoverageGap, totalSources int) string {
	cases := make([]junitTestCase, 0, totalSources)
	for _, g := range gaps {
		tc := junitTestCase{
			Name:      g.SourcePath,
			ClassName: "coverage",
			Failure: &junitFailure{
				Message: fmt.Sprintf("coverage gap: %s", g.Status),
				Type:    "coverage-gap",
				Body: strings.TrimSpace(fmt.Sprintf(
					"status: %s\npriority: %.2f\nsuggest: %s",
					g.Status, g.PriorityScore, g.SuggestedCommand,
				)),
			},
		}
		cases = append(cases, tc)
	}

	suite := junitSuite{
		Name:     "testsmith-gaps",
		Tests:    totalSources,
		Failures: len(gaps),
		Cases:    cases,
	}

	doc := junitTestSuites{
		Name:       "testsmith",
		TestSuites: []junitSuite{suite},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Sprintf("<!-- error: %s -->", err)
	}
	return xml.Header + string(out)
}

// ── validate JSON/JUnit ───────────────────────────────────────────────────────

// ValidateFileResult holds issues found for a single test file.
type ValidateFileResult struct {
	FilePath string
	Issues   []domain.ValidationIssue
}

type validateJSONReport struct {
	Checked  int                     `json:"checked"`
	Errors   int                     `json:"errors"`
	Warnings int                     `json:"warnings"`
	Files    []validateJSONFileEntry `json:"files"`
}

type validateJSONFileEntry struct {
	Path   string              `json:"path"`
	Issues []validateJSONIssue `json:"issues"`
}

type validateJSONIssue struct {
	Rule     string `json:"rule"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// GenerateValidateJSON renders validation results as a JSON document.
func GenerateValidateJSON(results []ValidateFileResult) string {
	var totalErrors, totalWarnings int
	entries := make([]validateJSONFileEntry, 0, len(results))

	for _, r := range results {
		issues := make([]validateJSONIssue, len(r.Issues))
		for i, iss := range r.Issues {
			issues[i] = validateJSONIssue{
				Rule:     iss.Rule,
				Severity: string(iss.Severity),
				Message:  iss.Message,
			}
			switch iss.Severity {
			case domain.SeverityError:
				totalErrors++
			case domain.SeverityWarning:
				totalWarnings++
			case domain.SeverityInfo:
				// info issues are not counted toward error/warning totals
			}
		}
		entries = append(entries, validateJSONFileEntry{
			Path:   r.FilePath,
			Issues: issues,
		})
	}

	report := validateJSONReport{
		Checked:  len(results),
		Errors:   totalErrors,
		Warnings: totalWarnings,
		Files:    entries,
	}

	b, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error":%q}`, err.Error())
	}
	return string(b)
}

// GenerateValidateJUnit renders validation results as JUnit XML.
// Each test file is a test case; files with error-severity issues become failures.
func GenerateValidateJUnit(results []ValidateFileResult) string {
	var failures int
	cases := make([]junitTestCase, 0, len(results))

	for _, r := range results {
		tc := junitTestCase{
			Name:      r.FilePath,
			ClassName: "validate",
		}

		var errMsgs []string
		for _, iss := range r.Issues {
			if iss.Severity == domain.SeverityError {
				errMsgs = append(errMsgs, fmt.Sprintf("[%s] %s", iss.Rule, iss.Message))
			}
		}

		if len(errMsgs) > 0 {
			failures++
			tc.Failure = &junitFailure{
				Message: fmt.Sprintf("%d error(s)", len(errMsgs)),
				Type:    "validation-error",
				Body:    strings.Join(errMsgs, "\n"),
			}
		}
		cases = append(cases, tc)
	}

	suite := junitSuite{
		Name:     "testsmith-validate",
		Tests:    len(results),
		Failures: failures,
		Cases:    cases,
	}

	doc := junitTestSuites{
		Name:       "testsmith",
		TestSuites: []junitSuite{suite},
	}

	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Sprintf("<!-- error: %s -->", err)
	}
	return xml.Header + string(out)
}
