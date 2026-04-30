package csharp

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

func makeCsAnalysis() *domain.SourceAnalysis {
	return &domain.SourceAnalysis{
		SourcePath: "/proj/Services/CartService.cs",
		ModulePath: "MyApp.Services.CartService",
		PublicAPI: []domain.PublicMember{
			{
				Name: "CartService",
				Kind: "class",
				Methods: []domain.MethodInfo{
					{Name: "AddItem", IsPublic: true},
					{Name: "RemoveItem", IsPublic: true},
					{Name: "privateValidate", IsPublic: false},
				},
			},
		},
		Project: &domain.ProjectContext{Language: "csharp", Metadata: map[string]any{}},
	}
}

func csCtxWith(framework, mockLib string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Language: "csharp",
		Metadata: map[string]any{"framework": framework, "mock_library": mockLib},
	}
}

// ── selectAdapter ─────────────────────────────────────────────────────────────

func TestCsSelectAdapter_DefaultIsXUnitMoq(t *testing.T) {
	a := selectAdapter(nil)
	if a.Framework() != "xunit" || a.MockLibrary() != "moq" {
		t.Errorf("expected xunit+moq, got %s+%s", a.Framework(), a.MockLibrary())
	}
}

func TestCsSelectAdapter_NUnit(t *testing.T) {
	a := selectAdapter(csCtxWith("nunit", "moq"))
	if a.Framework() != "nunit" {
		t.Errorf("expected nunit, got %q", a.Framework())
	}
}

func TestCsSelectAdapter_MSTest(t *testing.T) {
	a := selectAdapter(csCtxWith("mstest", "moq"))
	if a.Framework() != "mstest" {
		t.Errorf("expected mstest, got %q", a.Framework())
	}
}

func TestCsSelectAdapter_XUnitNSubstitute(t *testing.T) {
	a := selectAdapter(csCtxWith("xunit", "nsubstitute"))
	if a.MockLibrary() != "nsubstitute" {
		t.Errorf("expected nsubstitute, got %q", a.MockLibrary())
	}
}

func TestCsSelectAdapter_NUnitNSubstitute(t *testing.T) {
	a := selectAdapter(csCtxWith("nunit", "nsubstitute"))
	if a.Framework() != "nunit" || a.MockLibrary() != "nsubstitute" {
		t.Errorf("expected nunit+nsubstitute, got %s+%s", a.Framework(), a.MockLibrary())
	}
}

func TestCsSelectAdapter_UnknownFallsBackToDefault(t *testing.T) {
	a := selectAdapter(csCtxWith("specflow", "fakeiteasy"))
	if a.Framework() != "xunit" {
		t.Errorf("expected xunit fallback, got %q", a.Framework())
	}
}

// ── xUnit + Moq adapter ───────────────────────────────────────────────────────

func TestXUnitMoqAdapter_Imports(t *testing.T) {
	a := &xunitMoqAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "using Xunit;") {
		t.Error("missing Xunit using")
	}
	if !strings.Contains(out, "using Moq;") {
		t.Error("missing Moq using")
	}
	if !strings.Contains(out, "[Fact]") {
		t.Error("missing [Fact] attribute")
	}
}

func TestXUnitMoqAdapter_TestMethods(t *testing.T) {
	a := &xunitMoqAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "AddItem_ShouldWork") {
		t.Error("missing test for AddItem")
	}
	if !strings.Contains(out, "RemoveItem_ShouldWork") {
		t.Error("missing test for RemoveItem")
	}
	if strings.Contains(out, "privateValidate") {
		t.Error("private method must not appear in test")
	}
}

// ── NUnit + Moq adapter ───────────────────────────────────────────────────────

func TestNUnitMoqAdapter_Annotations(t *testing.T) {
	a := &nunitMoqAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "using NUnit.Framework;") {
		t.Error("missing NUnit using")
	}
	if !strings.Contains(out, "[TestFixture]") {
		t.Error("missing [TestFixture]")
	}
	if !strings.Contains(out, "[SetUp]") {
		t.Error("missing [SetUp]")
	}
	if !strings.Contains(out, "[Test]") {
		t.Error("missing [Test]")
	}
}

// ── MSTest + Moq adapter ──────────────────────────────────────────────────────

func TestMSTestMoqAdapter_Annotations(t *testing.T) {
	a := &mstestMoqAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "[TestClass]") {
		t.Error("missing [TestClass]")
	}
	if !strings.Contains(out, "[TestInitialize]") {
		t.Error("missing [TestInitialize]")
	}
	if !strings.Contains(out, "[TestMethod]") {
		t.Error("missing [TestMethod]")
	}
}

// ── NSubstitute variants ──────────────────────────────────────────────────────

func TestXUnitNSubstituteAdapter_SubstituteFor(t *testing.T) {
	a := &xunitNSubstituteAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "using NSubstitute;") {
		t.Error("missing NSubstitute using")
	}
	if !strings.Contains(out, "Substitute.For") {
		t.Error("missing Substitute.For comment")
	}
}

func TestNUnitNSubstituteAdapter_SubstituteFor(t *testing.T) {
	a := &nunitNSubstituteAdapter{}
	out, err := a.GenerateTestFile(makeCsAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "using NSubstitute;") {
		t.Error("missing NSubstitute using")
	}
	if !strings.Contains(out, "[TestFixture]") {
		t.Error("missing [TestFixture]")
	}
}

// ── LLM body injection ────────────────────────────────────────────────────────

func TestXUnitMoqAdapter_InjectsLLMBody(t *testing.T) {
	a := &xunitMoqAdapter{}
	opts := domain.GenerateOpts{
		LLMBodies: map[string][]string{
			"AddItem": {"var result = _sut.AddItem(item);", "Assert.NotNull(result);"},
		},
	}
	out, err := a.GenerateTestFile(makeCsAnalysis(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "var result = _sut.AddItem(item)") {
		t.Error("LLM body not injected")
	}
}
