package analysis_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/domain"
)

// ---- fakes ------------------------------------------------------------------

type fakeDriver struct {
	ext        string
	testSuffix string
}

func (f *fakeDriver) Language() string                                      { return "fake" }
func (f *fakeDriver) FileExtensions() []string                              { return []string{f.ext} }
func (f *fakeDriver) BodyGenerationPrompt() string                          { return "" }
func (f *fakeDriver) LLMContext(_ *domain.ProjectContext) map[string]string { return nil }
func (f *fakeDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{TestFileSuffix: f.testSuffix}
}
func (f *fakeDriver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return &domain.ProjectContext{Root: dir, Language: "fake"}, nil
}
func (f *fakeDriver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return &domain.SourceAnalysis{SourcePath: path, Project: ctx}, nil
}
func (f *fakeDriver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}
func (f *fakeDriver) DeriveTestPath(src string, ctx *domain.ProjectContext) (string, error) {
	ext := filepath.Ext(src)
	return src[:len(src)-len(ext)] + f.testSuffix, nil
}
func (f *fakeDriver) DeriveModulePath(src string, ctx *domain.ProjectContext) (string, error) {
	return src, nil
}
func (f *fakeDriver) GenerateTestFile(a *domain.SourceAnalysis, o domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (f *fakeDriver) GenerateFixture(dep string, a *domain.SourceAnalysis, o domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (f *fakeDriver) GenerateBootstrap(p *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (f *fakeDriver) ListAdapters(_ *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return nil, nil
}
func (f *fakeDriver) ListMigrators() []domain.Migrator                     { return nil }
func (f *fakeDriver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }

// ---- helpers ----------------------------------------------------------------

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// ---- DiscoverUntested -------------------------------------------------------

func TestDiscoverUntested_FindsFiles(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	writeFile(t, filepath.Join(dir, "payment.fake"), "// source")
	writeFile(t, filepath.Join(dir, "util.fake"), "// source")
	// util has a test already.
	writeFile(t, filepath.Join(dir, "util_test.fake"), "// test")

	files, err := p.DiscoverUntested(dir, ctx)
	if err != nil {
		t.Fatalf("DiscoverUntested: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 untested file, got %d: %v", len(files), files)
	}
}

func TestDiscoverUntested_ExcludesDirs(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir, ExcludeDirs: []string{"vendor"}}

	writeFile(t, filepath.Join(dir, "main.fake"), "// source")
	writeFile(t, filepath.Join(dir, "vendor", "dep.fake"), "// vendor dep")

	files, err := p.DiscoverUntested(dir, ctx)
	if err != nil {
		t.Fatalf("DiscoverUntested: %v", err)
	}
	for _, f := range files {
		if filepath.Base(filepath.Dir(f)) == "vendor" {
			t.Errorf("vendor file should be excluded: %s", f)
		}
	}
	if len(files) != 1 {
		t.Errorf("expected exactly 1 file (not vendor), got %d", len(files))
	}
}

func TestDiscoverUntested_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	files, err := p.DiscoverUntested(dir, ctx)
	if err != nil {
		t.Fatalf("DiscoverUntested: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("expected 0 files in empty dir, got %d", len(files))
	}
}

// ---- DiscoverInPath ---------------------------------------------------------

func TestDiscoverInPath_Subdirectory(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "services")
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	writeFile(t, filepath.Join(dir, "main.fake"), "// root source — should NOT appear")
	writeFile(t, filepath.Join(sub, "payment.fake"), "// sub source")

	files, err := p.DiscoverInPath(sub, ctx)
	if err != nil {
		t.Fatalf("DiscoverInPath: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 file in sub, got %d: %v", len(files), files)
	}
}

// ---- DiscoverAndAnalyzeAll --------------------------------------------------

func TestDiscoverAndAnalyzeAll_ReturnsAnalyses(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	writeFile(t, filepath.Join(dir, "a.fake"), "// a")
	writeFile(t, filepath.Join(dir, "b.fake"), "// b")

	analyses, err := p.DiscoverAndAnalyzeAll(dir, ctx)
	if err != nil {
		t.Fatalf("DiscoverAndAnalyzeAll: %v", err)
	}
	if len(analyses) != 2 {
		t.Errorf("expected 2 analyses, got %d", len(analyses))
	}
}

func TestDiscoverAndAnalyzeAll_NoSources(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	analyses, err := p.DiscoverAndAnalyzeAll(dir, ctx)
	if err != nil {
		t.Fatalf("DiscoverAndAnalyzeAll: %v", err)
	}
	if len(analyses) != 0 {
		t.Errorf("expected 0 analyses in empty dir, got %d", len(analyses))
	}
}

// ---- AnalyzeFile ------------------------------------------------------------

func TestAnalyzeFile_DelegatesToDriver(t *testing.T) {
	dir := t.TempDir()
	d := &fakeDriver{ext: ".fake", testSuffix: "_test.fake"}
	p := analysis.New(d)
	ctx := &domain.ProjectContext{Root: dir}

	path := filepath.Join(dir, "payment.fake")
	writeFile(t, path, "// source")

	a, err := p.AnalyzeFile(path, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}
	if a.SourcePath != path {
		t.Errorf("SourcePath: got %q, want %q", a.SourcePath, path)
	}
}
