package python

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

func makePyAnalysis(kind string) *domain.SourceAnalysis {
	member := domain.PublicMember{
		Name: "PaymentService",
		Kind: domain.MemberKind(kind),
		Methods: []domain.MethodInfo{
			{Name: "process_payment", IsPublic: true},
			{Name: "__init__", IsPublic: true},
			{Name: "_internal", IsPublic: false},
		},
	}
	if kind == "function" {
		member.Name = "calculate_fee"
		member.Methods = nil
	}
	return &domain.SourceAnalysis{
		SourcePath: "/proj/src/payments/service.py",
		ModulePath: "payments.service",
		PublicAPI:  []domain.PublicMember{member},
		Project:    &domain.ProjectContext{Language: "python", Metadata: map[string]any{}},
	}
}

func pyCtxWith(framework, mockLib string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Language: "python",
		Metadata: map[string]any{"framework": framework, "mock_library": mockLib},
	}
}

// ── selectAdapter ─────────────────────────────────────────────────────────────

func TestPySelectAdapter_DefaultIsPytestMock(t *testing.T) {
	a := selectAdapter(nil)
	if a.Framework() != "pytest" || a.MockLibrary() != "pytest-mock" {
		t.Errorf("expected pytest+pytest-mock, got %s+%s", a.Framework(), a.MockLibrary())
	}
}

func TestPySelectAdapter_UnittestMock(t *testing.T) {
	a := selectAdapter(pyCtxWith("pytest", "unittest.mock"))
	if a.MockLibrary() != "unittest.mock" {
		t.Errorf("expected unittest.mock, got %q", a.MockLibrary())
	}
}

func TestPySelectAdapter_Unittest(t *testing.T) {
	a := selectAdapter(pyCtxWith("unittest", "unittest.mock"))
	if a.Framework() != "unittest" {
		t.Errorf("expected unittest, got %q", a.Framework())
	}
}

func TestPySelectAdapter_UnknownFallsBackToDefault(t *testing.T) {
	a := selectAdapter(pyCtxWith("nose", "mock"))
	if a.Framework() != "pytest" {
		t.Errorf("expected default fallback, got %q", a.Framework())
	}
}

// ── pytest + pytest-mock adapter ──────────────────────────────────────────────

func TestPytestMockAdapter_FunctionScaffold(t *testing.T) {
	a := &pytestPytestMockAdapter{}
	out, err := a.GenerateTestFile(makePyAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "import pytest") {
		t.Error("missing pytest import")
	}
	if !strings.Contains(out, "class TestCalculate_fee") {
		t.Error("missing test class")
	}
	if !strings.Contains(out, "def test_calculate_fee") {
		t.Error("missing test method")
	}
}

func TestPytestMockAdapter_ClassScaffold_ExcludesDunderAndPrivate(t *testing.T) {
	a := &pytestPytestMockAdapter{}
	out, err := a.GenerateTestFile(makePyAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "class TestPaymentService") {
		t.Error("missing test class")
	}
	if !strings.Contains(out, "def test_process_payment") {
		t.Error("missing test for public method")
	}
	if strings.Contains(out, "__init__") {
		t.Error("__init__ must be excluded from test methods")
	}
	if strings.Contains(out, "_internal") {
		t.Error("private method must be excluded")
	}
}

func TestPytestMockAdapter_AutouseFixture(t *testing.T) {
	a := &pytestPytestMockAdapter{}
	out, err := a.GenerateTestFile(makePyAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@pytest.fixture(autouse=True)") {
		t.Error("missing autouse fixture")
	}
}

// ── pytest + unittest.mock adapter ───────────────────────────────────────────

func TestPytestUnittestMockAdapter_InlineMagicMock(t *testing.T) {
	a := &pytestUnittestMockAdapter{}
	out, err := a.GenerateTestFile(makePyAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "from unittest.mock import MagicMock") {
		t.Error("missing MagicMock import")
	}
}

// ── unittest adapter ──────────────────────────────────────────────────────────

func TestUnittestAdapter_Scaffold(t *testing.T) {
	a := &unittestAdapter{}
	out, err := a.GenerateTestFile(makePyAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "import unittest") {
		t.Error("missing unittest import")
	}
	if !strings.Contains(out, "unittest.TestCase") {
		t.Error("missing TestCase base class")
	}
	if !strings.Contains(out, "def setUp(self)") {
		t.Error("missing setUp method")
	}
	if !strings.Contains(out, `unittest.main()`) {
		t.Error("missing unittest.main() guard")
	}
}

// ── LLM body injection ────────────────────────────────────────────────────────

func TestPytestMockAdapter_InjectsLLMBody(t *testing.T) {
	a := &pytestPytestMockAdapter{}
	opts := domain.GenerateOpts{
		LLMBodies: map[string][]string{
			"calculate_fee": {"result = fee * 0.1", "assert result > 0"},
		},
	}
	out, err := a.GenerateTestFile(makePyAnalysis("function"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "result = fee * 0.1") {
		t.Error("LLM body not injected")
	}
}
