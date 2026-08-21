package golang

import (
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
)

func makeGoAnalysis(kind string) *domain.SourceAnalysis {
	member := domain.PublicMember{
		Name: "Calculator",
		Kind: domain.MemberKind(kind),
		Methods: []domain.MethodInfo{
			{Name: "Add", IsPublic: true},
			{Name: "privateHelper", IsPublic: false},
		},
	}
	if kind == "function" {
		member.Name = "Calculate"
		member.Methods = nil
	}
	return &domain.SourceAnalysis{
		SourcePath: "/proj/pkg/math/calc.go",
		ModulePath: "github.com/example/proj/pkg/math",
		PublicAPI:  []domain.PublicMember{member},
		Project:    &domain.ProjectContext{Language: "go", Metadata: map[string]any{}},
	}
}

func ctxWith(framework, mockLib string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Language: "go",
		Metadata: map[string]any{"framework": framework, "mock_library": mockLib},
	}
}

// ── selectAdapter ─────────────────────────────────────────────────────────────

func TestSelectAdapter_Default(t *testing.T) {
	a := selectAdapter(nil)
	if a.MockLibrary() != "interfaces" {
		t.Errorf("expected default mock_library 'interfaces', got %q", a.MockLibrary())
	}
}

func TestSelectAdapter_Testify(t *testing.T) {
	a := selectAdapter(ctxWith("testing", "testify"))
	if a.MockLibrary() != "testify" {
		t.Errorf("expected testify, got %q", a.MockLibrary())
	}
}

func TestSelectAdapter_Gomock(t *testing.T) {
	a := selectAdapter(ctxWith("testing", "gomock"))
	if a.MockLibrary() != "gomock" {
		t.Errorf("expected gomock, got %q", a.MockLibrary())
	}
}

func TestSelectAdapter_UnknownFallsBackToDefault(t *testing.T) {
	a := selectAdapter(ctxWith("unknown-framework", "unknown-lib"))
	if a.MockLibrary() != "interfaces" {
		t.Errorf("expected default fallback, got %q", a.MockLibrary())
	}
}

// ── stdlib adapter ────────────────────────────────────────────────────────────

func TestStdlibAdapter_GeneratesTableDrivenTest_ForFunction(t *testing.T) {
	a := &stdlibAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, `package math_test`) {
		t.Error("missing package declaration")
	}
	if !strings.Contains(out, `"testing"`) {
		t.Error("missing testing import")
	}
	if !strings.Contains(out, "TestCalculate") {
		t.Error("missing test function name")
	}
	if !strings.Contains(out, "t.Skip") {
		t.Error("missing t.Skip placeholder")
	}
}

func TestStdlibAdapter_GeneratesMethodTests_ForClass(t *testing.T) {
	a := &stdlibAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "TestCalculator_Add") {
		t.Error("missing method test name")
	}
	// private method must be excluded
	if strings.Contains(out, "privateHelper") {
		t.Error("private method must not appear in test")
	}
}

// ── testify adapter ───────────────────────────────────────────────────────────

func TestTestifyAdapter_Imports(t *testing.T) {
	a := &testifyAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "github.com/stretchr/testify/assert") {
		t.Error("missing testify assert import")
	}
	if !strings.Contains(out, "assert.True") {
		t.Error("missing assert.True call")
	}
}

func TestTestifyAdapter_MockStruct_ForClass(t *testing.T) {
	a := &testifyAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "MockCalculator") {
		t.Error("missing Mock struct")
	}
	if !strings.Contains(out, "mock.Mock") {
		t.Error("missing embedded mock.Mock")
	}
}

// ── gomock adapter ────────────────────────────────────────────────────────────

func TestGomockAdapter_ControllerSetup(t *testing.T) {
	a := &gomockAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "go.uber.org/mock/gomock") {
		t.Error("missing gomock import")
	}
	if !strings.Contains(out, "gomock.NewController") {
		t.Error("missing controller setup")
	}
	if !strings.Contains(out, "ctrl.Finish()") {
		t.Error("missing ctrl.Finish")
	}
}

func TestGomockAdapter_GenerateDirective_ForClass(t *testing.T) {
	a := &gomockAdapter{}
	out, err := a.GenerateTestFile(makeGoAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "go:generate") {
		t.Error("missing go:generate directive")
	}
}

// ── LLM body injection ────────────────────────────────────────────────────────

func TestStdlibAdapter_InjectsLLMBody(t *testing.T) {
	a := &stdlibAdapter{}
	opts := domain.GenerateOpts{
		LLMBodies: map[string][]string{
			"Calculate": {"result := 42", "return result"},
		},
	}
	out, err := a.GenerateTestFile(makeGoAnalysis("function"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "result := 42") {
		t.Error("LLM body not injected")
	}
	if strings.Contains(out, "t.Skip") {
		t.Error("default placeholder should be replaced by LLM body")
	}
}
