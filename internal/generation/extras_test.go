package generation_test

import (
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// ── VerifierFor ───────────────────────────────────────────────────────────────

func TestVerifierFor_KnownLanguages_ReturnNonNil(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"go", "typescript", "javascript", "python"} {
		if v := generation.VerifierFor(lang); v == nil {
			t.Errorf("VerifierFor(%q) = nil, want non-nil Verifier", lang)
		}
	}
}

func TestVerifierFor_UnsupportedLanguages_ReturnNil(t *testing.T) {
	t.Parallel()
	for _, lang := range []string{"java", "csharp", "rust", ""} {
		if v := generation.VerifierFor(lang); v != nil {
			t.Errorf("VerifierFor(%q) = %T, want nil", lang, v)
		}
	}
}

// ── WithVerifiers / NewVerifiedExecutor ───────────────────────────────────────

func TestWithVerifiers_SetsVerifierMap(t *testing.T) {
	t.Parallel()
	ex := &generation.Executor{}
	v := generation.VerifierFor("go")
	got := ex.WithVerifiers(map[string]generation.Verifier{"go": v})
	if got == nil {
		t.Fatal("WithVerifiers returned nil")
	}
}

func TestNewVerifiedExecutor_NonNilForKnownLanguage(t *testing.T) {
	t.Parallel()
	ex := generation.NewVerifiedExecutor("go")
	if ex == nil {
		t.Fatal("NewVerifiedExecutor(go) = nil")
	}
}

func TestNewVerifiedExecutor_NilVerifierForUnsupported(t *testing.T) {
	t.Parallel()
	// Java has no verifier — executor is returned but with an empty verifier map.
	ex := generation.NewVerifiedExecutor("java")
	if ex == nil {
		t.Fatal("NewVerifiedExecutor(java) = nil")
	}
}

// ── WithDepIndex / WithPromptTokenBudget / UpdateDepEntry ─────────────────────

func TestWithDepIndex_ReturnsSamePipeline(t *testing.T) {
	t.Parallel()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil)
	idx := map[string]*domain.SourceAnalysis{
		"mymod/util": {ModulePath: "mymod/util"},
	}
	got := pipeline.WithDepIndex(idx)
	if got == nil {
		t.Error("WithDepIndex returned nil")
	}
}

func TestWithPromptTokenBudget_ReturnsSamePipeline(t *testing.T) {
	t.Parallel()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil).WithPromptTokenBudget(6000)
	if pipeline == nil {
		t.Error("WithPromptTokenBudget returned nil")
	}
}

func TestUpdateDepEntry_NilDepIndex_IsNoop(t *testing.T) {
	t.Parallel()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	// No WithDepIndex → depIndex is nil; UpdateDepEntry must not panic.
	pipeline := generation.NewPipeline(driver, nil)
	pipeline.UpdateDepEntry("mymod/util", &domain.SourceAnalysis{ModulePath: "mymod/util"})
}

func TestUpdateDepEntry_EmptyModulePath_IsNoop(t *testing.T) {
	t.Parallel()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	idx := make(map[string]*domain.SourceAnalysis)
	pipeline := generation.NewPipeline(driver, nil).WithDepIndex(idx)
	pipeline.UpdateDepEntry("", &domain.SourceAnalysis{})
	if len(idx) != 0 {
		t.Error("UpdateDepEntry with empty modulePath should not modify depIndex")
	}
}

func TestUpdateDepEntry_UpdatesIndexEntry(t *testing.T) {
	t.Parallel()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	idx := make(map[string]*domain.SourceAnalysis)
	pipeline := generation.NewPipeline(driver, nil).WithDepIndex(idx)

	a := &domain.SourceAnalysis{ModulePath: "mymod/util"}
	pipeline.UpdateDepEntry("mymod/util", a)
	if idx["mymod/util"] != a {
		t.Error("UpdateDepEntry should store the analysis under the given key")
	}
}

// ── GapsForAnalyses ───────────────────────────────────────────────────────────

func TestGapsForAnalyses_EmptyInput(t *testing.T) {
	t.Parallel()
	gaps, err := generation.GapsForAnalyses(nil, &stubDriver{})
	if err != nil {
		t.Fatalf("GapsForAnalyses(nil): %v", err)
	}
	if gaps == nil {
		gaps = []domain.CoverageGap{}
	}
	if len(gaps) != 0 {
		t.Errorf("expected 0 gaps for nil input, got %d", len(gaps))
	}
}

func TestGapsForAnalyses_SingleAnalysis_ReturnsGap(t *testing.T) {
	t.Parallel()
	a := &domain.SourceAnalysis{
		SourcePath: "/tmp/nonexistent_source.py",
		ModulePath: "mymod",
	}
	gaps, err := generation.GapsForAnalyses([]*domain.SourceAnalysis{a}, &stubDriver{})
	if err != nil {
		t.Fatalf("GapsForAnalyses: %v", err)
	}
	// The test file does not exist, so we expect a CoverageNoTest gap.
	if len(gaps) != 1 {
		t.Errorf("expected 1 gap, got %d", len(gaps))
	}
}
