package generation_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// ---- fakes ------------------------------------------------------------------

type fakeDriver struct {
	lang        string
	testSuffix  string
	generatedContent string
}

func (f *fakeDriver) Language() string            { return f.lang }
func (f *fakeDriver) FileExtensions() []string    { return []string{".go"} }
func (f *fakeDriver) BodyGenerationPrompt() string { return "" }
func (f *fakeDriver) LLMContext() map[string]string { return nil }
func (f *fakeDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{TestFileSuffix: f.testSuffix}
}
func (f *fakeDriver) DetectProject(dir string) (*domain.ProjectContext, error) {
	return &domain.ProjectContext{Root: dir, Language: f.lang}, nil
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
	testPath, _ := f.DeriveTestPath(a.SourcePath, a.Project)
	content := f.generatedContent
	if content == "" {
		content = "// generated test"
	}
	return &domain.GeneratedFile{AbsPath: testPath, Content: content, Role: domain.RoleTestFile}, nil
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
func (f *fakeDriver) ListMigrators() []domain.Migrator { return nil }
func (f *fakeDriver) ValidateFile(_, _, _ string) []domain.ValidationIssue { return nil }

type fakeBodyGen struct {
	bodies map[string][]string
}

func (g *fakeBodyGen) GenerateBodies(_ context.Context, req domain.BodyGenRequest) ([]domain.BodyGenResult, error) {
	lines, ok := g.bodies[req.MemberName]
	if !ok {
		return nil, nil
	}
	return []domain.BodyGenResult{{MemberName: req.MemberName, CodeLines: lines}}, nil
}

// ---- helpers ----------------------------------------------------------------

func newAnalysis(t *testing.T, dir string) *domain.SourceAnalysis {
	t.Helper()
	return &domain.SourceAnalysis{
		SourcePath: filepath.Join(dir, "payment.go"),
		Project:    &domain.ProjectContext{Root: dir, Language: "go"},
		PublicAPI: []domain.PublicMember{
			{Name: "Charge", Kind: domain.KindFunction},
		},
	}
}

// ---- Plan -------------------------------------------------------------------

func TestPlan_CreatesTestFile(t *testing.T) {
	dir := t.TempDir()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil)
	analysis := newAnalysis(t, dir)

	plan, err := pipeline.Plan(context.Background(), analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	var testFile *domain.GeneratedFile
	for i := range plan.Files {
		if plan.Files[i].Role == domain.RoleTestFile {
			testFile = &plan.Files[i]
		}
	}
	if testFile == nil {
		t.Fatal("expected a test file in the plan")
	}
	if testFile.Action != domain.ActionCreate {
		t.Errorf("action: got %q, want created", testFile.Action)
	}
}

func TestPlan_SkipsExistingTestFile(t *testing.T) {
	dir := t.TempDir()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil)
	analysis := newAnalysis(t, dir)

	// Pre-create the test file so action should be Skip.
	testPath := filepath.Join(dir, "payment_test.go")
	if err := os.WriteFile(testPath, []byte("// existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	plan, err := pipeline.Plan(context.Background(), analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	for _, f := range plan.Files {
		if f.Role == domain.RoleTestFile {
			if f.Action != domain.ActionSkip {
				t.Errorf("expected ActionSkip for existing test, got %q", f.Action)
			}
			return
		}
	}
	t.Error("no test file entry in plan")
}

func TestPlan_OverwriteUpdatesExisting(t *testing.T) {
	dir := t.TempDir()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil)
	analysis := newAnalysis(t, dir)

	testPath := filepath.Join(dir, "payment_test.go")
	_ = os.WriteFile(testPath, []byte("// old"), 0o644)

	plan, err := pipeline.Plan(context.Background(), analysis, domain.GenerateOpts{OverwriteExisting: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}

	for _, f := range plan.Files {
		if f.Role == domain.RoleTestFile {
			if f.Action != domain.ActionUpdate {
				t.Errorf("expected ActionUpdate with overwrite, got %q", f.Action)
			}
			return
		}
	}
	t.Error("no test file entry in plan")
}

func TestPlan_DryRunFlagPropagated(t *testing.T) {
	dir := t.TempDir()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	pipeline := generation.NewPipeline(driver, nil)
	analysis := newAnalysis(t, dir)

	plan, err := pipeline.Plan(context.Background(), analysis, domain.GenerateOpts{DryRun: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !plan.DryRun {
		t.Error("expected plan.DryRun to be true")
	}
}

func TestPlan_WithLLMBodies(t *testing.T) {
	dir := t.TempDir()
	driver := &fakeDriver{lang: "go", testSuffix: "_test.go"}
	bodyGen := &fakeBodyGen{
		bodies: map[string][]string{
			"Charge": {"// LLM line 1", "// LLM line 2"},
		},
	}
	pipeline := generation.NewPipeline(driver, bodyGen)
	analysis := newAnalysis(t, dir)

	// Should not error — LLM is optional and non-fatal.
	_, err := pipeline.Plan(context.Background(), analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("Plan with LLM: %v", err)
	}
}

// ---- Executor ---------------------------------------------------------------

func TestExecutor_WritesFile(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "payment_test.go")

	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{AbsPath: dest, Content: "// generated", Action: domain.ActionCreate, Role: domain.RoleTestFile},
		},
	}

	ex := &generation.Executor{}
	results, err := ex.Execute(plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Action != domain.ActionCreate {
		t.Errorf("action: got %q, want created", results[0].Action)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(content) != "// generated" {
		t.Errorf("content: got %q, want %q", string(content), "// generated")
	}
}

func TestExecutor_DryRunDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "payment_test.go")

	plan := &domain.GenerationPlan{
		DryRun: true,
		Files: []domain.GeneratedFile{
			{AbsPath: dest, Content: "// generated", Action: domain.ActionCreate, Role: domain.RoleTestFile},
		},
	}

	ex := &generation.Executor{}
	results, err := ex.Execute(plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("dry-run should not write files to disk")
	}
}

func TestExecutor_SkipDoesNotWrite(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "payment_test.go")

	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{AbsPath: dest, Content: "// new", Action: domain.ActionSkip, Role: domain.RoleTestFile},
		},
	}

	ex := &generation.Executor{}
	_, err := ex.Execute(plan)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, err := os.Stat(dest); !os.IsNotExist(err) {
		t.Error("skip action should not write files")
	}
}

func TestExecutor_CreatesMissingDirectories(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "sub", "pkg", "payment_test.go")

	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{AbsPath: dest, Content: "// deep", Action: domain.ActionCreate, Role: domain.RoleTestFile},
		},
	}

	ex := &generation.Executor{}
	if _, err := ex.Execute(plan); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if _, err := os.Stat(dest); err != nil {
		t.Errorf("expected file at deep path: %v", err)
	}
}
