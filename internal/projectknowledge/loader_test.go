package projectknowledge_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/projectknowledge"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}

func TestLoad(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(root string)
		want  string
	}{
		{
			name:  "returns content when root ASSAY.md exists",
			setup: func(root string) { writeTestFile(t, filepath.Join(root, "ASSAY.md"), "# conventions") },
			want:  "# conventions",
		},
		{
			name:  "returns empty string when file is absent",
			setup: func(_ string) {},
			want:  "",
		},
		{
			name:  "trims surrounding whitespace",
			setup: func(root string) { writeTestFile(t, filepath.Join(root, "ASSAY.md"), "  content  \n") },
			want:  "content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			tt.setup(root)
			if got := projectknowledge.Load(root); got != tt.want {
				t.Errorf("Load() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadForFile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(root, sub string)
		want  string
	}{
		{
			name: "returns root content when no subdir file",
			setup: func(root, _ string) {
				writeTestFile(t, filepath.Join(root, "ASSAY.md"), "root")
			},
			want: "root",
		},
		{
			name: "merges root and subdir content",
			setup: func(root, sub string) {
				writeTestFile(t, filepath.Join(root, "ASSAY.md"), "root")
				writeTestFile(t, filepath.Join(sub, "ASSAY.md"), "pkg")
			},
			want: "root\n\n---\n\n## Package-level conventions\n\npkg",
		},
		{
			name:  "returns empty when neither file exists",
			setup: func(_, _ string) {},
			want:  "",
		},
		{
			name: "returns subdir content when root is absent",
			setup: func(_, sub string) {
				writeTestFile(t, filepath.Join(sub, "ASSAY.md"), "pkg-only")
			},
			want: "pkg-only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			sub := filepath.Join(root, "internal", "services")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			tt.setup(root, sub)
			sourcePath := filepath.Join(sub, "service.go")
			if got := projectknowledge.LoadForFile(sourcePath, root); got != tt.want {
				t.Errorf("LoadForFile() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadForDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		setup     func(root, sub string)
		useSubDir bool
		want      string
	}{
		{
			name: "returns root content when dir equals root",
			setup: func(root, _ string) {
				writeTestFile(t, filepath.Join(root, "ASSAY.md"), "root-only")
			},
			useSubDir: false,
			want:      "root-only",
		},
		{
			name: "returns root content when subdir has no file",
			setup: func(root, _ string) {
				writeTestFile(t, filepath.Join(root, "ASSAY.md"), "root")
			},
			useSubDir: true,
			want:      "root",
		},
		{
			name: "merges when both files exist",
			setup: func(root, sub string) {
				writeTestFile(t, filepath.Join(root, "ASSAY.md"), "root")
				writeTestFile(t, filepath.Join(sub, "ASSAY.md"), "pkg")
			},
			useSubDir: true,
			want:      "root\n\n---\n\n## Package-level conventions\n\npkg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			sub := filepath.Join(root, "pkg")
			if err := os.MkdirAll(sub, 0o755); err != nil {
				t.Fatal(err)
			}
			tt.setup(root, sub)
			dir := root
			if tt.useSubDir {
				dir = sub
			}
			if got := projectknowledge.LoadForDir(dir, root); got != tt.want {
				t.Errorf("LoadForDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLoadPatterns(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		setup    func(root string)
		contains []string
		empty    bool
	}{
		{
			name:  "returns empty when patterns directory is absent",
			setup: func(_ string) {},
			empty: true,
		},
		{
			name: "returns empty when directory has no markdown files",
			setup: func(root string) {
				dir := filepath.Join(root, ".assay", "patterns")
				if err := os.MkdirAll(dir, 0o755); err != nil {
					t.Fatal(err)
				}
				writeTestFile(t, filepath.Join(dir, "notes.txt"), "not markdown")
			},
			empty: true,
		},
		{
			name: "skips README.md",
			setup: func(root string) {
				dir := filepath.Join(root, ".assay", "patterns")
				writeTestFile(t, filepath.Join(dir, "README.md"), "# README content")
			},
			empty: true,
		},
		{
			name: "returns single pattern with header derived from filename",
			setup: func(root string) {
				dir := filepath.Join(root, ".assay", "patterns")
				writeTestFile(t, filepath.Join(dir, "mock-database-sqlmock.md"), "Use sqlmock for DB tests.")
			},
			contains: []string{"### Pattern: mock-database-sqlmock", "Use sqlmock for DB tests."},
		},
		{
			name: "merges multiple pattern files with separating newlines",
			setup: func(root string) {
				dir := filepath.Join(root, ".assay", "patterns")
				writeTestFile(t, filepath.Join(dir, "pattern-a.md"), "Content A")
				writeTestFile(t, filepath.Join(dir, "pattern-b.md"), "Content B")
			},
			contains: []string{"### Pattern: pattern-a", "Content A", "### Pattern: pattern-b", "Content B"},
		},
		{
			name: "skips empty markdown files",
			setup: func(root string) {
				dir := filepath.Join(root, ".assay", "patterns")
				writeTestFile(t, filepath.Join(dir, "empty.md"), "   ")
				writeTestFile(t, filepath.Join(dir, "real.md"), "Has content")
			},
			contains: []string{"### Pattern: real", "Has content"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			tt.setup(root)
			got := projectknowledge.LoadPatterns(root)
			if tt.empty {
				if got != "" {
					t.Errorf("LoadPatterns() = %q, want empty string", got)
				}
				return
			}
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("LoadPatterns() does not contain %q\ngot:\n%s", want, got)
				}
			}
		})
	}
}

func TestTemplate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		language string
		contains string
	}{
		{name: "go template contains naming convention", language: "go", contains: "Test<Function>"},
		{name: "python template contains pytest", language: "python", contains: "pytest"},
		{name: "typescript template mentions framework choice", language: "typescript", contains: "vitest"},
		{name: "java template contains JUnit", language: "java", contains: "JUnit 5"},
		{name: "csharp template contains xUnit", language: "csharp", contains: "xUnit"},
		{name: "unknown language falls back to default", language: "cobol", contains: "Test conventions"},
		{name: "empty language falls back to default", language: "", contains: "Test conventions"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := projectknowledge.Template(tt.language)
			if result == "" {
				t.Fatal("Template() returned empty string")
			}
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Template(%q) does not contain %q\ngot:\n%s", tt.language, tt.contains, result)
			}
		})
	}
}
