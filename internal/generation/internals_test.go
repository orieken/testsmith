package generation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
)

// ── trimOutput ────────────────────────────────────────────────────────────────

func TestTrimOutput_ShortString_ReturnedAsIs(t *testing.T) {
	t.Parallel()
	got := trimOutput([]byte("hello"))
	if got != "hello" {
		t.Errorf("trimOutput = %q, want %q", got, "hello")
	}
}

func TestTrimOutput_LongString_Truncated(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 500)
	got := trimOutput([]byte(long))
	if len(got) > 404 { // 400 chars + "…"
		t.Errorf("trimOutput did not truncate: len=%d", len(got))
	}
	if !strings.HasSuffix(got, "…") {
		t.Errorf("trimOutput should end with ellipsis, got: %q", got[len(got)-5:])
	}
}

func TestTrimOutput_WhitespaceOnlyInput_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	got := trimOutput([]byte("   \n\t  "))
	if got != "" {
		t.Errorf("trimOutput of whitespace-only = %q, want empty", got)
	}
}

func TestTrimOutput_LeadingTrailingWhitespace_Trimmed(t *testing.T) {
	t.Parallel()
	got := trimOutput([]byte("  hello  "))
	if got != "hello" {
		t.Errorf("trimOutput = %q, want trimmed", got)
	}
}

// ── fileExists ────────────────────────────────────────────────────────────────

func TestFileExists_ExistingFile_ReturnsTrue(t *testing.T) {
	t.Parallel()
	f, err := os.CreateTemp(t.TempDir(), "exists-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	if !fileExists(f.Name()) {
		t.Errorf("fileExists(%q) = false, want true", f.Name())
	}
}

func TestFileExists_NonExistentFile_ReturnsFalse(t *testing.T) {
	t.Parallel()
	if fileExists(filepath.Join(t.TempDir(), "does-not-exist.txt")) {
		t.Error("fileExists returned true for non-existent file")
	}
}

// ── deriveTestPathFromConfig ──────────────────────────────────────────────────

func TestDeriveTestPathFromConfig_EmptySuffix_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	cfg := domain.TestFrameworkConfig{TestFileSuffix: ""}
	if got := deriveTestPathFromConfig("/src/util.go", cfg); got != "" {
		t.Errorf("empty suffix should return empty string, got %q", got)
	}
}

func TestDeriveTestPathFromConfig_WithExtension_ReplacesDotPart(t *testing.T) {
	t.Parallel()
	cfg := domain.TestFrameworkConfig{TestFileSuffix: "_test.go"}
	got := deriveTestPathFromConfig("/src/util.go", cfg)
	if got != "/src/util_test.go" {
		t.Errorf("got %q, want /src/util_test.go", got)
	}
}

func TestDeriveTestPathFromConfig_WithoutExtension_AppendsSuffix(t *testing.T) {
	t.Parallel()
	cfg := domain.TestFrameworkConfig{TestFileSuffix: "_test.go"}
	got := deriveTestPathFromConfig("/src/util", cfg)
	if got != "/src/util_test.go" {
		t.Errorf("got %q, want /src/util_test.go", got)
	}
}

// ── buildDepsSignatures ───────────────────────────────────────────────────────

func TestBuildDepsSignatures_EmptyIndex_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	a := &domain.SourceAnalysis{
		Imports: domain.ClassifiedImports{
			Internal: []domain.ImportInfo{{Module: "mymod/repo"}},
		},
	}
	if got := buildDepsSignatures(a, nil); got != "" {
		t.Errorf("empty index: got %q, want empty", got)
	}
}

func TestBuildDepsSignatures_NoInternalImports_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	idx := map[string]*domain.SourceAnalysis{"mymod/repo": {}}
	a := &domain.SourceAnalysis{}
	if got := buildDepsSignatures(a, idx); got != "" {
		t.Errorf("no internal imports: got %q, want empty", got)
	}
}

func TestBuildDepsSignatures_MatchingEntry_ReturnsSignatures(t *testing.T) {
	t.Parallel()
	dep := &domain.SourceAnalysis{
		PublicAPI: []domain.PublicMember{
			{Name: "Save", Kind: domain.KindFunction, Parameters: []domain.ParamInfo{{Name: "ctx", TypeHint: "Context"}}},
			{Name: "Delete", Kind: domain.KindFunction},
		},
	}
	idx := map[string]*domain.SourceAnalysis{"mymod/repo": dep}
	a := &domain.SourceAnalysis{
		Imports: domain.ClassifiedImports{
			Internal: []domain.ImportInfo{{Module: "mymod/repo"}},
		},
	}
	got := buildDepsSignatures(a, idx)
	if !strings.Contains(got, "mymod/repo") {
		t.Errorf("signatures missing module path:\n%s", got)
	}
	if !strings.Contains(got, "Save") {
		t.Errorf("signatures missing Save function:\n%s", got)
	}
	if !strings.Contains(got, "ctx") {
		t.Errorf("signatures missing parameter name:\n%s", got)
	}
}

func TestBuildDepsSignatures_MissingDepInIndex_Skipped(t *testing.T) {
	t.Parallel()
	idx := map[string]*domain.SourceAnalysis{"other/mod": {}}
	a := &domain.SourceAnalysis{
		Imports: domain.ClassifiedImports{
			Internal: []domain.ImportInfo{{Module: "mymod/repo"}},
		},
	}
	// "mymod/repo" is not in idx → skipped, result is empty.
	if got := buildDepsSignatures(a, idx); got != "" {
		t.Errorf("missing dep should yield empty string, got %q", got)
	}
}
