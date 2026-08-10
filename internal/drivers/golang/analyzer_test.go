package golang_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/golang"
)

func testdataPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "golang")
	return filepath.Join(root, rel)
}

func TestDetectProject(t *testing.T) {
	d := golang.New()
	ctx, err := d.DetectProject(testdataPath("internal/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	if ctx.Language != "go" {
		t.Errorf("language: got %q, want %q", ctx.Language, "go")
	}
	if ctx.Root == "" {
		t.Error("expected non-empty Root")
	}
	mod, _ := ctx.Metadata["module"].(string)
	if mod != "example.com/testapp" {
		t.Errorf("module: got %q, want %q", mod, "example.com/testapp")
	}
}

func TestAnalyzeFile_Payment(t *testing.T) {
	d := golang.New()
	ctx, err := d.DetectProject(testdataPath("internal/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("internal/services/payment.go"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	// ---- imports ----
	stdlibMods := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibMods[imp.Module] = true
	}
	if !stdlibMods["fmt"] {
		t.Error("expected fmt as stdlib")
	}
	if !stdlibMods["net/http"] {
		t.Error("expected net/http as stdlib")
	}

	externalMods := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalMods[imp.Module] = true
	}
	if !externalMods["github.com/stripe/stripe-go/v76"] {
		t.Error("expected stripe-go as external")
	}

	// ---- public API ----
	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	if _, ok := membersByName["PaymentProcessor"]; !ok {
		t.Error("expected PaymentProcessor struct in public API")
	}
	if _, ok := membersByName["NewPaymentProcessor"]; !ok {
		t.Error("expected NewPaymentProcessor function in public API")
	}
	if _, ok := membersByName["CalculateTotal"]; !ok {
		t.Error("expected CalculateTotal function in public API")
	}
	if _, ok := membersByName["FormatCurrency"]; !ok {
		t.Error("expected FormatCurrency function in public API")
	}

	proc := membersByName["PaymentProcessor"]
	if proc.Kind != domain.KindClass {
		t.Errorf("PaymentProcessor kind: got %q, want class", proc.Kind)
	}
	methodNames := make(map[string]bool)
	for _, m := range proc.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["Charge"] {
		t.Error("expected Charge method on PaymentProcessor")
	}
	if !methodNames["Refund"] {
		t.Error("expected Refund method on PaymentProcessor")
	}
}

func TestClassifyDependency(t *testing.T) {
	d := golang.New()
	ctx := &domain.ProjectContext{
		Root:     "/proj",
		Language: "go",
		Metadata: map[string]any{"module": "github.com/example/myapp"},
	}

	cases := []struct {
		path string
		want domain.DependencyCategory
	}{
		{"fmt", domain.DepStdlib},
		{"net/http", domain.DepStdlib},
		{"os/exec", domain.DepStdlib},
		{"encoding/json", domain.DepStdlib},
		{"github.com/stripe/stripe-go/v76", domain.DepExternal},
		{"golang.org/x/sync/errgroup", domain.DepExternal},
		{"github.com/example/myapp/internal/services", domain.DepInternal},
		{"github.com/example/myapp", domain.DepInternal},
	}

	for _, tc := range cases {
		imp := domain.ImportInfo{Module: tc.path}
		got := d.ClassifyDependency(imp, ctx)
		if got != tc.want {
			t.Errorf("ClassifyDependency(%q): got %v, want %v", tc.path, got, tc.want)
		}
	}
}

func TestDeriveTestPath(t *testing.T) {
	d := golang.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "go"}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/internal/services/payment.go", "/proj/internal/services/payment_test.go"},
		{"/proj/cmd/main.go", "/proj/cmd/main_test.go"},
		{"/proj/pkg/auth/auth.go", "/proj/pkg/auth/auth_test.go"},
	}

	for _, tc := range cases {
		got, err := d.DeriveTestPath(tc.source, ctx)
		if err != nil {
			t.Errorf("DeriveTestPath(%q): %v", tc.source, err)
			continue
		}
		if filepath.ToSlash(got) != filepath.ToSlash(tc.want) {
			t.Errorf("DeriveTestPath(%q): got %q, want %q", tc.source, got, tc.want)
		}
	}
}

func TestDeriveModulePath(t *testing.T) {
	d := golang.New()
	ctx := &domain.ProjectContext{
		Root:     "/proj",
		Language: "go",
		Metadata: map[string]any{"module": "github.com/example/myapp"},
	}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/internal/services/payment.go", "github.com/example/myapp/internal/services"},
		{"/proj/cmd/main.go", "github.com/example/myapp/cmd"},
	}

	for _, tc := range cases {
		got, err := d.DeriveModulePath(tc.source, ctx)
		if err != nil {
			t.Errorf("DeriveModulePath(%q): %v", tc.source, err)
			continue
		}
		if got != tc.want {
			t.Errorf("DeriveModulePath(%q): got %q, want %q", tc.source, got, tc.want)
		}
	}
}
