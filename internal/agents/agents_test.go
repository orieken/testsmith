package agents_test

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/agents"
)

var expectedAgents = []string{
	"testsmith-test-author.md",
	"testsmith-pattern-curator.md",
	"testsmith-migration-guide.md",
}

func TestAll_returnsAllBundledAgents(t *testing.T) {
	files, err := agents.All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	for _, name := range expectedAgents {
		if _, ok := files[name]; !ok {
			t.Errorf("All() missing expected agent file %q", name)
		}
	}
}

func TestAll_noUnexpectedFiles(t *testing.T) {
	files, err := agents.All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	if got, want := len(files), len(expectedAgents); got != want {
		t.Errorf("All() returned %d files, want %d", got, want)
	}
}

func TestAll_eachFileHasContent(t *testing.T) {
	files, err := agents.All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	for name, content := range files {
		if len(content) == 0 {
			t.Errorf("agent file %q is empty", name)
		}
	}
}

func TestAll_eachFileHasFrontmatter(t *testing.T) {
	files, err := agents.All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	for name, content := range files {
		if !strings.HasPrefix(string(content), "---") {
			t.Errorf("agent file %q does not start with YAML frontmatter (---)", name)
		}
	}
}

func TestAll_eachFileHasNameField(t *testing.T) {
	files, err := agents.All()
	if err != nil {
		t.Fatalf("All() error: %v", err)
	}
	for name, content := range files {
		if !strings.Contains(string(content), "name:") {
			t.Errorf("agent file %q missing 'name:' frontmatter field", name)
		}
	}
}
