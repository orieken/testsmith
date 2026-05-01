package typescript

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

func makeTsAnalysis(kind string) *domain.SourceAnalysis {
	member := domain.PublicMember{
		Name: "AuthService",
		Kind: domain.MemberKind(kind),
		Methods: []domain.MethodInfo{
			{Name: "login", IsPublic: true},
			{Name: "logout", IsPublic: true},
			{Name: "privateRefresh", IsPublic: false},
		},
	}
	if kind == "function" {
		member.Name = "formatDate"
		member.Methods = nil
	}
	return &domain.SourceAnalysis{
		SourcePath: "/proj/src/auth/auth.service.ts",
		ModulePath: "src/auth/auth.service",
		PublicAPI:  []domain.PublicMember{member},
		Project:    &domain.ProjectContext{Language: "typescript", Metadata: map[string]any{}},
	}
}

func tsCtxWith(framework, mockLib string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Language: "typescript",
		Metadata: map[string]any{"framework": framework, "mock_library": mockLib},
	}
}

// ── selectAdapter ─────────────────────────────────────────────────────────────

func TestTsSelectAdapter_DefaultIsJest(t *testing.T) {
	a := selectAdapter(nil)
	if a.Framework() != "jest" {
		t.Errorf("expected jest, got %q", a.Framework())
	}
}

func TestTsSelectAdapter_Vitest(t *testing.T) {
	a := selectAdapter(tsCtxWith("vitest", "vitest"))
	if a.Framework() != "vitest" {
		t.Errorf("expected vitest, got %q", a.Framework())
	}
}

func TestTsSelectAdapter_MochaSinon(t *testing.T) {
	a := selectAdapter(tsCtxWith("mocha", "sinon"))
	if a.Framework() != "mocha" || a.MockLibrary() != "sinon" {
		t.Errorf("expected mocha+sinon, got %s+%s", a.Framework(), a.MockLibrary())
	}
}

func TestTsSelectAdapter_UnknownFallsBackToDefault(t *testing.T) {
	a := selectAdapter(tsCtxWith("jasmine", "unknown"))
	if a.Framework() != "jest" {
		t.Errorf("expected jest fallback, got %q", a.Framework())
	}
}

// ── Jest adapter ──────────────────────────────────────────────────────────────

func TestJestAdapter_FunctionImports(t *testing.T) {
	a := &jestAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@jest/globals") {
		t.Error("missing @jest/globals import")
	}
	if !strings.Contains(out, "describe('formatDate'") {
		t.Error("missing describe block")
	}
	if !strings.Contains(out, "expect(true).toBe(true)") {
		t.Error("missing jest assertion placeholder")
	}
}

func TestJestAdapter_ClassScaffold(t *testing.T) {
	a := &jestAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "let instance: AuthService") {
		t.Error("missing instance variable")
	}
	if !strings.Contains(out, "jest.fn()") {
		t.Error("missing jest.fn() hint")
	}
	// Private methods must not appear
	if strings.Contains(out, "privateRefresh") {
		t.Error("private method must not appear in test")
	}
}

// ── Vitest adapter ────────────────────────────────────────────────────────────

func TestVitestAdapter_Imports(t *testing.T) {
	a := &vitestAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "from 'vitest'") {
		t.Error("missing vitest import")
	}
	if strings.Contains(out, "@jest/globals") {
		t.Error("should not import @jest/globals")
	}
}

func TestVitestAdapter_ViFn(t *testing.T) {
	a := &vitestAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "vi.fn()") {
		t.Error("missing vi.fn() hint")
	}
}

// ── Mocha + Sinon adapter ─────────────────────────────────────────────────────

func TestMochaSinonAdapter_Imports(t *testing.T) {
	a := &mochaSinonAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("function"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "from 'mocha'") {
		t.Error("missing mocha import")
	}
	if !strings.Contains(out, "from 'chai'") {
		t.Error("missing chai import")
	}
	if !strings.Contains(out, "sinon") {
		t.Error("missing sinon import")
	}
}

func TestMochaSinonAdapter_SandboxLifecycle(t *testing.T) {
	a := &mochaSinonAdapter{}
	out, err := a.GenerateTestFile(makeTsAnalysis("class"), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "sinon.createSandbox()") {
		t.Error("missing sandbox creation")
	}
	if !strings.Contains(out, "sandbox.restore()") {
		t.Error("missing sandbox restore")
	}
}

// ── LLM body injection ────────────────────────────────────────────────────────

func TestJestAdapter_InjectsLLMBody(t *testing.T) {
	a := &jestAdapter{}
	opts := domain.GenerateOpts{
		LLMBodies: map[string][]string{
			"formatDate": {"const result = formatDate(new Date());", "expect(result).toBe('2024-01-01');"},
		},
	}
	out, err := a.GenerateTestFile(makeTsAnalysis("function"), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "expect(result).toBe('2024-01-01')") {
		t.Error("LLM body not injected")
	}
}
