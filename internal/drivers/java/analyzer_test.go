package java_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/java"
)

func testdataPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "java")
	return filepath.Join(root, rel)
}

func TestDetectProject(t *testing.T) {
	d := java.New()
	ctx, err := d.DetectProject(testdataPath("src/main/java/com/example/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	if ctx.Language != "java" {
		t.Errorf("language: got %q, want %q", ctx.Language, "java")
	}
	basePkg, _ := ctx.Metadata["base_package"].(string)
	if basePkg != "com.example" {
		t.Errorf("base_package: got %q, want %q", basePkg, "com.example")
	}
	buildSystem, _ := ctx.Metadata["build_system"].(string)
	if buildSystem != "maven" {
		t.Errorf("build_system: got %q, want %q", buildSystem, "maven")
	}
}

func TestAnalyzeFile_PaymentService(t *testing.T) {
	d := java.New()
	ctx, err := d.DetectProject(testdataPath("src/main/java/com/example/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/main/java/com/example/services/PaymentService.java"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	// ---- imports ----
	stdlibMods := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibMods[imp.Module] = true
	}
	if !stdlibMods["java.util.List"] {
		t.Error("expected java.util.List as stdlib")
	}
	if !stdlibMods["java.util.Optional"] {
		t.Error("expected java.util.Optional as stdlib")
	}

	externalMods := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalMods[imp.Module] = true
	}
	if !externalMods["org.springframework.stereotype.Service"] {
		t.Error("expected Spring as external")
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
	if !methodNames["charge"] {
		t.Error("expected charge method on PaymentService")
	}
	if !methodNames["refund"] {
		t.Error("expected refund method on PaymentService")
	}
	if methodNames["internalHelper"] {
		t.Error("private method internalHelper should not be in public API")
	}
}

func TestClassifyDependency(t *testing.T) {
	d := java.New()
	ctx := &domain.ProjectContext{
		Root:     "/proj",
		Language: "java",
		Metadata: map[string]any{"base_package": "com.example"},
	}

	cases := []struct {
		path string
		want domain.DependencyCategory
	}{
		{"java.util.List", domain.DepStdlib},
		{"java.io.IOException", domain.DepStdlib},
		{"javax.servlet.http.HttpServlet", domain.DepStdlib},
		{"jakarta.persistence.Entity", domain.DepStdlib},
		{"org.springframework.stereotype.Service", domain.DepExternal},
		{"org.junit.jupiter.api.Test", domain.DepExternal},
		{"com.example.domain.Payment", domain.DepInternal},
		{"com.example.services.PaymentService", domain.DepInternal},
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
	d := java.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "java"}

	cases := []struct {
		source string
		want   string
	}{
		{
			"/proj/src/main/java/com/example/PaymentService.java",
			"/proj/src/test/java/com/example/PaymentServiceTest.java",
		},
		{
			"/proj/src/main/java/com/example/utils/CurrencyUtils.java",
			"/proj/src/test/java/com/example/utils/CurrencyUtilsTest.java",
		},
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
