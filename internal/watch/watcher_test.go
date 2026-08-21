package watch

import (
	"testing"
	"time"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/generation"
)

// ---- helpers ----------------------------------------------------------------

func newTestWatcher(driver domain.LanguageDriver) *Watcher {
	ctx := &domain.ProjectContext{Root: "/proj", Language: driver.Language()}
	gen := generation.NewPipeline(driver, nil)
	return New(driver, ctx, gen, 200, false)
}

// fakeDriver implements domain.LanguageDriver for watch tests.
type fakeDriver struct {
	lang       string
	extensions []string
	suffix     string
	prefix     string
}

func (f *fakeDriver) Language() string                                      { return f.lang }
func (f *fakeDriver) FileExtensions() []string                              { return f.extensions }
func (f *fakeDriver) BodyGenerationPrompt() string                          { return "" }
func (f *fakeDriver) LLMContext(_ *domain.ProjectContext) map[string]string { return nil }
func (f *fakeDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		TestFileSuffix: f.suffix,
		TestFilePrefix: f.prefix,
	}
}
func (f *fakeDriver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return &domain.ProjectContext{Root: dir, Language: f.lang}, nil
}
func (f *fakeDriver) AnalyzeFile(p string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return &domain.SourceAnalysis{
		SourcePath: p,
		Project:    ctx,
	}, nil
}
func (f *fakeDriver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepStdlib
}
func (f *fakeDriver) DeriveTestPath(src string, ctx *domain.ProjectContext) (string, error) {
	return src + "_test", nil
}
func (f *fakeDriver) DeriveModulePath(src string, ctx *domain.ProjectContext) (string, error) {
	return src, nil
}
func (f *fakeDriver) GenerateTestFile(a *domain.SourceAnalysis, o domain.GenerateOpts) (*domain.GeneratedFile, error) {
	testPath := a.SourcePath + "_test"
	return &domain.GeneratedFile{AbsPath: testPath, Content: "// test", Role: domain.RoleTestFile}, nil
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

// ---- isSourceFile -----------------------------------------------------------

func TestIsSourceFile_MatchesExtension(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	w := newTestWatcher(d)

	if !w.isSourceFile("/proj/services/payment.go") {
		t.Error("expected payment.go to be a source file")
	}
}

func TestIsSourceFile_ExcludesTestSuffix(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	w := newTestWatcher(d)

	if w.isSourceFile("/proj/services/payment_test.go") {
		t.Error("expected payment_test.go to be excluded")
	}
}

func TestIsSourceFile_ExcludesTestPrefix(t *testing.T) {
	d := &fakeDriver{lang: "python", extensions: []string{".py"}, prefix: "test_"}
	w := newTestWatcher(d)

	if w.isSourceFile("/proj/tests/test_payment.py") {
		t.Error("expected test_payment.py to be excluded")
	}
}

func TestIsSourceFile_WrongExtension(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}}
	w := newTestWatcher(d)

	if w.isSourceFile("/proj/main.py") {
		t.Error("expected .py file to be rejected by go driver")
	}
}

func TestIsSourceFile_MultipleExtensions(t *testing.T) {
	d := &fakeDriver{lang: "typescript", extensions: []string{".ts", ".tsx"}, suffix: ".test.ts"}
	w := newTestWatcher(d)

	if !w.isSourceFile("/proj/src/app.tsx") {
		t.Error("expected .tsx to be a source file")
	}
	if !w.isSourceFile("/proj/src/service.ts") {
		t.Error("expected .ts to be a source file")
	}
}

// ---- pending map / flush logic ----------------------------------------------

func TestFlush_ExpiredEntriesProcessed(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	w := newTestWatcher(d)
	// Use a very short debounce so we can test expiry immediately.
	w.debounce = 10 * time.Millisecond

	// Inject a pending entry with a timestamp old enough to be flushed.
	w.mu.Lock()
	w.pending["/proj/payment.go"] = time.Now().Add(-50 * time.Millisecond)
	w.mu.Unlock()

	if pendingLen(w) != 1 {
		t.Fatalf("expected 1 pending entry before flush")
	}

	// flush should drain the expired entry (process will fail since no real
	// analysis is possible, but the pending map must be cleared).
	w.flush(time.Now())

	if pendingLen(w) != 0 {
		t.Errorf("expected 0 pending entries after flush, got %d", pendingLen(w))
	}
}

func TestFlush_FreshEntryNotFlushed(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	w := newTestWatcher(d)
	w.debounce = 500 * time.Millisecond

	// Entry is brand-new — should survive the flush.
	w.mu.Lock()
	w.pending["/proj/payment.go"] = time.Now()
	w.mu.Unlock()

	w.flush(time.Now())

	if pendingLen(w) != 1 {
		t.Errorf("expected fresh entry to remain after flush")
	}
}

func pendingLen(w *Watcher) int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.pending)
}

// ---- New / constructor ------------------------------------------------------

func TestNew_ExcludeDirsPropagated(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}}
	ctx := &domain.ProjectContext{
		Root:        "/proj",
		Language:    "go",
		ExcludeDirs: []string{"vendor", "node_modules"},
	}
	gen := generation.NewPipeline(d, nil)
	w := New(d, ctx, gen, 300, false)

	if !w.excludeDirs["vendor"] {
		t.Error("expected vendor to be in excludeDirs")
	}
	if !w.excludeDirs["node_modules"] {
		t.Error("expected node_modules to be in excludeDirs")
	}
}

func TestNew_DebounceSet(t *testing.T) {
	d := &fakeDriver{lang: "go", extensions: []string{".go"}}
	ctx := &domain.ProjectContext{Root: "/proj", Language: "go"}
	gen := generation.NewPipeline(d, nil)
	w := New(d, ctx, gen, 750, false)

	if w.debounce != 750*time.Millisecond {
		t.Errorf("debounce: got %v, want 750ms", w.debounce)
	}
}
