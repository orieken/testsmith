package domain

// Severity classifies the impact of a ValidationIssue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ValidationIssue is a single finding from validating a test file against its
// configured adapter's conventions.
type ValidationIssue struct {
	Rule     string   // stable identifier, e.g. "junit4-import-in-junit5"
	Severity Severity
	Message  string
}
