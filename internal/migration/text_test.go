package migration_test

import (
	"strings"
	"testing"

	"github.com/orieken/assay/internal/migration"
)

func TestTextMigrator_AppliesStepsInOrder(t *testing.T) {
	m := migration.New("a", "b").
		Add(`foo`, `bar`).
		Add(`bar`, `baz`)

	got, err := m.MigrateFile("foo")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "baz" {
		t.Errorf("got %q, want %q", got, "baz")
	}
}

func TestTextMigrator_InjectImport_BeforeFirstImport(t *testing.T) {
	content := "import os\n\ndef test_foo(): pass\n"
	m := migration.New("a", "b").InjectImport("from unittest.mock import patch")

	got, _ := m.MigrateFile(content)
	lines := strings.Split(got, "\n")

	// The injected import should appear before "import os".
	injectedIdx, osIdx := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "from unittest.mock") {
			injectedIdx = i
		}
		if strings.TrimSpace(l) == "import os" {
			osIdx = i
		}
	}
	if injectedIdx < 0 {
		t.Fatal("injected import not found in output")
	}
	if osIdx < 0 {
		t.Fatal("original import not found in output")
	}
	if injectedIdx >= osIdx {
		t.Errorf("injected import (line %d) should precede existing import (line %d)", injectedIdx, osIdx)
	}
}

func TestTextMigrator_InjectImport_FallsBackToTop(t *testing.T) {
	content := "def test_foo(): pass\n"
	m := migration.New("a", "b").InjectImport("import xyz")

	got, _ := m.MigrateFile(content)
	if !strings.HasPrefix(got, "import xyz\n") {
		t.Errorf("expected import at top, got:\n%s", got)
	}
}

func TestTextMigrator_FromTo(t *testing.T) {
	m := migration.New("jest", "vitest")
	if m.From() != "jest" {
		t.Errorf("From() = %q, want %q", m.From(), "jest")
	}
	if m.To() != "vitest" {
		t.Errorf("To() = %q, want %q", m.To(), "vitest")
	}
}
