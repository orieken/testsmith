package registry_test

import (
	"errors"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/registry"
)

// ---- fakes ------------------------------------------------------------------

type fakeDriver struct {
	lang       string
	extensions []string
	detectErr  error
}

func (f *fakeDriver) Language() string                                      { return f.lang }
func (f *fakeDriver) FileExtensions() []string                              { return f.extensions }
func (f *fakeDriver) BodyGenerationPrompt() string                          { return "" }
func (f *fakeDriver) LLMContext(_ *domain.ProjectContext) map[string]string { return nil }
func (f *fakeDriver) GetTestFrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{}
}
func (f *fakeDriver) DetectProject(dir string) (*domain.ProjectContext, error) {
	if f.detectErr != nil {
		return nil, f.detectErr
	}
	return &domain.ProjectContext{Root: dir, Language: f.lang}, nil
}
func (f *fakeDriver) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, nil
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

// ---- tests ------------------------------------------------------------------

func TestForLanguage_Found(t *testing.T) {
	r := registry.New()
	r.Register(&fakeDriver{lang: "python", extensions: []string{".py"}})
	r.Register(&fakeDriver{lang: "go", extensions: []string{".go"}})

	d, err := r.ForLanguage("python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Language() != "python" {
		t.Errorf("got language %q, want python", d.Language())
	}
}

func TestForLanguage_CaseInsensitive(t *testing.T) {
	r := registry.New()
	r.Register(&fakeDriver{lang: "go", extensions: []string{".go"}})

	d, err := r.ForLanguage("GO")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Language() != "go" {
		t.Errorf("got %q, want go", d.Language())
	}
}

func TestForLanguage_Unknown(t *testing.T) {
	r := registry.New()
	_, err := r.ForLanguage("rust")
	if !errors.Is(err, domain.ErrNoDriverForLanguage) {
		t.Errorf("got %v, want ErrNoDriverForLanguage", err)
	}
}

func TestForFile_Found(t *testing.T) {
	r := registry.New()
	r.Register(&fakeDriver{lang: "typescript", extensions: []string{".ts", ".tsx"}})

	d, err := r.ForFile("src/payment.ts")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Language() != "typescript" {
		t.Errorf("got %q, want typescript", d.Language())
	}
}

func TestForFile_MultiExtension(t *testing.T) {
	r := registry.New()
	r.Register(&fakeDriver{lang: "typescript", extensions: []string{".ts", ".tsx"}})

	d, err := r.ForFile("app/component.tsx")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Language() != "typescript" {
		t.Errorf("got %q, want typescript", d.Language())
	}
}

func TestForFile_Unknown(t *testing.T) {
	r := registry.New()
	_, err := r.ForFile("main.rb")
	if !errors.Is(err, domain.ErrNoDriverForFile) {
		t.Errorf("got %v, want ErrNoDriverForFile", err)
	}
}

func TestDetect_FirstSuccess(t *testing.T) {
	errDriver := &fakeDriver{lang: "java", extensions: []string{".java"}, detectErr: errors.New("no pom.xml")}
	okDriver := &fakeDriver{lang: "go", extensions: []string{".go"}}

	r := registry.New()
	r.Register(errDriver)
	r.Register(okDriver)

	d, ctx, err := r.Detect("/some/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.Language() != "go" {
		t.Errorf("got %q, want go", d.Language())
	}
	if ctx.Language != "go" {
		t.Errorf("ctx.Language got %q, want go", ctx.Language)
	}
}

func TestDetect_NoneMatch(t *testing.T) {
	r := registry.New()
	r.Register(&fakeDriver{lang: "java", extensions: []string{".java"}, detectErr: errors.New("no pom.xml")})
	r.Register(&fakeDriver{lang: "go", extensions: []string{".go"}, detectErr: errors.New("no go.mod")})

	_, _, err := r.Detect("/empty/dir")
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("got %v, want ErrProjectNotFound", err)
	}
}

func TestDetect_EmptyRegistry(t *testing.T) {
	r := registry.New()
	_, _, err := r.Detect("/any/path")
	if !errors.Is(err, domain.ErrProjectNotFound) {
		t.Errorf("got %v, want ErrProjectNotFound", err)
	}
}
