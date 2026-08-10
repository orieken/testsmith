package kotlin_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/kotlin"
)

// ── Driver surface ────────────────────────────────────────────────────────────

func TestNew_NotNil(t *testing.T) {
	if kotlin.New() == nil {
		t.Fatal("New() returned nil")
	}
}

func TestLanguage(t *testing.T) {
	if got := kotlin.New().Language(); got != "kotlin" {
		t.Errorf("Language() = %q, want %q", got, "kotlin")
	}
}

func TestFileExtensions(t *testing.T) {
	exts := kotlin.New().FileExtensions()
	want := map[string]bool{".kt": true, ".kts": true}
	if len(exts) != 2 {
		t.Fatalf("FileExtensions() len = %d, want 2", len(exts))
	}
	for _, e := range exts {
		if !want[e] {
			t.Errorf("unexpected extension %q", e)
		}
	}
}

func TestGetTestFrameworkConfig(t *testing.T) {
	cfg := kotlin.New().GetTestFrameworkConfig()
	if cfg.TestFileSuffix != "Test.kt" {
		t.Errorf("TestFileSuffix = %q, want Test.kt", cfg.TestFileSuffix)
	}
	if cfg.FixtureDir != "test" {
		t.Errorf("FixtureDir = %q, want test", cfg.FixtureDir)
	}
}

func TestBodyGenerationPrompt_NonEmpty(t *testing.T) {
	if p := kotlin.New().BodyGenerationPrompt(); p == "" {
		t.Error("BodyGenerationPrompt() returned empty string")
	}
}

func TestListMigrators_Empty(t *testing.T) {
	if got := kotlin.New().ListMigrators(); len(got) != 0 {
		t.Errorf("ListMigrators() len = %d, want 0", len(got))
	}
}

func TestValidateFile_NilIssues(t *testing.T) {
	issues := kotlin.New().ValidateFile("junit5", "mockk", "class FooTest {}")
	if issues != nil {
		t.Errorf("ValidateFile() = %v, want nil", issues)
	}
}

func TestGenerateFixture_ReturnsNil(t *testing.T) {
	f, err := kotlin.New().GenerateFixture("dep", nil, domain.GenerateOpts{})
	if err != nil || f != nil {
		t.Errorf("GenerateFixture() = (%v, %v), want (nil, nil)", f, err)
	}
}

func TestGenerateBootstrap_ReturnsNil(t *testing.T) {
	f, err := kotlin.New().GenerateBootstrap(nil, nil)
	if err != nil || f != nil {
		t.Errorf("GenerateBootstrap() = (%v, %v), want (nil, nil)", f, err)
	}
}

// ── ListAdapters ──────────────────────────────────────────────────────────────

func TestListAdapters_DefaultIsJUnit5(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "kotlin", Metadata: map[string]any{}}
	all, selected := kotlin.New().ListAdapters(ctx)
	if selected == nil {
		t.Fatal("ListAdapters() selected = nil")
	}
	if selected.Framework() != "junit5" {
		t.Errorf("default adapter framework = %q, want junit5", selected.Framework())
	}
	if len(all) < 2 {
		t.Errorf("expected at least 2 adapters, got %d", len(all))
	}
}

func TestListAdapters_SelectsKotest(t *testing.T) {
	ctx := &domain.ProjectContext{
		Language: "kotlin",
		Metadata: map[string]any{"framework": "kotest"},
	}
	_, selected := kotlin.New().ListAdapters(ctx)
	if selected == nil {
		t.Fatal("ListAdapters() selected = nil for kotest")
	}
	if selected.Framework() != "kotest" {
		t.Errorf("framework = %q, want kotest", selected.Framework())
	}
}

// ── LLMContext ────────────────────────────────────────────────────────────────

func TestLLMContext_ContainsLanguage(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "kotlin", Metadata: map[string]any{}}
	vocab := kotlin.New().LLMContext(ctx)
	if vocab["language"] != "kotlin" {
		t.Errorf("LLMContext()[\"language\"] = %q, want kotlin", vocab["language"])
	}
}

// ── DetectProject ─────────────────────────────────────────────────────────────

func TestDetectProject_EmptyDir_ReturnsContext(t *testing.T) {
	dir := t.TempDir()
	ctx, err := kotlin.New().DetectProject(dir)
	// May error (no build file) or succeed with defaults — either is acceptable.
	if err != nil && ctx != nil {
		t.Errorf("DetectProject() returned both error and non-nil context")
	}
}

func TestDetectProject_WithGradleFile(t *testing.T) {
	dir := t.TempDir()
	content := `rootProject.name = "myapp"`
	if err := os.WriteFile(filepath.Join(dir, "settings.gradle.kts"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, err := kotlin.New().DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject() error: %v", err)
	}
	if ctx == nil {
		t.Fatal("DetectProject() returned nil context")
	}
}

// ── ClassifyDependency ────────────────────────────────────────────────────────

func TestClassifyDependency_Stdlib(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "kotlin", Metadata: map[string]any{}}
	imp := domain.ImportInfo{Module: "kotlin.collections.List"}
	got := kotlin.New().ClassifyDependency(imp, ctx)
	if got != domain.DepStdlib {
		t.Errorf("ClassifyDependency(kotlin.*) = %v, want DepStdlib", got)
	}
}

func TestClassifyDependency_External(t *testing.T) {
	ctx := &domain.ProjectContext{Language: "kotlin", Metadata: map[string]any{}}
	imp := domain.ImportInfo{Module: "com.squareup.retrofit2.Retrofit"}
	got := kotlin.New().ClassifyDependency(imp, ctx)
	if got != domain.DepExternal {
		t.Errorf("ClassifyDependency(retrofit) = %v, want DepExternal", got)
	}
}

// ── DeriveTestPath / DeriveModulePath ─────────────────────────────────────────

func TestDeriveTestPath_MavenLayout(t *testing.T) {
	dir := t.TempDir()
	ctx := &domain.ProjectContext{Root: dir, Language: "kotlin"}
	src := filepath.Join(dir, "src", "main", "kotlin", "com", "example", "Foo.kt")
	got, err := kotlin.New().DeriveTestPath(src, ctx)
	if err != nil {
		t.Fatalf("DeriveTestPath() error: %v", err)
	}
	if got == "" {
		t.Error("DeriveTestPath() returned empty string")
	}
}

func TestDeriveModulePath_FallsBackToFilename(t *testing.T) {
	dir := t.TempDir()
	ctx := &domain.ProjectContext{Root: dir, Language: "kotlin"}
	src := filepath.Join(dir, "Foo.kt")
	got, err := kotlin.New().DeriveModulePath(src, ctx)
	if err != nil {
		t.Fatalf("DeriveModulePath() error: %v", err)
	}
	if got == "" {
		t.Error("DeriveModulePath() returned empty string")
	}
}

// ── AnalyzeFile ───────────────────────────────────────────────────────────────

func TestAnalyzeFile_SimpleClass(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "Greeting.kt")
	content := `package com.example

import kotlin.String

class Greeting {
    fun greet(name: String): String = "Hello, $name"
}
`
	if err := os.WriteFile(src, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &domain.ProjectContext{Root: dir, Language: "kotlin", Metadata: map[string]any{}}
	a, err := kotlin.New().AnalyzeFile(src, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile() error: %v", err)
	}
	if a.SourcePath != src {
		t.Errorf("SourcePath = %q, want %q", a.SourcePath, src)
	}
	if len(a.PublicAPI) == 0 {
		t.Error("expected at least one public member")
	}
}

func TestAnalyzeFile_MissingFile_ReturnsError(t *testing.T) {
	ctx := &domain.ProjectContext{Root: t.TempDir(), Metadata: map[string]any{}}
	_, err := kotlin.New().AnalyzeFile("/nonexistent/file.kt", ctx)
	if err == nil {
		t.Error("expected error for missing file, got nil")
	}
}

// ── GenerateTestFile ──────────────────────────────────────────────────────────

func TestGenerateTestFile_ProducesFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src", "main", "kotlin", "com", "example", "Calc.kt")
	if err := os.MkdirAll(filepath.Dir(src), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(src, []byte(`package com.example
fun add(a: Int, b: Int): Int = a + b
`), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &domain.ProjectContext{Root: dir, Language: "kotlin", Metadata: map[string]any{}}
	a, err := kotlin.New().AnalyzeFile(src, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile() error: %v", err)
	}

	f, err := kotlin.New().GenerateTestFile(a, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateTestFile() error: %v", err)
	}
	if f == nil {
		t.Fatal("GenerateTestFile() returned nil")
	}
	if f.Language != "kotlin" {
		t.Errorf("Language = %q, want kotlin", f.Language)
	}
	if f.Content == "" {
		t.Error("Content should not be empty")
	}
}
