package python

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
)

// ── mirrorTestPathToSrcDir ─────────────────────────────────────────────────────

// backfill / AC: mirrorTestPathToSrcDir strips tests/ prefix and returns the src directory
func TestMirrorTestPathToSrcDir(t *testing.T) {
	t.Parallel()
	cases := []struct {
		testRel string
		want    string
	}{
		// Standard src-layout: tests/src/services/test_foo.py → src/services
		{"tests/src/services/test_payment.py", "src/services"},
		// Deep nesting
		{"tests/src/core/auth/test_login.py", "src/core/auth"},
		// Direct under tests/<dir>/
		{"tests/myapp/test_models.py", "myapp"},
		// File directly under tests/ — filepath.Dir is "." → returns ""
		{"tests/test_utils.py", ""},
		// No tests/ prefix — not a test path at all
		{"src/services/payment.py", ""},
	}
	for _, tc := range cases {
		got := mirrorTestPathToSrcDir(tc.testRel, "/proj")
		if got != tc.want {
			t.Errorf("mirrorTestPathToSrcDir(%q) = %q, want %q", tc.testRel, got, tc.want)
		}
	}
}

// ── extractExistingPaths ───────────────────────────────────────────────────────

// backfill / AC: extractExistingPaths parses quoted string entries from paths_to_add list
func TestExtractExistingPaths_ParsesEntries(t *testing.T) {
	t.Parallel()
	content := `import sys
from pathlib import Path

paths_to_add = [
    "src/services",
    "tests",
    "tests/fixtures",
]


def pytest_configure(config):
    pass
`
	paths := extractExistingPaths(content)
	want := map[string]bool{"src/services": true, "tests": true, "tests/fixtures": true}
	for _, p := range paths {
		if !want[p] {
			t.Errorf("unexpected path %q in result", p)
		}
		delete(want, p)
	}
	for p := range want {
		t.Errorf("expected path %q was not extracted", p)
	}
}

// backfill / AC: extractExistingPaths returns nil when paths_to_add is absent
func TestExtractExistingPaths_MissingList(t *testing.T) {
	t.Parallel()
	content := "import sys\ndef pytest_configure(config): pass\n"
	paths := extractExistingPaths(content)
	if len(paths) != 0 {
		t.Errorf("extractExistingPaths with no list: got %v, want empty", paths)
	}
}

// backfill / AC: extractExistingPaths returns nil when closing bracket is absent
func TestExtractExistingPaths_MissingClosingBracket(t *testing.T) {
	t.Parallel()
	// paths_to_add present but no \n] terminator
	content := "paths_to_add = [\n    \"src\",\n"
	paths := extractExistingPaths(content)
	if len(paths) != 0 {
		t.Errorf("extractExistingPaths with no closing bracket: got %v, want empty", paths)
	}
}

// ── injectPaths ────────────────────────────────────────────────────────────────

// backfill / AC: injectPaths reports no change when all required paths already exist
func TestInjectPaths_NoChange_WhenAllPresent(t *testing.T) {
	t.Parallel()
	content := "paths_to_add = [\n    \"src\",\n    \"tests\",\n]\n"
	updated, changed := injectPaths(content, []string{"src", "tests"})
	if changed {
		t.Error("injectPaths must not report a change when all paths already exist")
	}
	if updated != content {
		t.Error("injectPaths must return original content when nothing changed")
	}
}

// backfill / AC: injectPaths inserts missing paths before the closing bracket
func TestInjectPaths_AddsMissingPaths(t *testing.T) {
	t.Parallel()
	content := "paths_to_add = [\n]\n\ndef pytest_configure(config):\n    pass\n"
	updated, changed := injectPaths(content, []string{"src/services", "tests"})
	if !changed {
		t.Error("injectPaths must report a change when paths are missing")
	}
	if !strings.Contains(updated, `"src/services"`) {
		t.Error("updated content must contain \"src/services\"")
	}
	if !strings.Contains(updated, `"tests"`) {
		t.Error("updated content must contain \"tests\"")
	}
}

// backfill / AC: injectPaths only inserts the paths that are genuinely missing
func TestInjectPaths_OnlyMissingPaths(t *testing.T) {
	t.Parallel()
	content := "paths_to_add = [\n    \"tests\",\n]\n"
	updated, changed := injectPaths(content, []string{"tests", "src/new"})
	if !changed {
		t.Error("injectPaths must report a change when some paths are missing")
	}
	// "tests" must not be duplicated.
	if strings.Count(updated, `"tests"`) > 1 {
		t.Error("injectPaths must not duplicate an already-present path")
	}
	if !strings.Contains(updated, `"src/new"`) {
		t.Error("updated content must contain the new path \"src/new\"")
	}
}

// backfill / AC: injectPaths appends new declaration when paths_to_add is absent and no hook present
func TestInjectPaths_AppendNewDeclaration_NoPytestConfigure(t *testing.T) {
	t.Parallel()
	content := "import sys\n"
	updated, changed := injectPaths(content, []string{"src"})
	if !changed {
		t.Error("injectPaths must report a change when adding a new declaration")
	}
	if !strings.Contains(updated, "paths_to_add") {
		t.Error("updated content must contain 'paths_to_add'")
	}
	if !strings.Contains(updated, `"src"`) {
		t.Error("updated content must contain \"src\"")
	}
}

// backfill / AC: injectPaths inserts the declaration before def pytest_configure when paths_to_add is absent
func TestInjectPaths_InsertsBeforePytestConfigure(t *testing.T) {
	t.Parallel()
	content := "import sys\n\ndef pytest_configure(config):\n    pass\n"
	updated, changed := injectPaths(content, []string{"tests"})
	if !changed {
		t.Error("injectPaths must report a change")
	}
	confIdx := strings.Index(updated, "def pytest_configure")
	pathsIdx := strings.Index(updated, "paths_to_add")
	if pathsIdx == -1 {
		t.Fatal("updated content is missing 'paths_to_add'")
	}
	if pathsIdx > confIdx {
		t.Error("paths_to_add declaration must appear before def pytest_configure")
	}
}

// backfill / AC: injectPaths returns unchanged content when closing bracket is missing after finding paths_to_add
func TestInjectPaths_NoChange_MissingClosingBracket(t *testing.T) {
	t.Parallel()
	// paths_to_add is present but the \n] terminator is absent — cannot inject safely.
	content := "paths_to_add = [\n    \"tests\",\n"
	_, changed := injectPaths(content, []string{"src"})
	if changed {
		t.Error("injectPaths must not report a change when the closing bracket is missing")
	}
}

// ── computeRequiredPaths ───────────────────────────────────────────────────────

// backfill / AC: computeRequiredPaths returns empty slice when plan has no files
func TestComputeRequiredPaths_EmptyPlan(t *testing.T) {
	t.Parallel()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}
	paths := computeRequiredPaths(&domain.GenerationPlan{}, ctx)
	if len(paths) != 0 {
		t.Errorf("computeRequiredPaths with empty plan: got %v, want empty", paths)
	}
}

// backfill / AC: computeRequiredPaths skips fixture files and skipped test files
func TestComputeRequiredPaths_SkipsNonTestAndSkipped(t *testing.T) {
	t.Parallel()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			// fixture file — must be skipped
			{AbsPath: "/proj/tests/fixtures/stripe_fixture.py", Role: domain.RoleFixture, Action: domain.ActionCreate},
			// skipped test file — must be skipped
			{AbsPath: "/proj/tests/src/services/test_payment.py", Role: domain.RoleTestFile, Action: domain.ActionSkip},
		},
	}
	paths := computeRequiredPaths(plan, ctx)
	if len(paths) != 0 {
		t.Errorf("computeRequiredPaths must skip fixture/skipped files, got %v", paths)
	}
}

// backfill / AC: computeRequiredPaths derives src dir, tests, and tests/fixtures for a new test file
func TestComputeRequiredPaths_DerivesAllPaths(t *testing.T) {
	t.Parallel()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{
				AbsPath: "/proj/tests/src/services/test_payment.py",
				Role:    domain.RoleTestFile,
				Action:  domain.ActionCreate,
			},
		},
	}
	paths := computeRequiredPaths(plan, ctx)
	pathSet := make(map[string]bool)
	for _, p := range paths {
		pathSet[p] = true
	}
	for _, want := range []string{"src/services", "tests", "tests/fixtures"} {
		if !pathSet[want] {
			t.Errorf("expected %q in required paths, got %v", want, paths)
		}
	}
}

// backfill / AC: computeRequiredPaths deduplicates paths across multiple test files
func TestComputeRequiredPaths_Deduplicates(t *testing.T) {
	t.Parallel()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{AbsPath: "/proj/tests/src/services/test_a.py", Role: domain.RoleTestFile, Action: domain.ActionCreate},
			{AbsPath: "/proj/tests/src/services/test_b.py", Role: domain.RoleTestFile, Action: domain.ActionCreate},
		},
	}
	paths := computeRequiredPaths(plan, ctx)
	seen := make(map[string]int)
	for _, p := range paths {
		seen[p]++
	}
	for p, count := range seen {
		if count > 1 {
			t.Errorf("path %q appears %d times, want 1", p, count)
		}
	}
}

// ── generateBootstrap ──────────────────────────────────────────────────────────

// backfill / AC: generateBootstrap returns nil when the plan has no new test files
func TestGenerateBootstrap_NilWhenNoPaths(t *testing.T) {
	ctx := &domain.ProjectContext{Root: t.TempDir(), Language: "python"}
	gf, err := generateBootstrap(&domain.GenerationPlan{}, ctx)
	if err != nil {
		t.Fatalf("generateBootstrap empty plan: %v", err)
	}
	if gf != nil {
		t.Errorf("generateBootstrap with no paths must return nil, got %+v", gf)
	}
}

// backfill / AC: generateBootstrap creates a fresh conftest.py with injected paths (ActionUpdate)
func TestGenerateBootstrap_CreatesFreshConftest(t *testing.T) {
	root := t.TempDir()
	ctx := &domain.ProjectContext{Root: root, Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{
				AbsPath: filepath.Join(root, "tests", "src", "services", "test_payment.py"),
				Role:    domain.RoleTestFile,
				Action:  domain.ActionCreate,
			},
		},
	}

	gf, err := generateBootstrap(plan, ctx)
	if err != nil {
		t.Fatalf("generateBootstrap: %v", err)
	}
	if gf == nil {
		t.Fatal("generateBootstrap must return a GeneratedFile when new paths are needed")
	}
	if gf.Action != domain.ActionUpdate {
		t.Errorf("Action = %q, want %q", gf.Action, domain.ActionUpdate)
	}
	if gf.Role != domain.RoleBootstrap {
		t.Errorf("Role = %q, want %q", gf.Role, domain.RoleBootstrap)
	}
	if !strings.HasSuffix(gf.AbsPath, "conftest.py") {
		t.Errorf("AbsPath = %q, want suffix 'conftest.py'", gf.AbsPath)
	}
	if !strings.Contains(gf.Content, "src/services") {
		t.Error("conftest content must include the derived src directory")
	}
	if !strings.Contains(gf.Content, "sys.path") {
		t.Error("conftest content must reference sys.path")
	}
}

// backfill / AC: generateBootstrap returns ActionSkip when all required paths are already present
func TestGenerateBootstrap_SkipWhenAlreadyPresent(t *testing.T) {
	root := t.TempDir()

	// Existing conftest.py already contains all required paths.
	existing := "import sys\nfrom pathlib import Path\n\npaths_to_add = [\n    \"src/services\",\n    \"tests\",\n    \"tests/fixtures\",\n]\n\n\ndef pytest_configure(config):\n    for p in paths_to_add:\n        path = Path(__file__).parent / p\n        sys.path.insert(0, str(path))\n"
	if err := os.WriteFile(filepath.Join(root, "conftest.py"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &domain.ProjectContext{Root: root, Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{
				AbsPath: filepath.Join(root, "tests", "src", "services", "test_payment.py"),
				Role:    domain.RoleTestFile,
				Action:  domain.ActionCreate,
			},
		},
	}

	gf, err := generateBootstrap(plan, ctx)
	if err != nil {
		t.Fatalf("generateBootstrap: %v", err)
	}
	if gf == nil {
		t.Fatal("generateBootstrap must return a GeneratedFile even for a skip")
	}
	if gf.Action != domain.ActionSkip {
		t.Errorf("Action = %q, want %q", gf.Action, domain.ActionSkip)
	}
}
