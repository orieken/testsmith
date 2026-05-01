package python_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/drivers/python"
)

func testdataPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Join(filepath.Dir(file), "..", "..", "..", "testdata", "python")
	return filepath.Join(root, rel)
}

func TestAnalyzeFile_Payment(t *testing.T) {
	d := python.New()
	ctx, err := d.DetectProject(testdataPath("src/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/services/payment.py"), ctx)
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
	if !externalModules["requests"] {
		t.Error("expected requests to be classified as external")
	}

	stdlibModules := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibModules[imp.Module] = true
	}
	if !stdlibModules["os"] {
		t.Error("expected os to be classified as stdlib")
	}
	if !stdlibModules["logging"] {
		t.Error("expected logging to be classified as stdlib")
	}

	// ---- public API ----
	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	if _, ok := membersByName["PaymentProcessor"]; !ok {
		t.Error("expected PaymentProcessor class in public API")
	}
	if _, ok := membersByName["calculate_total"]; !ok {
		t.Error("expected calculate_total function in public API")
	}
	if _, ok := membersByName["fetch_exchange_rate"]; !ok {
		t.Error("expected fetch_exchange_rate function in public API")
	}

	cls := membersByName["PaymentProcessor"]
	if cls.Kind != domain.KindClass {
		t.Errorf("PaymentProcessor kind: got %q, want %q", cls.Kind, domain.KindClass)
	}

	methodNames := make(map[string]bool)
	for _, m := range cls.Methods {
		methodNames[m.Name] = true
	}
	if !methodNames["charge"] {
		t.Error("expected charge method on PaymentProcessor")
	}
	if !methodNames["refund"] {
		t.Error("expected refund method on PaymentProcessor")
	}
}

func TestAnalyzeFile_UserService(t *testing.T) {
	d := python.New()
	ctx, err := d.DetectProject(testdataPath("src/services"))
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}

	analysis, err := d.AnalyzeFile(testdataPath("src/services/user_service.py"), ctx)
	if err != nil {
		t.Fatalf("AnalyzeFile: %v", err)
	}

	externalModules := make(map[string]bool)
	for _, imp := range analysis.Imports.External {
		externalModules[imp.Module] = true
	}
	if !externalModules["bcrypt"] {
		t.Error("expected bcrypt to be external")
	}
	if !externalModules["sqlalchemy"] {
		t.Error("expected sqlalchemy to be external")
	}

	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	if _, ok := membersByName["UserService"]; !ok {
		t.Error("expected UserService class in public API")
	}
	if _, ok := membersByName["hash_password"]; !ok {
		t.Error("expected hash_password function in public API")
	}
	// UserDTO is a dataclass — should be detected as a class.
	if _, ok := membersByName["UserDTO"]; !ok {
		t.Error("expected UserDTO dataclass in public API")
	}
}

func TestClassifyDependency(t *testing.T) {
	d := python.New()
	ctx := &domain.ProjectContext{
		Root:       "/tmp/proj",
		Language:   "python",
		PackageMap: map[string]string{"services": "/tmp/proj/src/services"},
	}

	cases := []struct {
		module string
		want   domain.DependencyCategory
	}{
		{"os", domain.DepStdlib},
		{"sys", domain.DepStdlib},
		{"logging", domain.DepStdlib},
		{"stripe", domain.DepExternal},
		{"requests", domain.DepExternal},
		{"services", domain.DepInternal},
		{"services.models", domain.DepInternal},
		{".utils", domain.DepInternal},
		{"..base", domain.DepInternal},
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
	d := python.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/src/services/payment.py", "/proj/tests/services/test_payment.py"},
		{"/proj/src/core/auth.py", "/proj/tests/core/test_auth.py"},
		{"/proj/myapp/models.py", "/proj/tests/myapp/test_models.py"},
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
