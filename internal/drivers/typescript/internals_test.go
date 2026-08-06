package typescript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

// ── trimQuotes ────────────────────────────────────────────────────────────────

// backfill / AC: trimQuotes strips surrounding double quotes
func TestTrimQuotes_DoubleQuotes(t *testing.T) {
	t.Parallel()
	got := trimQuotes(`"hello"`)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// backfill / AC: trimQuotes strips surrounding single quotes
func TestTrimQuotes_SingleQuotes(t *testing.T) {
	t.Parallel()
	got := trimQuotes(`'hello'`)
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// backfill / AC: trimQuotes strips surrounding backticks
func TestTrimQuotes_Backticks(t *testing.T) {
	t.Parallel()
	got := trimQuotes("`hello`")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// backfill / AC: trimQuotes leaves unquoted input unchanged
func TestTrimQuotes_NoQuotes(t *testing.T) {
	t.Parallel()
	got := trimQuotes("hello")
	if got != "hello" {
		t.Errorf("got %q, want %q", got, "hello")
	}
}

// backfill / AC: trimQuotes returns empty string unchanged
func TestTrimQuotes_Empty(t *testing.T) {
	t.Parallel()
	got := trimQuotes("")
	if got != "" {
		t.Errorf("got %q, want empty string", got)
	}
}

// ── FrameworkConfig ───────────────────────────────────────────────────────────

// backfill / AC: jestAdapter.FrameworkConfig returns the jest TestFrameworkConfig
func TestJestAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	cfg := (&jestAdapter{}).FrameworkConfig()
	if cfg.Name != "jest" {
		t.Errorf("Name: got %q, want \"jest\"", cfg.Name)
	}
	if cfg.TestFileSuffix != ".test.ts" {
		t.Errorf("TestFileSuffix: got %q, want \".test.ts\"", cfg.TestFileSuffix)
	}
	if cfg.FixtureDir != "__mocks__/" {
		t.Errorf("FixtureDir: got %q, want \"__mocks__/\"", cfg.FixtureDir)
	}
	if cfg.BootstrapFile != "jest.setup.ts" {
		t.Errorf("BootstrapFile: got %q, want \"jest.setup.ts\"", cfg.BootstrapFile)
	}
	if cfg.TestFuncPrefix != "it(" {
		t.Errorf("TestFuncPrefix: got %q, want \"it(\"", cfg.TestFuncPrefix)
	}
}

// backfill / AC: vitestAdapter.FrameworkConfig returns the vitest TestFrameworkConfig
func TestVitestAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	cfg := (&vitestAdapter{}).FrameworkConfig()
	if cfg.Name != "vitest" {
		t.Errorf("Name: got %q, want \"vitest\"", cfg.Name)
	}
	if cfg.TestFileSuffix != ".test.ts" {
		t.Errorf("TestFileSuffix: got %q, want \".test.ts\"", cfg.TestFileSuffix)
	}
	if cfg.BootstrapFile != "vitest.setup.ts" {
		t.Errorf("BootstrapFile: got %q, want \"vitest.setup.ts\"", cfg.BootstrapFile)
	}
	if cfg.TestFuncPrefix != "it(" {
		t.Errorf("TestFuncPrefix: got %q, want \"it(\"", cfg.TestFuncPrefix)
	}
}

// backfill / AC: mochaSinonAdapter.FrameworkConfig returns the mocha TestFrameworkConfig
func TestMochaSinonAdapter_FrameworkConfig(t *testing.T) {
	t.Parallel()
	cfg := (&mochaSinonAdapter{}).FrameworkConfig()
	if cfg.Name != "mocha" {
		t.Errorf("Name: got %q, want \"mocha\"", cfg.Name)
	}
	if cfg.FixtureDir != "test/fixtures/" {
		t.Errorf("FixtureDir: got %q, want \"test/fixtures/\"", cfg.FixtureDir)
	}
	if cfg.BootstrapFile != "" {
		t.Errorf("BootstrapFile: got %q, want empty (mocha has no bootstrap)", cfg.BootstrapFile)
	}
}

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

// backfill / AC: jestAdapter.LLMVocabulary returns jest-specific vocabulary map
func TestJestAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	vocab := (&jestAdapter{}).LLMVocabulary()
	if vocab["framework"] != "jest" {
		t.Errorf("framework: got %q, want \"jest\"", vocab["framework"])
	}
	if vocab["mock_library"] != "jest" {
		t.Errorf("mock_library: got %q, want \"jest\"", vocab["mock_library"])
	}
	if !strings.Contains(vocab["assert_style"], "toBe") {
		t.Errorf("assert_style missing \"toBe\": %q", vocab["assert_style"])
	}
	if !strings.Contains(vocab["mock_style"], "jest.fn()") {
		t.Errorf("mock_style missing \"jest.fn()\": %q", vocab["mock_style"])
	}
	if !strings.Contains(vocab["import_style"], "@jest/globals") {
		t.Errorf("import_style missing \"@jest/globals\": %q", vocab["import_style"])
	}
}

// backfill / AC: vitestAdapter.LLMVocabulary returns vitest-specific vocabulary map
func TestVitestAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	vocab := (&vitestAdapter{}).LLMVocabulary()
	if vocab["framework"] != "vitest" {
		t.Errorf("framework: got %q, want \"vitest\"", vocab["framework"])
	}
	if vocab["mock_library"] != "vitest" {
		t.Errorf("mock_library: got %q, want \"vitest\"", vocab["mock_library"])
	}
	if !strings.Contains(vocab["mock_style"], "vi.fn()") {
		t.Errorf("mock_style missing \"vi.fn()\": %q", vocab["mock_style"])
	}
	if !strings.Contains(vocab["import_style"], "vitest") {
		t.Errorf("import_style missing \"vitest\": %q", vocab["import_style"])
	}
}

// backfill / AC: mochaSinonAdapter.LLMVocabulary returns mocha+sinon vocabulary map
func TestMochaSinonAdapter_LLMVocabulary(t *testing.T) {
	t.Parallel()
	vocab := (&mochaSinonAdapter{}).LLMVocabulary()
	if vocab["framework"] != "mocha" {
		t.Errorf("framework: got %q, want \"mocha\"", vocab["framework"])
	}
	if vocab["mock_library"] != "sinon" {
		t.Errorf("mock_library: got %q, want \"sinon\"", vocab["mock_library"])
	}
	if !strings.Contains(vocab["assert_style"], "to.equal") {
		t.Errorf("assert_style missing \"to.equal\": %q", vocab["assert_style"])
	}
	if !strings.Contains(vocab["import_style"], "chai") {
		t.Errorf("import_style missing \"chai\": %q", vocab["import_style"])
	}
	if !strings.Contains(vocab["mock_style"], "sinon") {
		t.Errorf("mock_style missing \"sinon\": %q", vocab["mock_style"])
	}
}

// ── generateTestFile ──────────────────────────────────────────────────────────

func makeTsProject(root string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Root:     root,
		Language: "typescript",
		Metadata: map[string]any{"framework": "jest", "mock_library": "jest"},
	}
}

// backfill / AC: generateTestFile returns a GeneratedFile with RoleTestFile and .test.ts path
func TestGenerateTestFile_ProducesTestRoleAndCorrectPath(t *testing.T) {
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/auth/auth.service.ts",
		ModulePath: "src/auth/auth.service",
		PublicAPI: []domain.PublicMember{
			{
				Name: "AuthService",
				Kind: domain.KindClass,
				Methods: []domain.MethodInfo{
					{Name: "login", IsPublic: true},
				},
			},
		},
		Project: makeTsProject("/proj"),
	}
	gf, err := generateTestFile(analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("generateTestFile: %v", err)
	}
	if gf.Role != domain.RoleTestFile {
		t.Errorf("Role: got %q, want %q", gf.Role, domain.RoleTestFile)
	}
	if !strings.HasSuffix(gf.AbsPath, ".test.ts") {
		t.Errorf("AbsPath %q does not end with \".test.ts\"", gf.AbsPath)
	}
}

// backfill / AC: generateTestFile content references the exported member name
func TestGenerateTestFile_ContentContainsMemberName(t *testing.T) {
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/utils.ts",
		ModulePath: "src/utils",
		PublicAPI: []domain.PublicMember{
			{Name: "formatDate", Kind: domain.KindFunction},
		},
		Project: makeTsProject("/proj"),
	}
	gf, err := generateTestFile(analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("generateTestFile: %v", err)
	}
	if !strings.Contains(gf.Content, "formatDate") {
		t.Errorf("expected content to reference 'formatDate'")
	}
}

// ── generateMock ──────────────────────────────────────────────────────────────

// backfill / AC: generateMock returns a GeneratedFile with RoleFixture
func TestGenerateMock_ProducesFixtureRole(t *testing.T) {
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/services/payment.ts",
		Project:    makeTsProject("/proj"),
	}
	gf, err := generateMock("stripe", analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("generateMock: %v", err)
	}
	if gf.Role != domain.RoleFixture {
		t.Errorf("Role: got %q, want %q", gf.Role, domain.RoleFixture)
	}
}

// backfill / AC: generateMock content references the dep name and title-cased mock var
func TestGenerateMock_ContentContainsDep(t *testing.T) {
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/services/payment.ts",
		Project:    makeTsProject("/proj"),
	}
	gf, err := generateMock("stripe", analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("generateMock: %v", err)
	}
	if !strings.Contains(gf.Content, "stripe") {
		t.Errorf("expected content to reference dep name 'stripe'")
	}
	if !strings.Contains(gf.Content, "mockStripe") {
		t.Errorf("expected content to contain title-cased 'mockStripe'")
	}
}

// backfill / AC: generateMock with lang_config fixture_dir places file under that dir
func TestGenerateMock_WithFixtureDir_PlacesFileThere(t *testing.T) {
	ctx := &domain.ProjectContext{
		Root:     "/proj",
		Language: "typescript",
		Metadata: map[string]any{
			"framework":    "jest",
			"mock_library": "jest",
			"lang_config":  map[string]string{"fixture_dir": "__mocks__"},
		},
	}
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/services/payment.ts",
		Project:    ctx,
	}
	gf, err := generateMock("stripe", analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("generateMock: %v", err)
	}
	if !strings.Contains(filepath.ToSlash(gf.AbsPath), "__mocks__") {
		t.Errorf("AbsPath %q should be under __mocks__ when fixture_dir is configured", gf.AbsPath)
	}
}

// ── analyzeFile error path ────────────────────────────────────────────────────

// backfill / AC: analyzeFile returns an error when the source file does not exist
func TestAnalyzeFile_FileNotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	ctx := &domain.ProjectContext{Root: "/nonexistent", Language: "typescript", Metadata: map[string]any{}}
	_, err := analyzeFile("/nonexistent/path/file.ts", ctx)
	if err == nil {
		t.Error("expected error for a nonexistent source file")
	}
}

// ── parseLexicalDeclaration (exercised via analyzeFile with a temp file) ──────

// backfill / AC: exported const arrow functions are detected as KindFunction public members;
// private (underscore-prefixed) and non-arrow const values are excluded
func TestAnalyzeFile_ExportedConstArrowFunction(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// untyped params → exercises "identifier" branch in extractFormalParams
	// typed params   → exercises "required_parameter" branch
	// rest params    → exercises "rest_pattern" branch
	src := []byte(`import { readFile } from "fs";

export const formatDate = (d: Date): string => d.toISOString();
export const add = (a, b) => a + b;
export function concat(...items: string[]): string { return items.join(""); }
export const _privateArrow = () => {};
export const NOT_ARROW = "not-a-function";
export enum Color { Red, Green, Blue }
export interface Shape { draw(): void; }
export type Fn = () => void;
`)
	fpath := filepath.Join(dir, "utils.ts")
	if err := os.WriteFile(fpath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &domain.ProjectContext{Root: dir, Language: "typescript", Metadata: map[string]any{}}

	analysis, err := analyzeFile(fpath, ctx)
	if err != nil {
		t.Fatalf("analyzeFile: %v", err)
	}

	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}

	// Arrow function with typed param should appear
	if m, ok := membersByName["formatDate"]; !ok {
		t.Error("expected formatDate in public API")
	} else if m.Kind != domain.KindFunction {
		t.Errorf("formatDate kind: got %q, want function", m.Kind)
	}

	// Arrow function with untyped params should appear
	if _, ok := membersByName["add"]; !ok {
		t.Error("expected add (untyped params) in public API")
	}

	// Regular exported function should appear
	if _, ok := membersByName["concat"]; !ok {
		t.Error("expected concat function in public API")
	}

	// Private arrow function must not appear
	if _, ok := membersByName["_privateArrow"]; ok {
		t.Error("_privateArrow should not appear (private by naming convention)")
	}

	// Non-arrow const value must not appear
	if _, ok := membersByName["NOT_ARROW"]; ok {
		t.Error("NOT_ARROW (string value) should not appear — not an arrow function")
	}

	// Interface should appear (parsed as KindClass)
	if _, ok := membersByName["Shape"]; !ok {
		t.Error("expected Shape interface in public API")
	}

	// Type alias should appear
	if _, ok := membersByName["Fn"]; !ok {
		t.Error("expected Fn type alias in public API")
	}

	// Stdlib import from "fs" should be classified correctly
	stdlibMods := make(map[string]bool)
	for _, imp := range analysis.Imports.Stdlib {
		stdlibMods[imp.Module] = true
	}
	if !stdlibMods["fs"] {
		t.Error("expected 'fs' import to be classified as stdlib")
	}
}

// backfill / AC: analyzeFile with .tsx extension uses the tsx grammar (not the ts grammar)
func TestAnalyzeFile_TsxExtension_UsesTsxGrammar(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	src := []byte(`export function greeting(): string { return "hello"; }
`)
	fpath := filepath.Join(dir, "greeting.tsx")
	if err := os.WriteFile(fpath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &domain.ProjectContext{Root: dir, Language: "typescript", Metadata: map[string]any{}}

	analysis, err := analyzeFile(fpath, ctx)
	if err != nil {
		t.Fatalf("analyzeFile on .tsx file: %v", err)
	}
	membersByName := make(map[string]domain.PublicMember)
	for _, m := range analysis.PublicAPI {
		membersByName[m.Name] = m
	}
	if _, ok := membersByName["greeting"]; !ok {
		t.Error("expected greeting function in public API for .tsx file")
	}
}

// backfill / AC: require() calls are extracted as imports and classified correctly
func TestAnalyzeFile_RequireCallExtractsModule(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// "events" is a Node.js stdlib module — should be classified as DepStdlib.
	// foo("not-require") exercises the early-return path in parseRequireCall.
	src := []byte(`const events = require("events");
const x = foo("not-require");
`)
	fpath := filepath.Join(dir, "handler.ts")
	if err := os.WriteFile(fpath, src, 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := &domain.ProjectContext{Root: dir, Language: "typescript", Metadata: map[string]any{}}

	analysis, err := analyzeFile(fpath, ctx)
	if err != nil {
		t.Fatalf("analyzeFile: %v", err)
	}

	found := false
	for _, imp := range analysis.Imports.Stdlib {
		if imp.Module == "events" {
			found = true
		}
	}
	if !found {
		t.Error("expected require('events') to be classified as a stdlib import")
	}
}
