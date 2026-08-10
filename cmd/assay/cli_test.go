// Package main_test contains CLI integration tests.
// TestMain uses the TestHelperProcess pattern: the test binary re-invokes
// itself via TestSubprocessMain so coverage counters stay in the same
// instrumented binary and can be collected via GOCOVERDIR.
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// binaryPath is set by TestMain and shared across all tests.
var binaryPath string

// testdataDir is the absolute path to v2/testdata.
var testdataDir string

func TestMain(m *testing.M) {
	// Short-circuit: when we are the subprocess, just run the requested test.
	if os.Getenv("GO_TESTSMITH_SUBPROCESS") == "1" {
		os.Exit(m.Run())
	}

	// Locate the module root from this file's path.
	_, file, _, _ := runtime.Caller(0)
	moduleRoot := filepath.Join(filepath.Dir(file), "..", "..")
	testdataDir = filepath.Join(moduleRoot, "testdata")

	// Use the test binary itself as the CLI subprocess so that coverage
	// counters live in the same instrumented binary (collected via GOCOVERDIR).
	binaryPath = os.Args[0]

	os.Exit(m.Run())
}

// run executes the test binary as a subprocess via the TestHelperProcess pattern.
// It returns stdout+stderr combined, and the exit code.
func run(t *testing.T, dir string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(binaryPath,
		"-test.run=TestSubprocessMain",
		"-test.v=false",
		"-test.count=1",
	)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"GO_TESTSMITH_SUBPROCESS=1",
		"GO_TESTSMITH_ARGS="+strings.Join(args, "\x1e"),
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			exitCode = ee.ExitCode()
		} else {
			t.Fatalf("exec error: %v", err)
		}
	}
	return buf.String(), exitCode
}

// mustRun asserts exit code 0 and returns combined output.
func mustRun(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, code := run(t, dir, args...)
	if code != 0 {
		t.Fatalf("command %v exited %d\noutput:\n%s", args, code, out)
	}
	return out
}

// copyDir recursively copies src into dst (shallow enough for test fixtures).
func copyDir(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatalf("copyDir %s -> %s: %v", src, dst, err)
	}
}

// ---- version ----------------------------------------------------------------

func TestVersion(t *testing.T) {
	out := mustRun(t, t.TempDir(), "version")
	if !strings.HasPrefix(out, "assay ") {
		t.Errorf("version output = %q, want prefix 'assay '", out)
	}
}

// ---- adapters ---------------------------------------------------------------

func TestAdapters_Python(t *testing.T) {
	dir := filepath.Join(testdataDir, "python")
	out := mustRun(t, dir, "adapters", "list")
	if !strings.Contains(out, "pytest") {
		t.Errorf("adapters list output missing 'pytest':\n%s", out)
	}
}

func TestAdapters_LangFlag(t *testing.T) {
	out := mustRun(t, t.TempDir(), "adapters", "list", "--lang", "typescript")
	if !strings.Contains(out, "jest") {
		t.Errorf("adapters list --lang typescript missing 'jest':\n%s", out)
	}
}

func TestAdapters_UnknownLang_Fails(t *testing.T) {
	_, code := run(t, t.TempDir(), "adapters", "list", "--lang", "cobol")
	if code == 0 {
		t.Errorf("expected non-zero exit for unknown language, got 0")
	}
}

// ---- config show ------------------------------------------------------------

func TestConfigShow_Defaults(t *testing.T) {
	out := mustRun(t, t.TempDir(), "config", "show")
	for _, want := range []string{"test_root", "fixture_dir", "llm"} {
		if !strings.Contains(out, want) {
			t.Errorf("config show missing %q in output:\n%s", want, out)
		}
	}
}

func TestConfigShow_WithFile(t *testing.T) {
	dir := t.TempDir()
	cfg := "language: python\ntest_root: mytests/\n"
	if err := os.WriteFile(filepath.Join(dir, ".assay.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	out := mustRun(t, dir, "config", "show")
	if !strings.Contains(out, "mytests/") {
		t.Errorf("config show did not reflect .assay.yaml:\n%s", out)
	}
}

// ---- generate ---------------------------------------------------------------

func TestGenerate_SingleFile_Python(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	src := filepath.Join(dir, "src", "services", "payment.py")
	out := mustRun(t, dir, "generate", src)

	if !strings.Contains(out, "created") && !strings.Contains(out, "skipped") {
		t.Errorf("generate output unexpected:\n%s", out)
	}
}

func TestGenerate_All_Go(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "golang"), dir)

	out := mustRun(t, dir, "generate", "--all")

	if !strings.Contains(out, "Processed") {
		t.Errorf("generate --all output missing summary:\n%s", out)
	}
	// At least one test file should have been created.
	if !strings.Contains(out, "created") {
		t.Errorf("generate --all created nothing:\n%s", out)
	}
}

func TestGenerate_All_DryRun(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	out := mustRun(t, dir, "generate", "--all", "--dry-run")

	// Dry-run must not create any real files.
	var createdFiles []string
	filepath.WalkDir(dir, func(p string, d os.DirEntry, _ error) error {
		if !d.IsDir() && strings.Contains(p, "test_") {
			createdFiles = append(createdFiles, p)
		}
		return nil
	})
	if len(createdFiles) > 0 {
		t.Errorf("dry-run created files: %v", createdFiles)
	}
	_ = out
}

func TestGenerate_Workspace(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "workspace"), dir)

	out := mustRun(t, dir, "generate", "--all")

	if !strings.Contains(out, "workspace: api") {
		t.Errorf("workspace mode did not process 'api':\n%s", out)
	}
	if !strings.Contains(out, "workspace: frontend") {
		t.Errorf("workspace mode did not process 'frontend':\n%s", out)
	}
}

func TestGenerate_Workspace_Filter(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "workspace"), dir)

	out := mustRun(t, dir, "generate", "--all", "--workspace", "api")

	if !strings.Contains(out, "workspace: api") {
		t.Errorf("workspace filter did not process 'api':\n%s", out)
	}
	if strings.Contains(out, "workspace: frontend") {
		t.Errorf("workspace filter included 'frontend' when it should be skipped:\n%s", out)
	}
}

func TestGenerate_PathFlag(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	subdir := filepath.Join(dir, "src", "services")
	out := mustRun(t, dir, "generate", "--path", subdir)

	if !strings.Contains(out, "Processed") {
		t.Errorf("generate --path output missing summary:\n%s", out)
	}
}

// ---- validate ---------------------------------------------------------------

func TestValidate_CleanProject_Go(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "golang"), dir)

	// Generate tests first so validate has something to scan.
	mustRun(t, dir, "generate", "--all")
	out := mustRun(t, dir, "validate")
	// No errors expected for freshly generated tests.
	if strings.Contains(out, "error(s)") && !strings.Contains(out, "0 error(s)") {
		t.Errorf("validate reported errors on clean project:\n%s", out)
	}
}

func TestValidate_LangFlag(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	// validate with an explicit lang should not fail even with no test files
	out, code := run(t, dir, "validate", "--lang", "python")
	// exit 0 when there are no errors (may have warnings)
	if code != 0 && strings.Contains(out, "0 error(s)") {
		t.Errorf("validate --lang python: exit %d\n%s", code, out)
	}
}

func TestValidate_Workspace(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "workspace"), dir)

	out := mustRun(t, dir, "validate")

	if !strings.Contains(out, "workspace: api") {
		t.Errorf("validate workspace mode missing 'api':\n%s", out)
	}
	if !strings.Contains(out, "workspace: frontend") {
		t.Errorf("validate workspace mode missing 'frontend':\n%s", out)
	}
}

// ---- migrate ----------------------------------------------------------------

func TestMigrate_JestToVitest(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "typescript"), dir)

	// Use the pre-baked jest test file as migration input.
	src := filepath.Join(dir, "src", "services", "payment.jest.test.ts")
	out := mustRun(t, dir, "migrate", "--from", "jest", "--to", "vitest", "--path", filepath.Dir(src))

	if !strings.Contains(out, "migrated") {
		t.Errorf("migrate output missing 'migrated':\n%s", out)
	}

	// Verify vi. appears in the rewritten file.
	content, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read migrated file: %v", err)
	}
	if !strings.Contains(string(content), "vi.") {
		t.Errorf("migration did not replace jest. with vi.:\n%s", string(content))
	}
}

func TestMigrate_InvalidPair_Fails(t *testing.T) {
	_, code := run(t, t.TempDir(), "migrate", "--from", "jest", "--to", "junit5")
	if code == 0 {
		t.Errorf("expected non-zero exit for invalid migration pair, got 0")
	}
}

func TestMigrate_DryRun_NoChanges(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "typescript"), dir)

	src := filepath.Join(dir, "src", "services", "payment.jest.test.ts")
	original, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}

	mustRun(t, dir, "migrate", "--from", "jest", "--to", "vitest",
		"--path", filepath.Dir(src), "--dry-run")

	after, _ := os.ReadFile(src)
	if !bytes.Equal(original, after) {
		t.Errorf("dry-run modified file on disk")
	}
}

// ---- gaps -------------------------------------------------------------------

func TestGaps_DryRun_Python(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	out := mustRun(t, dir, "gaps", "--dry-run")

	if !strings.Contains(out, "Coverage Gap Report") {
		t.Errorf("gaps --dry-run missing report header:\n%s", out)
	}
	// Source files exist but no tests — expect gaps.
	if !strings.Contains(out, "no_test") {
		t.Errorf("gaps --dry-run expected no_test entries:\n%s", out)
	}
}

func TestGaps_TopFlag(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	out := mustRun(t, dir, "gaps", "--dry-run", "--top", "1")
	// Table should contain exactly one data row (after header + separator).
	rows := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "| ") && !strings.Contains(line, "Priority") && !strings.Contains(line, "---") {
			rows++
		}
	}
	if rows != 1 {
		t.Errorf("gaps --top 1 produced %d row(s), want 1\noutput:\n%s", rows, out)
	}
}

func TestGaps_WritesFile(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	report := filepath.Join(dir, "report.md")
	mustRun(t, dir, "gaps", "--output", report)

	data, err := os.ReadFile(report)
	if err != nil {
		t.Fatalf("report not created: %v", err)
	}
	if !strings.Contains(string(data), "Coverage Gap Report") {
		t.Errorf("report file missing header:\n%s", string(data))
	}
}

// ---- graph ------------------------------------------------------------------

func TestGraph_DryRun_Go(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "golang"), dir)

	out := mustRun(t, dir, "graph", "--dry-run")

	if !strings.Contains(out, "Dependency Graph") {
		t.Errorf("graph --dry-run missing header:\n%s", out)
	}
	if !strings.Contains(out, "Coupling Score") {
		t.Errorf("graph --dry-run missing metrics table:\n%s", out)
	}
}

func TestGraph_WritesFile(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "golang"), dir)

	output := filepath.Join(dir, "graph.md")
	mustRun(t, dir, "graph", "--output", output)

	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("graph file not created: %v", err)
	}
	if !strings.Contains(string(data), "mermaid") {
		t.Errorf("graph file missing mermaid block:\n%s", string(data))
	}
}

// ---- prune ------------------------------------------------------------------

func TestPrune_NoUnused(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	out := mustRun(t, dir, "prune")
	if !strings.Contains(out, "No unused fixtures") {
		t.Errorf("prune output = %q, want 'No unused fixtures'", out)
	}
}

func TestPrune_DryRun_DoesNotDelete(t *testing.T) {
	dir := t.TempDir()
	copyDir(t, filepath.Join(testdataDir, "python"), dir)

	// Plant a stale fixture file.
	fixtureDir := filepath.Join(dir, "tests", "fixtures")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(fixtureDir, "nonexistent_dep_fixture.py")
	if err := os.WriteFile(stale, []byte("# stale"), 0o644); err != nil {
		t.Fatal(err)
	}

	out := mustRun(t, dir, "prune")

	if !strings.Contains(out, "would delete") {
		t.Errorf("prune dry-run should report 'would delete':\n%s", out)
	}
	if _, err := os.Stat(stale); os.IsNotExist(err) {
		t.Errorf("prune dry-run deleted the file when it should not have")
	}
}

// ---- init -------------------------------------------------------------------

func TestInit_CreatesConfigFile(t *testing.T) {
	dir := t.TempDir()
	// Copy just the python pyproject.toml so detection works.
	src := filepath.Join(testdataDir, "python", "pyproject.toml")
	dst := filepath.Join(dir, "pyproject.toml")
	data, _ := os.ReadFile(src)
	os.WriteFile(dst, data, 0o644)

	mustRun(t, dir, "init")

	cfgPath := filepath.Join(dir, ".assay.yaml")
	content, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf(".assay.yaml not created: %v", err)
	}
	body := string(content)
	if !strings.Contains(body, "language: python") {
		t.Errorf("init config missing 'language: python':\n%s", body)
	}
	if !strings.Contains(body, "test_root:") {
		t.Errorf("init config missing 'test_root:':\n%s", body)
	}
}

func TestInit_DryRun_NoFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(testdataDir, "python", "pyproject.toml")
	data, _ := os.ReadFile(src)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), data, 0o644)

	out := mustRun(t, dir, "init", "--dry-run")

	if !strings.Contains(out, ".assay.yaml") {
		t.Errorf("init dry-run output missing config mention:\n%s", out)
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay.yaml")); !os.IsNotExist(err) {
		t.Errorf("init dry-run should not create .assay.yaml")
	}
}

func TestInit_AlreadyExists_Skips(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(testdataDir, "python", "pyproject.toml")
	data, _ := os.ReadFile(src)
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), data, 0o644)

	existing := "language: go\n"
	os.WriteFile(filepath.Join(dir, ".assay.yaml"), []byte(existing), 0o644)

	out := mustRun(t, dir, "init")
	if !strings.Contains(out, "already exists") {
		t.Errorf("init should report already-exists:\n%s", out)
	}
	// Original content must be preserved.
	preserved, _ := os.ReadFile(filepath.Join(dir, ".assay.yaml"))
	if string(preserved) != existing {
		t.Errorf("init overwrote existing .assay.yaml")
	}
}

// ---- completion -------------------------------------------------------------

func TestCompletion_Bash(t *testing.T) {
	out := mustRun(t, t.TempDir(), "completion", "bash")
	if !strings.Contains(out, "bash completion") {
		t.Errorf("completion bash missing header:\n%s", out[:min(200, len(out))])
	}
}

func TestCompletion_Zsh(t *testing.T) {
	out := mustRun(t, t.TempDir(), "completion", "zsh")
	if !strings.Contains(out, "#compdef") {
		t.Errorf("completion zsh missing #compdef:\n%s", out[:min(200, len(out))])
	}
}

func TestCompletion_InvalidShell_Fails(t *testing.T) {
	_, code := run(t, t.TempDir(), "completion", "fish2")
	if code == 0 {
		t.Errorf("expected non-zero exit for invalid shell, got 0")
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
