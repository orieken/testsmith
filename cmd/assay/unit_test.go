package main

// In-process unit tests for the cmd/assay package.
//
// Pure-function tests use t.Parallel(). Tests that call run* functions rely on
// testChdir() and must NOT call t.Parallel() because os.Chdir is process-global.
// The registry is initialised once via init() in subprocess_test.go.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
)

// ── helpers ───────────────────────────────────────────────────────────────────

// testChdir changes the working directory to dir for the duration of the test.
// It restores the original directory via t.Cleanup regardless of pass/fail.
func testChdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func goDriver() domain.LanguageDriver {
	d, err := reg.ForLanguage("go")
	if err != nil {
		panic("go driver not registered: " + err.Error())
	}
	return d
}

func pythonAdapter() domain.TestAdapter {
	d, err := reg.ForLanguage("python")
	if err != nil {
		panic("python driver not registered: " + err.Error())
	}
	ctx := &domain.ProjectContext{Language: "python", Metadata: map[string]any{}}
	_, selected := d.ListAdapters(ctx)
	return selected
}

func mkFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ── pure functions: config.go ─────────────────────────────────────────────────

func TestParseChoice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		raw  string
		max  int
		want int
	}{
		{"1", 3, 0},
		{"2", 3, 1},
		{"3", 3, 2},
		{"0", 3, -1},  // below range
		{"4", 3, -1},  // above range
		{"", 3, -1},   // empty
		{"a", 3, -1},  // non-digit
		{"2a", 3, -1}, // mixed
		{"1", 1, 0},   // single choice
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.raw, func(t *testing.T) {
			t.Parallel()
			if got := parseChoice(tt.raw, tt.max); got != tt.want {
				t.Errorf("parseChoice(%q, %d) = %d, want %d", tt.raw, tt.max, got, tt.want)
			}
		})
	}
}

func TestAdapterIndex(t *testing.T) {
	t.Parallel()

	d := goDriver()
	ctx := &domain.ProjectContext{Language: "go", Metadata: map[string]any{}}
	available, selected := d.ListAdapters(ctx)

	if len(available) == 0 {
		t.Skip("no adapters registered")
	}
	idx := adapterIndex(available, selected)
	if idx < 0 || idx >= len(available) {
		t.Errorf("adapterIndex out of bounds: %d (len=%d)", idx, len(available))
	}
	// adapterIndex of a non-existent adapter returns 0 (safe fallback).
	bogus, _ := reg.ForLanguage("python")
	_, bogusSelected := bogus.ListAdapters(&domain.ProjectContext{Language: "python", Metadata: map[string]any{}})
	if got := adapterIndex(available, bogusSelected); got != 0 {
		t.Errorf("adapterIndex for unknown adapter = %d, want 0", got)
	}
}

func TestFormatList(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		items []string
		want  string
	}{
		{"nil returns none", nil, "(none)"},
		{"empty returns none", []string{}, "(none)"},
		{"one item", []string{"a"}, "a"},
		{"four items joined", []string{"a", "b", "c", "d"}, "a, b, c, d"},
		{"five items truncated", []string{"a", "b", "c", "d", "e"}, "a, b, c, d, … (5 total)"},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := formatList(tt.items); got != tt.want {
				t.Errorf("formatList() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestOrderedLangKeys(t *testing.T) {
	t.Parallel()

	langs := map[string]config.LanguageConfig{
		"python":     {},
		"go":         {},
		"typescript": {},
	}

	keys := orderedLangKeys(langs, "go")
	if len(keys) == 0 || keys[0] != "go" {
		t.Errorf("orderedLangKeys first = %q, want \"go\"", keys[0])
	}
	if len(keys) == 3 && keys[1] > keys[2] {
		t.Errorf("remaining keys not sorted: %v", keys[1:])
	}

	// Unknown primary: still returns all keys.
	if got := orderedLangKeys(langs, "rust"); len(got) != 3 {
		t.Errorf("orderedLangKeys len = %d, want 3", len(got))
	}
}

func TestDivider(t *testing.T) {
	t.Parallel()

	empty := divider("")
	if !strings.Contains(empty, "─") {
		t.Errorf("divider(\"\") = %q, want rule chars", empty)
	}
	labelled := divider("Preview")
	if !strings.Contains(labelled, "Preview") {
		t.Errorf("divider(\"Preview\") = %q, missing label", labelled)
	}
}

// ── selectionReason ───────────────────────────────────────────────────────────

func TestSelectionReason(t *testing.T) {
	t.Parallel()

	adapter := pythonAdapter()

	tests := []struct {
		name     string
		cfg      *config.Config
		ctx      *domain.ProjectContext
		selected domain.TestAdapter
		want     string
	}{
		{
			name: "from config when language config is set",
			cfg: &config.Config{
				ConfigPath: "/some/.assay.yaml",
				Languages:  map[string]config.LanguageConfig{"python": {Framework: adapter.Framework()}},
			},
			ctx:      &domain.ProjectContext{Language: "python", Metadata: map[string]any{}},
			selected: adapter,
			want:     "from config",
		},
		{
			name: "auto-detected when metadata matches selected",
			cfg:  &config.Config{},
			ctx: &domain.ProjectContext{
				Language: "python",
				Metadata: map[string]any{
					"framework":    adapter.Framework(),
					"mock_library": adapter.MockLibrary(),
				},
			},
			selected: adapter,
			want:     "auto-detected",
		},
		{
			name:     "default when neither config nor metadata match",
			cfg:      &config.Config{},
			ctx:      &domain.ProjectContext{Language: "python", Metadata: map[string]any{}},
			selected: adapter,
			want:     "default",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := selectionReason(tt.cfg, tt.ctx, tt.selected); got != tt.want {
				t.Errorf("selectionReason() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ── renderConfigYAML ──────────────────────────────────────────────────────────

func TestRenderConfigYAML(t *testing.T) {
	t.Parallel()

	adapter := pythonAdapter()

	tests := []struct {
		name        string
		lang        string
		llmProvider string
		wantParts   []string
	}{
		{
			name:        "anthropic emits ANTHROPIC_API_KEY",
			lang:        "python",
			llmProvider: "anthropic",
			wantParts:   []string{"language: python", "enabled: true", "ANTHROPIC_API_KEY"},
		},
		{
			name:        "openai emits OPENAI_API_KEY",
			lang:        "go",
			llmProvider: "openai",
			wantParts:   []string{"language: go", "OPENAI_API_KEY"},
		},
		{
			name:        "ollama emits base_url",
			lang:        "typescript",
			llmProvider: "ollama",
			wantParts:   []string{"language: typescript", "http://localhost:11434"},
		},
		{
			name:        "exclude_dirs are rendered",
			lang:        "java",
			llmProvider: "anthropic",
			wantParts:   []string{"exclude_dirs:", "- node_modules"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := renderConfigYAML(
				tt.lang, adapter,
				true, tt.llmProvider, "some-model",
				[]string{"node_modules", ".git"},
			)
			for _, part := range tt.wantParts {
				if !strings.Contains(result, part) {
					t.Errorf("renderConfigYAML missing %q\ngot:\n%s", part, result)
				}
			}
		})
	}
}

// ── walkTestFiles ─────────────────────────────────────────────────────────────

func TestWalkTestFiles_FindsTestFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	mkFile(t, filepath.Join(root, "foo_test.go"), "package main")
	mkFile(t, filepath.Join(root, "bar_test.go"), "package main")
	mkFile(t, filepath.Join(root, "main.go"), "package main")

	// Files in skip dirs must be excluded.
	mkFile(t, filepath.Join(root, "node_modules", "pkg", "util_test.go"), "// skip")
	mkFile(t, filepath.Join(root, ".hidden", "skip_test.go"), "// skip")

	files, err := walkTestFiles(root, goDriver())
	if err != nil {
		t.Fatalf("walkTestFiles: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("walkTestFiles found %d file(s), want 2: %v", len(files), files)
	}
	for _, f := range files {
		if strings.Contains(f, "node_modules") || strings.Contains(f, ".hidden") {
			t.Errorf("walkTestFiles returned path from skip dir: %s", f)
		}
	}
}

func TestWalkTestFiles_EmptyRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	files, err := walkTestFiles(root, goDriver())
	if err != nil {
		t.Fatalf("walkTestFiles: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("empty dir produced %d files", len(files))
	}
}

// ── run* functions via t.Chdir ────────────────────────────────────────────────

func TestRunAdaptersList_ExplicitLang(t *testing.T) {
	testChdir(t, t.TempDir())
	if err := runAdaptersList("python"); err != nil {
		t.Fatalf("runAdaptersList(python): %v", err)
	}
}

func TestRunAdaptersList_UnknownLang_Errors(t *testing.T) {
	testChdir(t, t.TempDir())
	if err := runAdaptersList("brainfuck"); err == nil {
		t.Error("runAdaptersList(unknown) should return error")
	}
}

func TestRunConfigShow_ExplicitLang(t *testing.T) {
	testChdir(t, t.TempDir())
	if err := runConfigShow("go"); err != nil {
		t.Fatalf("runConfigShow(go): %v", err)
	}
}

func TestRunConfigShow_NoProject(t *testing.T) {
	testChdir(t, t.TempDir())
	if err := runConfigShow(""); err != nil {
		t.Fatalf("runConfigShow(\"\"): %v", err)
	}
}

func TestRunConfigShow_WithConfigFile(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	os.WriteFile(filepath.Join(dir, ".assay.yaml"), []byte("language: python\ntest_root: mytests/\n"), 0o644)
	if err := runConfigShow("python"); err != nil {
		t.Fatalf("runConfigShow(python) with config: %v", err)
	}
}

func TestRunInit_LangHint(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	dryRun = false
	if err := runInit("go", false); err != nil {
		t.Fatalf("runInit(go): %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay.yaml")); err != nil {
		t.Error(".assay.yaml not created")
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay", "patterns")); err != nil {
		t.Error(".assay/patterns/ not created")
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay", "patterns", "README.md")); err != nil {
		t.Error(".assay/patterns/README.md not created")
	}
}

func TestRunInit_DryRun(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	dryRun = true
	t.Cleanup(func() { dryRun = false })
	if err := runInit("python", false); err != nil {
		t.Fatalf("runInit dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay.yaml")); !os.IsNotExist(err) {
		t.Error("dry-run must not create .assay.yaml")
	}
}

func TestRunInit_AlreadyExists(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	existing := "language: typescript\n"
	os.WriteFile(filepath.Join(dir, ".assay.yaml"), []byte(existing), 0o644)
	dryRun = false
	if err := runInit("typescript", false); err != nil {
		t.Fatalf("runInit when file exists: %v", err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, ".assay.yaml"))
	if string(data) != existing {
		t.Errorf("runInit overwrote .assay.yaml: got %q", string(data))
	}
}

func TestRunConfigInit_YesMode(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	dryRun = false
	if err := runConfigInit("go", true, ".assay.yaml"); err != nil {
		t.Fatalf("runConfigInit yes-mode: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dir, ".assay.yaml"))
	if err != nil {
		t.Fatal(".assay.yaml not created")
	}
	if !strings.Contains(string(data), "language: go") {
		t.Errorf("config missing language:\n%s", string(data))
	}
}

func TestRunConfigInit_DryRun(t *testing.T) {
	dir := t.TempDir()
	testChdir(t, dir)
	dryRun = true
	t.Cleanup(func() { dryRun = false })
	if err := runConfigInit("python", true, ".assay.yaml"); err != nil {
		t.Fatalf("runConfigInit dry-run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".assay.yaml")); !os.IsNotExist(err) {
		t.Error("dry-run must not create .assay.yaml")
	}
}

// ── parseLearnResponse ────────────────────────────────────────────────────────

func TestParseLearnResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		wantSlug string
		wantBody string
	}{
		{
			name:     "well-formed response with separator",
			input:    "filename: mock-database-sqlmock\n\n---\n\nUse sqlmock for DB tests.",
			wantSlug: "mock-database-sqlmock",
			wantBody: "Use sqlmock for DB tests.",
		},
		{
			name:     "filename line with trailing whitespace",
			input:    "filename:  http-handler-httptest  \n\n---\n\nPattern body.",
			wantSlug: "http-handler-httptest",
			wantBody: "Pattern body.",
		},
		{
			name:     "no separator between filename and body",
			input:    "filename: table-driven-errors\n\nPattern body without separator.",
			wantSlug: "table-driven-errors",
			wantBody: "Pattern body without separator.",
		},
		{
			name:     "no filename line returns empty slug and full raw body",
			input:    "Some pattern content with no filename header.",
			wantSlug: "",
			wantBody: "Some pattern content with no filename header.",
		},
		{
			name:     "empty input returns empty slug and empty body",
			input:    "",
			wantSlug: "",
			wantBody: "",
		},
		{
			name:     "multiline body preserved after separator",
			input:    "filename: postgres-testcontainers\n\n---\n\nLine one.\n\nLine two.",
			wantSlug: "postgres-testcontainers",
			wantBody: "Line one.\n\nLine two.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			gotSlug, gotBody := parseLearnResponse(tt.input)
			if gotSlug != tt.wantSlug {
				t.Errorf("slug = %q, want %q", gotSlug, tt.wantSlug)
			}
			if gotBody != tt.wantBody {
				t.Errorf("body = %q, want %q", gotBody, tt.wantBody)
			}
		})
	}
}

// ── slugFromPath ──────────────────────────────────────────────────────────────

func TestSlugFromPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "Go test file", input: "internal/payment/payment_test.go", want: "payment-pattern"},
		{name: "TypeScript test file", input: "src/services/user.test.ts", want: "user-pattern"},
		{name: "TypeScript spec file", input: "src/components/button.spec.ts", want: "button-pattern"},
		{name: "JavaScript test file", input: "lib/utils.test.js", want: "utils-pattern"},
		{name: "Java test file", input: "PaymentServiceTest.java", want: "paymentservice-pattern"},
		{name: "Swift test file", input: "UserProfileViewModelTests.swift", want: "userprofileviewmodel-pattern"},
		{name: "no known suffix keeps extension in slug", input: "internal/auth/handler.go", want: "handler-go-pattern"},
		{name: "path with multiple directories uses basename only", input: "a/b/c/order_test.go", want: "order-pattern"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := slugFromPath(tt.input); got != tt.want {
				t.Errorf("slugFromPath(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
