package csharp_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/drivers/csharp"
)

func testdataPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "csharp")
	return filepath.Join(root, rel)
}

func TestDetectProject(t *testing.T) {
	d := csharp.New()
	ctx, err := d.DetectProject(testdataPath("src/Services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	if ctx.Language != "csharp" {
		t.Errorf("language: got %q, want %q", ctx.Language, "csharp")
	}
	rootNS, _ := ctx.Metadata["root_namespace"].(string)
	if rootNS != "MyApp" {
		t.Errorf("root_namespace: got %q, want %q", rootNS, "MyApp")
	}
}

func TestAnalyzeFile_PaymentService(t *testing.T) {
	d := csharp.New()
	ctx, err := d.DetectProject(testdataPath("src/Services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/Services/PaymentService.cs"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	// ---- imports (using directives) ----
	stdlibMods := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibMods[imp.Module] = true
	}
	if !stdlibMods["System"] {
		t.Error("expected System as stdlib")
	}
	if !stdlibMods["System.Threading.Tasks"] {
		t.Error("expected System.Threading.Tasks as stdlib")
	}

	externalMods := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalMods[imp.Module] = true
	}
	if !externalMods["Stripe"] {
		t.Error("expected Stripe as external")
	}

	// ---- public API ----
	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	svc, ok := membersByName["PaymentService"]
	if !ok {
		t.Fatal("expected PaymentService class in public API")
	}
	if svc.Kind != domain.KindClass {
		t.Errorf("PaymentService kind: got %q, want class", svc.Kind)
	}

	methodNames := make(map[string]bool)
	for _, m := range svc.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["ChargeAsync"] {
		t.Error("expected ChargeAsync method on PaymentService")
	}
	if !methodNames["Refund"] {
		t.Error("expected Refund method on PaymentService")
	}
	if methodNames["InternalHelper"] {
		t.Error("private method InternalHelper should not appear")
	}
}

func TestAnalyzeFile_CurrencyUtils(t *testing.T) {
	d := csharp.New()
	ctx, err := d.DetectProject(testdataPath("src/Services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/Services/CurrencyUtils.cs"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	if _, ok := membersByName["CurrencyUtils"]; !ok {
		t.Error("expected CurrencyUtils class in public API")
	}
	utils := membersByName["CurrencyUtils"]
	methodNames := make(map[string]bool)
	for _, m := range utils.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["CalculateTotal"] {
		t.Error("expected CalculateTotal method")
	}
	if !methodNames["FormatAmount"] {
		t.Error("expected FormatAmount method")
	}
}

func TestClassifyDependency(t *testing.T) {
	d := csharp.New()
	ctx := &domain.ProjectContext{
		Root:     "/proj",
		Language: "csharp",
		Metadata: map[string]any{"root_namespace": "MyApp"},
	}

	cases := []struct {
		ns   string
		want domain.DependencyCategory
	}{
		{"System", domain.DepStdlib},
		{"System.Collections.Generic", domain.DepStdlib},
		{"System.Threading.Tasks", domain.DepStdlib},
		{"Microsoft.Extensions.Logging", domain.DepStdlib},
		{"Stripe", domain.DepExternal},
		{"Newtonsoft.Json", domain.DepExternal},
		{"MyApp.Services", domain.DepInternal},
		{"MyApp.Domain.Payment", domain.DepInternal},
	}

	for _, tc := range cases {
		imp := domain.ImportInfo{Module: tc.ns}
		got := d.ClassifyDependency(imp, ctx)
		if got != tc.want {
			t.Errorf("ClassifyDependency(%q): got %v, want %v", tc.ns, got, tc.want)
		}
	}
}

func TestDeriveTestPath(t *testing.T) {
	d := csharp.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "csharp"}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/src/Services/PaymentService.cs", "/proj/src/Services/PaymentServiceTests.cs"},
		{"/proj/src/Utils/CurrencyUtils.cs", "/proj/src/Utils/CurrencyUtilsTests.cs"},
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
