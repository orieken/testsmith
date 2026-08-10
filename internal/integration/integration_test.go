// Package integration runs end-to-end tests that exercise the full
// analysis → generation → executor pipeline for each language driver
// against real testdata fixtures.
package integration_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/analysis"
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/csharp"
	"github.com/orieken/assay/internal/drivers/golang"
	"github.com/orieken/assay/internal/drivers/java"
	"github.com/orieken/assay/internal/drivers/python"
	"github.com/orieken/assay/internal/drivers/typescript"
	"github.com/orieken/assay/internal/generation"
)

// testdataRoot returns the absolute path to the v2/testdata directory.
func testdataRoot() string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", "testdata")
}

// generateToTemp runs the full pipeline on srcFile and returns the generated content.
// All output is directed to a temp directory so testdata is never modified.
func generateToTemp(
	t *testing.T,
	driver domain.LanguageDriver,
	srcFile string,
	projectDir string,
) string {
	t.Helper()

	ctx, err := driver.DetectProject(projectDir)
	if err != nil {
		t.Fatalf("DetectProject(%s): %v", projectDir, err)
	}

	analysisPipeline := analysis.New(driver)
	a, err := analysisPipeline.AnalyzeFile(srcFile, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile(%s): %v", srcFile, err)
	}

	genPipeline := generation.NewPipeline(driver, nil)
	plan, err := genPipeline.Plan(context.Background(), a, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("Plan(%s): %v", srcFile, err)
	}

	// Redirect all output paths into a temp dir.
	tmpDir := t.TempDir()
	for i, f := range plan.Files {
		plan.Files[i].AbsPath = filepath.Join(tmpDir, filepath.Base(f.AbsPath))
		plan.Files[i].Action = domain.ActionCreate
	}

	ex := &generation.Executor{}
	results, err := ex.Execute(plan)
	if err != nil {
		t.Fatalf("Execute(%s): %v", srcFile, err)
	}

	for _, r := range results {
		if r.Role == domain.RoleTestFile && r.Action == domain.ActionCreate {
			content, err := os.ReadFile(r.AbsPath)
			if err != nil {
				t.Fatalf("ReadFile(%s): %v", r.AbsPath, err)
			}
			return string(content)
		}
	}
	t.Fatal("no test file was created")
	return ""
}

// ---- Python -----------------------------------------------------------------

func TestIntegration_Python_PaymentService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "python")
	src := filepath.Join(root, "src", "services", "payment.py")
	projectDir := filepath.Join(root, "src", "services")

	content := generateToTemp(t, python.New(), src, projectDir)

	if !strings.Contains(content, "def test_") {
		t.Error("expected pytest-style test functions (def test_...)")
	}
	if !strings.Contains(content, "PaymentProcessor") {
		t.Error("expected PaymentProcessor reference in generated test")
	}
}

func TestIntegration_Python_UserService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "python")
	src := filepath.Join(root, "src", "services", "user_service.py")
	projectDir := filepath.Join(root, "src", "services")

	content := generateToTemp(t, python.New(), src, projectDir)

	if !strings.Contains(content, "def test_") {
		t.Error("expected pytest-style test functions")
	}
}

// ---- TypeScript -------------------------------------------------------------

func TestIntegration_TypeScript_PaymentService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "typescript")
	src := filepath.Join(root, "src", "services", "payment.ts")

	content := generateToTemp(t, typescript.New(), src, root)

	if !strings.Contains(content, "describe") && !strings.Contains(content, "it(") {
		t.Error("expected Jest describe/it blocks in generated test")
	}
}

func TestIntegration_TypeScript_UserService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "typescript")
	src := filepath.Join(root, "src", "services", "user_service.ts")

	content := generateToTemp(t, typescript.New(), src, root)

	if !strings.Contains(content, "describe") && !strings.Contains(content, "it(") {
		t.Error("expected Jest describe/it blocks in generated test")
	}
}

// ---- Go ---------------------------------------------------------------------

func TestIntegration_Go_PaymentProcessor(t *testing.T) {
	root := filepath.Join(testdataRoot(), "golang")
	src := filepath.Join(root, "internal", "services", "payment.go")

	content := generateToTemp(t, golang.New(), src, root)

	if !strings.Contains(content, "func Test") {
		t.Error("expected Go-style TestFoo functions")
	}
	if !strings.Contains(content, "testing.T") {
		t.Error("expected *testing.T parameter")
	}
}

// ---- Java -------------------------------------------------------------------

func TestIntegration_Java_PaymentService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "java")
	src := filepath.Join(root, "src", "main", "java", "com", "example", "services", "PaymentService.java")
	projectDir := filepath.Join(root, "src", "main", "java", "com", "example", "services")

	content := generateToTemp(t, java.New(), src, projectDir)

	if !strings.Contains(content, "@Test") {
		t.Error("expected @Test annotations in generated test")
	}
	if !strings.Contains(content, "PaymentService") {
		t.Error("expected PaymentService reference")
	}
}

func TestIntegration_Java_CurrencyUtils(t *testing.T) {
	root := filepath.Join(testdataRoot(), "java")
	src := filepath.Join(root, "src", "main", "java", "com", "example", "services", "CurrencyUtils.java")
	projectDir := filepath.Join(root, "src", "main", "java", "com", "example", "services")

	content := generateToTemp(t, java.New(), src, projectDir)

	if !strings.Contains(content, "@Test") {
		t.Error("expected @Test annotations")
	}
}

// ---- C# ---------------------------------------------------------------------

func TestIntegration_CSharp_PaymentService(t *testing.T) {
	root := filepath.Join(testdataRoot(), "csharp")
	src := filepath.Join(root, "src", "Services", "PaymentService.cs")
	projectDir := filepath.Join(root, "src", "Services")

	content := generateToTemp(t, csharp.New(), src, projectDir)

	if !strings.Contains(content, "[Fact]") {
		t.Error("expected [Fact] attribute in generated xUnit test")
	}
	if !strings.Contains(content, "PaymentService") {
		t.Error("expected PaymentService reference")
	}
}

func TestIntegration_CSharp_CurrencyUtils(t *testing.T) {
	root := filepath.Join(testdataRoot(), "csharp")
	src := filepath.Join(root, "src", "Services", "CurrencyUtils.cs")
	projectDir := filepath.Join(root, "src", "Services")

	content := generateToTemp(t, csharp.New(), src, projectDir)

	if !strings.Contains(content, "[Fact]") {
		t.Error("expected [Fact] attribute")
	}
}

// ---- Cross-cutting: dry-run produces no files ------------------------------

func TestIntegration_DryRun_WritesNoFiles(t *testing.T) {
	root := filepath.Join(testdataRoot(), "golang")
	src := filepath.Join(root, "internal", "services", "payment.go")

	driver := golang.New()
	ctx, err := driver.DetectProject(root)
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysisPipeline := analysis.New(driver)
	a, err := analysisPipeline.AnalyzeFile(src, ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	genPipeline := generation.NewPipeline(driver, nil)
	plan, err := genPipeline.Plan(context.Background(), a, domain.GenerateOpts{DryRun: true})
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	if !plan.DryRun {
		t.Fatal("expected plan.DryRun to be true")
	}

	tmpDir := t.TempDir()
	for i, f := range plan.Files {
		plan.Files[i].AbsPath = filepath.Join(tmpDir, filepath.Base(f.AbsPath))
		plan.Files[i].Action = domain.ActionCreate
	}

	ex := &generation.Executor{}
	if _, err := ex.Execute(plan); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	entries, _ := os.ReadDir(tmpDir)
	if len(entries) != 0 {
		t.Errorf("dry-run wrote %d file(s), expected 0", len(entries))
	}
}

// ---- Discover untested across multiple files --------------------------------

func TestIntegration_DiscoverUntested_Python(t *testing.T) {
	root := filepath.Join(testdataRoot(), "python")
	driver := python.New()
	ctx, err := driver.DetectProject(filepath.Join(root, "src", "services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	p := analysis.New(driver)
	files, err := p.DiscoverUntested(root, ctx)
	if err != nil {
		t.Fatalf("DiscoverUntested: %v", err)
	}
	// Both payment.py and user_service.py have no test files in testdata.
	if len(files) < 2 {
		t.Errorf("expected at least 2 untested python files, got %d: %v", len(files), files)
	}
}

func TestIntegration_DiscoverAndAnalyzeAll_Java(t *testing.T) {
	root := filepath.Join(testdataRoot(), "java")
	driver := java.New()
	ctx, err := driver.DetectProject(filepath.Join(root, "src", "main", "java", "com", "example", "services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	p := analysis.New(driver)
	analyses, err := p.DiscoverAndAnalyzeAll(root, ctx)
	if err != nil {
		t.Fatalf("DiscoverAndAnalyzeAll: %v", err)
	}
	if len(analyses) < 2 {
		t.Errorf("expected at least 2 java analyses, got %d", len(analyses))
	}
}
