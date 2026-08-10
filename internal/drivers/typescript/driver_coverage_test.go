package typescript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/typescript"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func jestCtx() *domain.ProjectContext {
	return &domain.ProjectContext{
		Root:     "/proj",
		Language: "typescript",
		Metadata: map[string]any{"framework": "jest", "mock_library": "jest"},
	}
}

func simpleAnalysis() *domain.SourceAnalysis {
	return &domain.SourceAnalysis{
		SourcePath: "/proj/src/utils.ts",
		ModulePath: "src/utils",
		PublicAPI:  []domain.PublicMember{{Name: "formatDate", Kind: domain.KindFunction}},
		Project:    jestCtx(),
	}
}

// ── Driver.Language ───────────────────────────────────────────────────────────

// backfill / AC: Driver.Language returns "typescript"
func TestDriver_Language(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	if d.Language() != "typescript" {
		t.Errorf("got %q, want \"typescript\"", d.Language())
	}
}

// ── Driver.FileExtensions ─────────────────────────────────────────────────────

// backfill / AC: Driver.FileExtensions includes .ts, .tsx, .js, .jsx
func TestDriver_FileExtensions(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	got := d.FileExtensions()

	want := map[string]bool{".ts": true, ".tsx": true, ".js": true, ".jsx": true}
	for _, ext := range got {
		if !want[ext] {
			t.Errorf("unexpected extension %q", ext)
		}
		delete(want, ext)
	}
	for ext := range want {
		t.Errorf("missing expected extension %q", ext)
	}
}

// ── Driver.BodyGenerationPrompt ───────────────────────────────────────────────

// backfill / AC: Driver.BodyGenerationPrompt returns a non-empty template string mentioning typescript
func TestDriver_BodyGenerationPrompt(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	prompt := d.BodyGenerationPrompt()
	if prompt == "" {
		t.Error("expected a non-empty BodyGenerationPrompt")
	}
	if !strings.Contains(prompt, "typescript") {
		t.Error("expected prompt to mention 'typescript'")
	}
}

// ── Driver.GetTestFrameworkConfig ─────────────────────────────────────────────

// backfill / AC: Driver.GetTestFrameworkConfig returns jest as the default framework
func TestDriver_GetTestFrameworkConfig(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	cfg := d.GetTestFrameworkConfig()
	if cfg.Name != "jest" {
		t.Errorf("Name: got %q, want \"jest\"", cfg.Name)
	}
	if cfg.TestFileSuffix != ".test.ts" {
		t.Errorf("TestFileSuffix: got %q, want \".test.ts\"", cfg.TestFileSuffix)
	}
	if cfg.TestFuncPrefix != "it(" {
		t.Errorf("TestFuncPrefix: got %q, want \"it(\"", cfg.TestFuncPrefix)
	}
}

// ── Driver.LLMContext ─────────────────────────────────────────────────────────

// backfill / AC: Driver.LLMContext injects "language":"typescript" alongside adapter vocabulary
func TestDriver_LLMContext_InjectsLanguageKey(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	vocab := d.LLMContext(jestCtx())
	if vocab["language"] != "typescript" {
		t.Errorf("language: got %q, want \"typescript\"", vocab["language"])
	}
	if vocab["framework"] == "" {
		t.Error("expected non-empty framework key in LLMContext output")
	}
}

// backfill / AC: Driver.LLMContext with nil ctx defaults to jest vocabulary + language key
func TestDriver_LLMContext_NilCtx_DefaultsToJest(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	vocab := d.LLMContext(nil)
	if vocab["language"] != "typescript" {
		t.Errorf("language: got %q, want \"typescript\"", vocab["language"])
	}
}

// ── Driver.DeriveModulePath ───────────────────────────────────────────────────

// backfill / AC: Driver.DeriveModulePath strips the file extension and returns the root-relative path
func TestDriver_DeriveModulePath(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "typescript", Metadata: map[string]any{}}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/src/services/payment.ts", "src/services/payment"},
		{"/proj/src/utils/auth.ts", "src/utils/auth"},
		{"/proj/components/Button.tsx", "components/Button"},
		{"/proj/index.js", "index"},
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

// ── Driver.GenerateTestFile ───────────────────────────────────────────────────

// backfill / AC: Driver.GenerateTestFile returns a non-nil GeneratedFile with RoleTestFile
func TestDriver_GenerateTestFile_ReturnsTestFile(t *testing.T) {
	d := typescript.New()
	gf, err := d.GenerateTestFile(simpleAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateTestFile: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}
	if gf.Role != domain.RoleTestFile {
		t.Errorf("Role: got %q, want %q", gf.Role, domain.RoleTestFile)
	}
	if !strings.Contains(gf.Content, "formatDate") {
		t.Errorf("expected content to reference 'formatDate'")
	}
}

// ── Driver.GenerateFixture ────────────────────────────────────────────────────

// backfill / AC: Driver.GenerateFixture returns a non-nil GeneratedFile with RoleFixture
func TestDriver_GenerateFixture_ReturnsFixtureFile(t *testing.T) {
	d := typescript.New()
	gf, err := d.GenerateFixture("stripe", simpleAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateFixture: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}
	if gf.Role != domain.RoleFixture {
		t.Errorf("Role: got %q, want %q", gf.Role, domain.RoleFixture)
	}
	if !strings.Contains(gf.Content, "stripe") {
		t.Errorf("expected fixture content to reference dep 'stripe'")
	}
}

// ── Driver.GenerateBootstrap ──────────────────────────────────────────────────

// backfill / AC: Driver.GenerateBootstrap returns nil, nil (TypeScript has no framework bootstrap)
func TestDriver_GenerateBootstrap_ReturnsNil(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	gf, err := d.GenerateBootstrap(nil, nil)
	if err != nil {
		t.Errorf("unexpected error from GenerateBootstrap: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil GeneratedFile, got %+v", gf)
	}
}

// ── Driver.ListAdapters ───────────────────────────────────────────────────────

// backfill / AC: Driver.ListAdapters returns all registered adapters; default context selects jest
func TestDriver_ListAdapters_DefaultContextSelectsJest(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	all, selected := d.ListAdapters(jestCtx())
	if len(all) == 0 {
		t.Error("expected at least one adapter in the registry")
	}
	if selected == nil {
		t.Fatal("expected a non-nil selected adapter")
	}
	if selected.Framework() != "jest" {
		t.Errorf("selected framework: got %q, want \"jest\"", selected.Framework())
	}
}

// backfill / AC: Driver.ListAdapters selects vitest when context specifies vitest
func TestDriver_ListAdapters_VitestContextSelectsVitest(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	ctx := &domain.ProjectContext{
		Language: "typescript",
		Metadata: map[string]any{"framework": "vitest", "mock_library": "vitest"},
	}
	_, selected := d.ListAdapters(ctx)
	if selected.Framework() != "vitest" {
		t.Errorf("selected framework: got %q, want \"vitest\"", selected.Framework())
	}
}

// backfill / AC: Driver.ListAdapters returns all three registered adapters (jest, vitest, mocha)
func TestDriver_ListAdapters_ContainsAllThreeAdapters(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	all, _ := d.ListAdapters(nil)
	frameworks := make(map[string]bool)
	for _, a := range all {
		frameworks[a.Framework()] = true
	}
	for _, want := range []string{"jest", "vitest", "mocha"} {
		if !frameworks[want] {
			t.Errorf("expected adapter for framework %q", want)
		}
	}
}

// ── Driver.ValidateFile ───────────────────────────────────────────────────────

// backfill / AC: ValidateFile with jest framework flags vi.* API usage as an error
func TestDriver_ValidateFile_Jest_FlagsVitestAPI(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	issues := d.ValidateFile("jest", "jest", "const spy = vi.fn();")
	if len(issues) == 0 {
		t.Error("expected at least one issue when vi.fn() is used in a jest file")
	}
	foundError := false
	for _, issue := range issues {
		if issue.Severity == domain.SeverityError {
			foundError = true
		}
	}
	if !foundError {
		t.Error("expected an error-severity issue for vi.* usage in jest context")
	}
}

// backfill / AC: ValidateFile with vitest framework flags jest.* API usage as a warning
func TestDriver_ValidateFile_Vitest_FlagsJestAPI(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	issues := d.ValidateFile("vitest", "vitest", "const spy = jest.fn();")
	if len(issues) == 0 {
		t.Error("expected at least one issue when jest.fn() is used in a vitest file")
	}
	foundWarning := false
	for _, issue := range issues {
		if issue.Severity == domain.SeverityWarning {
			foundWarning = true
		}
	}
	if !foundWarning {
		t.Error("expected a warning-severity issue for jest.* usage in vitest context")
	}
}

// backfill / AC: ValidateFile with an unknown framework/mockLib pair returns nil
func TestDriver_ValidateFile_UnknownFramework_ReturnsNil(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	issues := d.ValidateFile("jasmine", "unknown", "const x = jasmine.createSpy();")
	if issues != nil {
		t.Errorf("expected nil for unknown framework, got %v", issues)
	}
}

// backfill / AC: ValidateFile with jest framework and clean jest content returns no issues
func TestDriver_ValidateFile_Jest_CleanContent_NoIssues(t *testing.T) {
	t.Parallel()
	d := typescript.New()
	issues := d.ValidateFile("jest", "jest", "const spy = jest.fn();")
	if len(issues) != 0 {
		t.Errorf("expected no issues for valid jest content, got %v", issues)
	}
}

// ── Driver.DetectProject with temp directories ────────────────────────────────

// backfill / AC: DetectProject detects vitest when listed in devDependencies
func TestDriver_DetectProject_VitestPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkgJSON := `{"devDependencies": {"vitest": "^1.0.0", "typescript": "^5.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	d := typescript.New()
	ctx, err := d.DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	fw, _ := ctx.Metadata["framework"].(string)
	if fw != "vitest" {
		t.Errorf("framework: got %q, want \"vitest\"", fw)
	}
	ml, _ := ctx.Metadata["mock_library"].(string)
	if ml != "vitest" {
		t.Errorf("mock_library: got %q, want \"vitest\"", ml)
	}
}

// backfill / AC: DetectProject detects mocha+sinon when both are listed in devDependencies
func TestDriver_DetectProject_MochaPlusSinonPackage(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	pkgJSON := `{"devDependencies": {"mocha": "^10.0.0", "sinon": "^15.0.0"}}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	d := typescript.New()
	ctx, err := d.DetectProject(dir)
	if err != nil {
		t.Fatalf("DetectProject: %v", err)
	}
	fw, _ := ctx.Metadata["framework"].(string)
	if fw != "mocha" {
		t.Errorf("framework: got %q, want \"mocha\"", fw)
	}
	ml, _ := ctx.Metadata["mock_library"].(string)
	if ml != "sinon" {
		t.Errorf("mock_library: got %q, want \"sinon\"", ml)
	}
}

// backfill / AC: DetectProject returns ErrProjectNotFound when .git stops the walk before package.json
func TestDriver_DetectProject_StopsAtGitBoundary(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// Place a .git marker in the temp root — no package.json anywhere.
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(dir, "src")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	d := typescript.New()
	_, err := d.DetectProject(subdir)
	if err == nil {
		t.Error("expected ErrProjectNotFound when .git stops the walk before any package.json")
	}
}
