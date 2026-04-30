// Package migration provides the TextMigrator, a composable regex-based
// transformer for rewriting test file content between framework conventions.
package migration

import (
	"regexp"
	"strings"
)

type step struct {
	re  *regexp.Regexp
	sub string
}

// TextMigrator applies an ordered list of regex replacements to migrate
// test-file content between two frameworks.
type TextMigrator struct {
	from, to string
	steps    []step
	imports  []string // lines injected near the first import block
}

// New returns a TextMigrator for the given (from, to) pair with no steps yet.
func New(from, to string) *TextMigrator {
	return &TextMigrator{from: from, to: to}
}

// Add appends a regex-replace step. pattern is a Go regexp; sub is the
// replacement string (supports $1, ${name} back-references).
// Panics on an invalid pattern — use only with compile-time literals.
func (m *TextMigrator) Add(pattern, sub string) *TextMigrator {
	m.steps = append(m.steps, step{regexp.MustCompile(pattern), sub})
	return m
}

// InjectImport registers an import line to be inserted before the first
// existing import block after all regex steps have been applied.
func (m *TextMigrator) InjectImport(importLine string) *TextMigrator {
	m.imports = append(m.imports, importLine)
	return m
}

func (m *TextMigrator) From() string { return m.from }
func (m *TextMigrator) To() string   { return m.to }

// MigrateFile applies all steps in registration order then injects any
// pending import lines.
func (m *TextMigrator) MigrateFile(content string) (string, error) {
	for _, s := range m.steps {
		content = s.re.ReplaceAllString(content, s.sub)
	}
	if len(m.imports) > 0 {
		content = injectImports(content, m.imports)
	}
	return content, nil
}

// injectImports inserts the given lines before the first import/from/using
// statement in content. Falls back to prepending at the very top.
func injectImports(content string, lines []string) string {
	block := strings.Join(lines, "\n") + "\n"
	parts := strings.Split(content, "\n")
	for i, line := range parts {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "import ") ||
			strings.HasPrefix(t, "from ") ||
			strings.HasPrefix(t, "using ") {
			out := make([]string, 0, len(parts)+len(lines)+1)
			out = append(out, parts[:i]...)
			out = append(out, block)
			out = append(out, parts[i:]...)
			return strings.Join(out, "\n")
		}
	}
	return block + content
}
