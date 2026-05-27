package generation_test

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// ---- classifyTestContent (via DetectCoverage) --------------------------------

func TestClassifyTestContent_NoTest(t *testing.T) {
	gaps := coverageGapsWith("")
	if len(gaps) != 1 || gaps[0].Status != domain.CoverageNoTest {
		t.Errorf("empty test path should yield CoverageNoTest, got %v", gaps)
	}
}

func TestClassifyTestContent_SkeletonOnly(t *testing.T) {
	content := "def test_foo(self):\n    pass\n"
	status := classifyContent(content)
	if status != domain.CoverageSkeletonOnly {
		t.Errorf("stub-only content: got %v, want CoverageSkeletonOnly", status)
	}
}

func TestClassifyTestContent_Full(t *testing.T) {
	content := "def test_foo(self):\n    assert result == 42\n"
	status := classifyContent(content)
	if status != domain.CoverageFull {
		t.Errorf("asserted content: got %v, want CoverageFull", status)
	}
}

func TestClassifyTestContent_Partial(t *testing.T) {
	content := "def test_foo(self):\n    assert result == 1\ndef test_bar(self):\n    pass\n"
	status := classifyContent(content)
	if status != domain.CoveragePartial {
		t.Errorf("mixed content: got %v, want CoveragePartial", status)
	}
}

// ---- PrioritizeGaps ----------------------------------------------------------

func TestPrioritizeGaps_OrderedByScore(t *testing.T) {
	gaps := []domain.CoverageGap{
		{SourcePath: "a.py", Status: domain.CoveragePartial, ExternalDeps: 1},
		{SourcePath: "b.py", Status: domain.CoverageNoTest, ExternalDeps: 3},
		{SourcePath: "c.py", Status: domain.CoverageSkeletonOnly, ExternalDeps: 0},
	}
	result := generation.PrioritizeGaps(gaps, nil)
	if result[0].SourcePath != "b.py" {
		t.Errorf("highest gap should be b.py (no test + most deps), got %s", result[0].SourcePath)
	}
	for i := 1; i < len(result); i++ {
		if result[i].PriorityScore > result[i-1].PriorityScore {
			t.Errorf("gaps not sorted descending at index %d", i)
		}
	}
}

// ---- GenerateReport ----------------------------------------------------------

func TestGenerateReport_AllCovered(t *testing.T) {
	report := generation.GenerateReport(nil, 5)
	if !strings.Contains(report, "5 / 5") {
		t.Errorf("report should show 5/5 covered, got:\n%s", report)
	}
	if !strings.Contains(report, "No coverage gaps") {
		t.Errorf("report should indicate no gaps")
	}
}

func TestGenerateReport_WithGaps(t *testing.T) {
	gaps := []domain.CoverageGap{
		{SourcePath: "src/auth.py", Status: domain.CoverageNoTest, PriorityScore: 0.9, SuggestedCommand: "testsmith generate src/auth.py"},
	}
	report := generation.GenerateReport(gaps, 3)
	if !strings.Contains(report, "auth.py") {
		t.Errorf("report should list auth.py")
	}
	if !strings.Contains(report, "2 / 3") {
		t.Errorf("report should show 2/3 covered")
	}
}

// ---- helpers -----------------------------------------------------------------

// classifyContent uses a spy driver to exercise the classify path via DetectCoverage.
func classifyContent(content string) domain.CoverageStatus {
	return generation.ClassifyContentForTest(content, domain.TestFrameworkConfig{})
}

// coverageGapsWith creates a single analysis pointing at a non-existent test file.
func coverageGapsWith(testFileContent string) []domain.CoverageGap {
	_ = testFileContent
	a := &domain.SourceAnalysis{
		SourcePath: "/tmp/nonexistent_source.py",
		ModulePath: "nonexistent",
	}
	driver := &stubDriver{}
	return generation.DetectCoverage([]*domain.SourceAnalysis{a}, driver)
}

// stubDriver satisfies domain.LanguageDriver minimally for coverage tests.
type stubDriver struct{}

func (s *stubDriver) Language() string         { return "python" }
func (s *stubDriver) FileExtensions() []string { return []string{".py"} }
func (s *stubDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{TestFilePrefix: "test_", FixtureSuffix: "_fixture.py"}
}
func (s *stubDriver) DeriveTestPath(src string, ctx *domain.ProjectContext) (string, error) {
	return "/tmp/does_not_exist_test.py", nil
}
func (s *stubDriver) DetectProject(dir string) (*domain.ProjectContext, error) { return nil, nil }
func (s *stubDriver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, nil
}
func (s *stubDriver) ClassifyDependency(dep domain.ImportInfo, ctx *domain.ProjectContext) domain.DependencyCategory {
	return domain.DepExternal
}
func (s *stubDriver) DeriveModulePath(src string, ctx *domain.ProjectContext) (string, error) {
	return "", nil
}
func (s *stubDriver) GenerateTestFile(a *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) GenerateFixture(dep string, a *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) GenerateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	return nil, nil
}
func (s *stubDriver) BodyGenerationPrompt() string                          { return "" }
func (s *stubDriver) LLMContext(_ *domain.ProjectContext) map[string]string { return nil }
func (s *stubDriver) ListAdapters(_ *domain.ProjectContext) ([]domain.TestAdapter, domain.TestAdapter) {
	return nil, nil
}
func (s *stubDriver) ListMigrators() []domain.Migrator                     { return nil }
func (s *stubDriver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }
