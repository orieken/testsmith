package rust_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/rust"
)

// ── Driver surface ────────────────────────────────────────────────────────────

func TestNew_NotNil(t *testing.T) {
	if rust.New() == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLanguage(t *testing.T) {
	if got := rust.New().Language(); got != "rust" {
		t.Errorf("Language() = %q, want %q", got, "rust")
	}
}

func TestFileExtensions(t *testing.T) {
	exts := rust.New().FileExtensions()
	if len(exts) != 1 || exts[0] != ".rs" {
		t.Errorf("FileExtensions() = %v, want [.rs]", exts)
	}
}

func TestGetTestFrameworkConfig(t *testing.T) {
	cfg := rust.New().GetTestFrameworkConfig()
	if cfg.TestFileSuffix != "_test.rs" {
		t.Errorf("TestFileSuffix = %q, want _test.rs", cfg.TestFileSuffix)
	}
	if cfg.TestFuncPrefix != "test_" {
		t.Errorf("TestFuncPrefix = %q, want test_", cfg.TestFuncPrefix)
	}
	if cfg.FixtureDir != "tests" {
		t.Errorf("FixtureDir = %q, want tests", cfg.FixtureDir)
	}
}

func TestBodyGenerationPrompt_NonEmpty(t *testing.T) {
	if p := rust.New().BodyGenerationPrompt(); p == "" {
		t.Error("BodyGenerationPrompt() returned empty string")
	}
}

func TestListMigrators_Empty(t *testing.T) {
	if got := rust.New().ListMigrators(); len(got) != 0 {
		t.Errorf("ListMigrators() len = %d, want 0", len(got))
	}
}

func TestValidateFile_NilIssues(t *testing.T) {
	issues := rust.New().ValidateFile("test", "mockall", "fn test_foo() {}")
	if issues != nil {
		t.Errorf("ValidateFile() = %v, want nil", issues)
	}
}

func TestGenerateFixture_ReturnsNil(t *testing.T) {
	f, err := rust.New().GenerateFixture("dep", nil, domain.GenerateOpts{})
	if err != nil || f != nil {
		t.Errorf("GenerateFixture() = (%v, %v), want (nil, nil)", f, err)
	}
}

func TestGenerateBootstrap_ReturnsNil(t *testing.T) {
	f, err := rust.New().GenerateBootstrap(nil, nil)
	if err != nil || f != nil {
		t.Errorf("GenerateBootstrap() = (%v, %v), want (nil, nil)", f, err)
	}
}

// ── ListAdapters ──────────────────────────────────────────────────────────────

func TestListAdapters_DefaultIsBuiltin(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "rust", Metadata: map[string]any{}}
	all, selected := rust.New().ListAdapters(ctx)
	if selected == nil {
		t.Fatal("ListAdapters() selected = nil")
	}
	if selected.Framework() != "test" {
		t.Errorf("default adapter framework = %q, want test", selected.Framework())
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 adapters, got %d", len(all))
	}
}

func TestListAdapters_SelectsRstest(t *testing.T) {
	ctx := &domain.ProjectContext{
		Language: "rust",
		Metadata: map[string]any{"framework": "rstest"},
	}
	_, selected := rust.New().ListAdapters(ctx)
	if selected == nil {
		t.Fatal("ListAdapters() selected = nil for rstest")
	}
	if selected.Framework() != "rstest" {
		t.Errorf("framework = %q, want rstest", selected.Framework())
	}
}

// ── LLMContext ────────────────────────────────────────────────────────────────

func TestLLMContext_ContainsLanguage(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "rust", Metadata: map[string]any{}}
	vocab := rust.New().LLMContext(ctx)
	if vocab["language"] != "rust" {
		t.Errorf("LLMContext()[\"language\"] = %q, want rust", vocab["language"])
	}
}

// ── DetectProject ─────────────────────────────────────────────────────────────

func TestDetectProject_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	ctx, err := rust.New().DetectProject(dir)
	// May error (no Cargo.toml) or return a fallback — either is valid.
	if err != nil && ctx != nil {
		t.Errorf("DetectProject() returned both error and non-nil context")
	}
}

func TestDetectProject_WithCargoToml(t *testing.T) {
	dir := t.TempDir()
	content := "[package]\nname = \"myapp\"\nversion = \"0.1.0\"\nedition = \"2021\"\n"
	if err := os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, err := rust.New().DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject() error: %v", err)
	}
	if ctx == nil {
		t.Fatal("DetectProject() returned nil context")
	}
}

// ── ClassifyDependency ────────────────────────────────────────────────────────

func TestClassifyDependency_Stdlib(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "rust", Metadata: map[string]any{}}
	imp := domain.ImportInfo{Module: "std::collections::HashMap"}
	got := rust.New().ClassifyDependency(imp, ctx)
	if got != domain.DepStdlib {
		t.Errorf("ClassifyDependency(std::*) = %v, want DepStdlib", got)
	}
}

func TestClassifyDependency_Internal(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "rust", Metadata: map[string]any{}}
	imp := domain.ImportInfo{Module: "crate::models::User"}
	got := rust.New().ClassifyDependency(imp, ctx)
	if got != domain.DepInternal {
		t.Errorf("ClassifyDependency(crate::*) = %v, want DepInternal", got)
	}
}

func TestClassifyDependency_External(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "rust", Metadata: map[string]any{}}
	imp := domain.ImportInfo{Module: "serde::Deserialize"}
	got := rust.New().ClassifyDependency(imp, ctx)
	if got != domain.DepExternal {
		t.Errorf("ClassifyDependency(serde) = %v, want DepExternal", got)
	}
}

// ── DeriveTestPath / DeriveModulePath ─────────────────────────────────────────

func TestDeriveTestPath_IntegrationConvention(t *testing.T) {
	dir := t.TempDir()
	ctx := &domain.ProjectContext{Root: dir, Language: "rust"}
	src := filepath.Join(dir, "src", "lib.rs")
	got, err := rust.New().DeriveTestPath(src, ctx)
	if err != nil {
		t.Fatalf("DeriveTestPath() error: %v", err)
	}
	if got == "" {
		t.Error("DeriveTestPath() returned empty string")
	}
}

func TestDeriveModulePath_FallsBackToFilename(t *testing.T) {
	dir := t.TempDir()
	ctx := &domain.ProjectContext{Root: dir, Language: "rust"}
	src := filepath.Join(dir, "src", "main.rs")
	got, err := rust.New().DeriveModulePath(src, ctx)
	if err != nil {
		t.Fatalf("DeriveModulePath() error: %v", err)
	}
	if got == "" {
		t.Error("DeriveModulePath() returned empty string")
	}
}

// ── AnalyzeFile ───────────────────────────────────────────────────────────────

func TestAnalyzeFile_SimpleStruct(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "lib.rs")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	content := `use std::fmt;

pub struct Greeter {
    name: String,
}

impl Greeter {
    pub fn new(name: &str) -> Self {
        Greeter { name: name.to_string() }
    }

    pub fn greet(&self) -> String {
        format!("Hello, {}!", self.name)
    }
}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &domain.ProjectContext{Root: dir, Language: "rust", Metadata: map[string]any{}}
	a, err := rust.New().AnalyzeFile(src, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile() error: %v", err)
	}
	if a.SourcePath != src {
		t.Errorf("SourcePath = %q, want %q", a.SourcePath, src)
	}
	if len(a.PublicAPI) == 0 {
		t.Error("expected at least one public member (Greeter)")
	}
}

func TestAnalyzeFile_MissingFile_ReturnsError(t *testing.T) {
	ctx := &domain.ProjectContext{Root: t.TempDir(), Metadata: map[string]any{}}
	_, err := rust.New().AnalyzeFile("/nonexistent/file.rs", ctx)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ── GenerateTestFile ──────────────────────────────────────────────────────────

func TestGenerateTestFile_ProducesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "math.rs")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte("pub fn add(a: i32, b: i32) -> i32 { a + b }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &domain.ProjectContext{Root: dir, Language: "rust", Metadata: map[string]any{}}
	a, err := rust.New().AnalyzeFile(src, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile() error: %v", err)
	}

	f, err := rust.New().GenerateTestFile(a, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateTestFile() error: %v", err)
	}
	if f == nil {
		t.Fatal("GenerateTestFile() returned nil")
	}
	if f.Language != "rust" {
		t.Errorf("Language = %q, want rust", f.Language)
	}
	if f.Content == "" {
		t.Error("Content should not be empty")
	}
}
