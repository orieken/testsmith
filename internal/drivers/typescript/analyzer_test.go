package typescript_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/drivers/typescript"
)

func testdataPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "typescript")
	return filepath.Join(root, rel)
}

func TestDetectProject(t *testing.T) {
	d := typescript.New()
	ctx, err := d.DetectProject(testdataPath("src/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	if ctx.Language != "typescript" {
		t.Errorf("language: got %q, want %q", ctx.Language, "typescript")
	}
	if ctx.Root == "" {
		t.Error("expected non-empty Root")
	}
	// package.json lists stripe and axios as dependencies.
	if _, ok := ctx.PackageMap["stripe"]; !ok {
		t.Error("expected stripe in PackageMap")
	}
	if _, ok := ctx.PackageMap["axios"]; !ok {
		t.Error("expected axios in PackageMap")
	}
}

func TestAnalyzeFile_Payment(t *testing.T) {
	d := typescript.New()
	ctx, err := d.DetectProject(testdataPath("src/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/services/payment.ts"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	// ---- imports ----
	externalModules := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalModules[imp.Module] = true
	}
	if !externalModules["stripe"] {
		t.Error("expected stripe to be classified as external")
	}
	if !externalModules["axios"] {
		t.Error("expected axios to be classified as external")
	}

	stdlibModules := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibModules[imp.Module] = true
	}
	if !stdlibModules["events"] {
		t.Error("expected events to be classified as stdlib")
	}
	if !stdlibModules["path"] {
		t.Error("expected path to be classified as stdlib")
	}

	// ---- public API ----
	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	if _, ok := membersByName["PaymentService"]; !ok {
		t.Error("expected PaymentService class in public API")
	}
	if _, ok := membersByName["fetchExchangeRate"]; !ok {
		t.Error("expected fetchExchangeRate function in public API")
	}
	if _, ok := membersByName["formatAmount"]; !ok {
		t.Error("expected formatAmount function in public API")
	}
	// Interfaces and types are detected as KindClass.
	if _, ok := membersByName["PaymentConfig"]; !ok {
		t.Error("expected PaymentConfig interface in public API")
	}
	if _, ok := membersByName["PaymentStatus"]; !ok {
		t.Error("expected PaymentStatus type alias in public API")
	}

	cls := membersByName["PaymentService"]
	if cls.Kind != domain.KindClass {
		t.Errorf("PaymentService kind: got %q, want class", cls.Kind)
	}
	methodNames := make(map[string]bool)
	for _, m := range cls.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["charge"] {
		t.Error("expected charge method on PaymentService")
	}
	if !methodNames["refund"] {
		t.Error("expected refund method on PaymentService")
	}
}

func TestAnalyzeFile_UserService(t *testing.T) {
	d := typescript.New()
	ctx, err := d.DetectProject(testdataPath("src/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/services/user_service.ts"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	externalModules := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalModules[imp.Module] = true
	}
	if !externalModules["axios"] {
		t.Error("expected axios to be external")
	}

	stdlibModules := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibModules[imp.Module] = true
	}
	if !stdlibModules["crypto"] {
		t.Error("expected crypto to be classified as stdlib")
	}
	if !stdlibModules["fs"] {
		t.Error("expected fs to be classified as stdlib")
	}

	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}
	if _, ok := membersByName["UserService"]; !ok {
		t.Error("expected UserService class in public API")
	}
	if _, ok := membersByName["hashPassword"]; !ok {
		t.Error("expected hashPassword function in public API")
	}
	if _, ok := membersByName["loadConfig"]; !ok {
		t.Error("expected loadConfig function in public API")
	}
}

func TestClassifyDependency(t *testing.T) {
	d := typescript.New()
	ctx := &domain.ProjectContext{
		Root:       "/tmp/proj",
		Language:   "typescript",
		PackageMap: map[string]string{"stripe": "stripe", "axios": "axios"},
	}

	cases := []struct {
		module string
		want   domain.DependencyCategory
	}{
		{"fs", domain.DepStdlib},
		{"path", domain.DepStdlib},
		{"crypto", domain.DepStdlib},
		{"events", domain.DepStdlib},
		{"node:fs", domain.DepStdlib},
		{"node:path/promises", domain.DepStdlib},
		{"stripe", domain.DepExternal},
		{"axios", domain.DepExternal},
		{"@aws-sdk/client-s3", domain.DepExternal},
		{"./utils", domain.DepInternal},
		{"../models/user", domain.DepInternal},
	}

	for _, tc := range cases {
		imp := domain.ImportInfo{Module: tc.module}
		got := d.ClassifyDependency(imp, ctx)
		if got != tc.want {
			t.Errorf("ClassifyDependency(%q): got %v, want %v", tc.module, got, tc.want)
		}
	}
}

func TestDeriveTestPath(t *testing.T) {
	d := typescript.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "typescript"}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/src/services/payment.ts", "/proj/src/services/payment.test.ts"},
		{"/proj/src/utils/auth.ts", "/proj/src/utils/auth.test.ts"},
		{"/proj/components/Button.tsx", "/proj/components/Button.test.tsx"},
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
