package python_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/drivers/python"
)

// ── Language / FileExtensions / BodyGenerationPrompt ──────────────────────────

// backfill / AC: Language returns "python"
func TestDriver_Language(t *testing.T) {
	t.Parallel()
	d := python.New()
	if got := d.Language(); got != "python" {
		t.Errorf("Language() = %q, want %q", got, "python")
	}
}

// backfill / AC: FileExtensions returns [".py"]
func TestDriver_FileExtensions(t *testing.T) {
	t.Parallel()
	d := python.New()
	exts := d.FileExtensions()
	if len(exts) != 1 || exts[0] != ".py" {
		t.Errorf("FileExtensions() = %v, want [\".py\"]", exts)
	}
}

// backfill / AC: BodyGenerationPrompt returns a non-empty Python-specific prompt template
func TestDriver_BodyGenerationPrompt(t *testing.T) {
	t.Parallel()
	d := python.New()
	prompt := d.BodyGenerationPrompt()
	if prompt == "" {
		t.Fatal("BodyGenerationPrompt() must not be empty")
	}
	if !strings.Contains(prompt, "Python") {
		t.Error("BodyGenerationPrompt() must reference Python")
	}
	// Must contain template placeholders used by the LLM pipeline.
	if !strings.Contains(prompt, "{{") {
		t.Error("BodyGenerationPrompt() must contain Go template placeholders")
	}
}

// ── LLMContext ─────────────────────────────────────────────────────────────────

// backfill / AC: LLMContext always sets "language" = "python" and includes adapter vocabulary
func TestDriver_LLMContext_ContainsLanguageKey(t *testing.T) {
	t.Parallel()
	d := python.New()
	ctx := &domain.ProjectContext{
		Language: "python",
		Metadata: map[string]any{"framework": "pytest", "mock_library": "pytest-mock"},
	}
	vocab := d.LLMContext(ctx)
	if vocab["language"] != "python" {
		t.Errorf("LLMContext()[\"language\"] = %q, want %q", vocab["language"], "python")
	}
	if vocab["framework"] == "" {
		t.Error("LLMContext() must include \"framework\" from the adapter vocabulary")
	}
	if vocab["mock_library"] == "" {
		t.Error("LLMContext() must include \"mock_library\" from the adapter vocabulary")
	}
}

// backfill / AC: LLMContext with nil ctx uses the default adapter and still sets "language"
func TestDriver_LLMContext_NilCtx(t *testing.T) {
	t.Parallel()
	d := python.New()
	vocab := d.LLMContext(nil)
	if vocab["language"] != "python" {
		t.Errorf("LLMContext(nil)[\"language\"] = %q, want %q", vocab["language"], "python")
	}
	// Default adapter is pytest + pytest-mock.
	if vocab["framework"] != "pytest" {
		t.Errorf("LLMContext(nil)[\"framework\"] = %q, want %q", vocab["framework"], "pytest")
	}
}

// ── GetTestFrameworkConfig ─────────────────────────────────────────────────────

// backfill / AC: GetTestFrameworkConfig returns the pytest framework conventions
func TestDriver_GetTestFrameworkConfig(t *testing.T) {
	t.Parallel()
	d := python.New()
	cfg := d.GetTestFrameworkConfig()
	if cfg.Name != "pytest" {
		t.Errorf("GetTestFrameworkConfig().Name = %q, want %q", cfg.Name, "pytest")
	}
	if cfg.TestFilePrefix != "test_" {
		t.Errorf("GetTestFrameworkConfig().TestFilePrefix = %q, want %q", cfg.TestFilePrefix, "test_")
	}
	if cfg.TestFileSuffix != ".py" {
		t.Errorf("GetTestFrameworkConfig().TestFileSuffix = %q, want %q", cfg.TestFileSuffix, ".py")
	}
	if cfg.BootstrapFile != "conftest.py" {
		t.Errorf("GetTestFrameworkConfig().BootstrapFile = %q, want %q", cfg.BootstrapFile, "conftest.py")
	}
	if cfg.TestFuncPrefix != "test_" {
		t.Errorf("GetTestFrameworkConfig().TestFuncPrefix = %q, want %q", cfg.TestFuncPrefix, "test_")
	}
	if cfg.FixtureDir != "tests/fixtures/" {
		t.Errorf("GetTestFrameworkConfig().FixtureDir = %q, want %q", cfg.FixtureDir, "tests/fixtures/")
	}
}

// ── ListAdapters ──────────────────────────────────────────────────────────────

// backfill / AC: ListAdapters returns at least 3 adapters; selected matches context metadata
func TestDriver_ListAdapters_SelectsByContext(t *testing.T) {
	t.Parallel()
	d := python.New()
	ctx := &domain.ProjectContext{
		Language: "python",
		Metadata: map[string]any{"framework": "pytest", "mock_library": "pytest-mock"},
	}
	available, selected := d.ListAdapters(ctx)
	if len(available) < 3 {
		t.Errorf("ListAdapters() returned %d adapters, want >= 3", len(available))
	}
	if selected == nil {
		t.Fatal("ListAdapters() selected adapter must not be nil")
	}
	if selected.Framework() != "pytest" || selected.MockLibrary() != "pytest-mock" {
		t.Errorf("selected = %s+%s, want pytest+pytest-mock", selected.Framework(), selected.MockLibrary())
	}
}

// backfill / AC: ListAdapters with nil ctx returns the default (pytest+pytest-mock) adapter
func TestDriver_ListAdapters_NilCtxUsesDefault(t *testing.T) {
	t.Parallel()
	d := python.New()
	available, selected := d.ListAdapters(nil)
	if len(available) == 0 {
		t.Error("ListAdapters(nil) must return at least one adapter")
	}
	if selected == nil {
		t.Fatal("ListAdapters(nil) selected must not be nil")
	}
	if selected.Framework() != "pytest" {
		t.Errorf("ListAdapters(nil) default framework = %q, want %q", selected.Framework(), "pytest")
	}
}

// ── DeriveModulePath ──────────────────────────────────────────────────────────

// backfill / AC: DeriveModulePath converts a src-layout source path to a dotted module path
func TestDriver_DeriveModulePath(t *testing.T) {
	t.Parallel()
	d := python.New()
	ctx := &domain.ProjectContext{Root: "/proj", Language: "python"}

	cases := []struct {
		source string
		want   string
	}{
		{"/proj/src/services/payment.py", "services.payment"},
		{"/proj/src/core/auth.py", "core.auth"},
		{"/proj/myapp/models.py", "myapp.models"},
		{"/proj/utils.py", "utils"},
	}
	for _, tc := range cases {
		got, err := d.DeriveModulePath(tc.source, ctx)
		if err != nil {
			t.Errorf("DeriveModulePath(%q): %v", tc.source, err)
			continue
		}
		if got != tc.want {
			t.Errorf("DeriveModulePath(%q) = %q, want %q", tc.source, got, tc.want)
		}
	}
}

// ── GenerateTestFile ──────────────────────────────────────────────────────────

// backfill / AC: GenerateTestFile returns a GeneratedFile with a pytest scaffold
func TestDriver_GenerateTestFile(t *testing.T) {
	t.Parallel()
	d := python.New()
	analysis := &domain.SourceAnalysis{
		SourcePath: "/proj/src/billing/invoice.py",
		ModulePath: "billing.invoice",
		PublicAPI: []domain.PublicMember{
			{Name: "calculate_tax", Kind: domain.KindFunction},
		},
		Project: &domain.ProjectContext{
			Root:     "/proj",
			Language: "python",
			Metadata: map[string]any{},
		},
	}

	gf, err := d.GenerateTestFile(analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateTestFile: %v", err)
	}
	if gf == nil {
		t.Fatal("GenerateTestFile returned nil")
	}
	if !strings.HasSuffix(gf.AbsPath, "test_invoice.py") {
		t.Errorf("AbsPath = %q, want suffix 'test_invoice.py'", gf.AbsPath)
	}
	if !strings.Contains(gf.Content, "import pytest") {
		t.Error("generated content must contain 'import pytest'")
	}
	if gf.Role != domain.RoleTestFile {
		t.Errorf("Role = %q, want %q", gf.Role, domain.RoleTestFile)
	}
}

// ── GenerateFixture ───────────────────────────────────────────────────────────

// backfill / AC: GenerateFixture creates a fresh pytest fixture file for an external dependency
func TestDriver_GenerateFixture_NewFile(t *testing.T) {
	d := python.New()
	root := t.TempDir()
	ctx := &domain.ProjectContext{
		Root:     root,
		Language: "python",
		Metadata: map[string]any{
			"lang_config": map[string]string{"fixture_dir": "tests/fixtures"},
		},
	}
	analysis := &domain.SourceAnalysis{
		SourcePath: filepath.Join(root, "src", "services", "payment.py"),
		ModulePath: "services.payment",
		Imports: domain.ClassifiedImports{
			External: []domain.ImportInfo{
				{Module: "stripe", IsFrom: false},
				{Module: "stripe.error", IsFrom: true},
			},
		},
		Project: ctx,
	}

	gf, err := d.GenerateFixture("stripe", analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateFixture: %v", err)
	}
	if gf == nil {
		t.Fatal("GenerateFixture returned nil")
	}
	if !strings.HasSuffix(gf.AbsPath, "stripe_fixture.py") {
		t.Errorf("AbsPath = %q, want suffix 'stripe_fixture.py'", gf.AbsPath)
	}
	if !strings.Contains(gf.Content, "mock_stripe") {
		t.Error("fixture content must contain 'mock_stripe'")
	}
	if gf.Role != domain.RoleFixture {
		t.Errorf("Role = %q, want %q", gf.Role, domain.RoleFixture)
	}
	// Sub-module should also be referenced in the generated fixture.
	if !strings.Contains(gf.Content, "stripe.error") {
		t.Error("fixture content must reference the stripe.error sub-module")
	}
}

// backfill / AC: GenerateFixture merges new sub-modules into an existing fixture file
func TestDriver_GenerateFixture_MergeExisting(t *testing.T) {
	d := python.New()
	root := t.TempDir()

	fixtureDir := filepath.Join(root, "tests", "fixtures")
	if err := os.MkdirAll(fixtureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// Existing fixture only knows about root "stripe", not "stripe.error".
	existingContent := `"""Shared mock fixtures for the stripe external dependency."""
import pytest


@pytest.fixture
def mock_stripe(mocker):
    """Mock for stripe and its sub-modules."""
    mock = mocker.Mock()
    mocker.patch.dict("sys.modules", {
        "stripe": mock,
    })
    return mock
`
	fixturePath := filepath.Join(fixtureDir, "stripe_fixture.py")
	if err := os.WriteFile(fixturePath, []byte(existingContent), 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := &domain.ProjectContext{
		Root:     root,
		Language: "python",
		Metadata: map[string]any{
			"lang_config": map[string]string{"fixture_dir": "tests/fixtures"},
		},
	}
	analysis := &domain.SourceAnalysis{
		SourcePath: filepath.Join(root, "src", "billing", "invoice.py"),
		ModulePath: "billing.invoice",
		Imports: domain.ClassifiedImports{
			External: []domain.ImportInfo{
				{Module: "stripe", IsFrom: false},
				{Module: "stripe.error", IsFrom: true},
			},
		},
		Project: ctx,
	}

	gf, err := d.GenerateFixture("stripe", analysis, domain.GenerateOpts{})
	if err != nil {
		t.Fatalf("GenerateFixture merge: %v", err)
	}
	if gf == nil {
		t.Fatal("GenerateFixture returned nil for merge case")
	}
	if !strings.Contains(gf.Content, "stripe.error") {
		t.Error("merged fixture must include the stripe.error sub-module")
	}
}

// ── GenerateBootstrap ─────────────────────────────────────────────────────────

// backfill / AC: GenerateBootstrap returns nil when the plan has no new test files
func TestDriver_GenerateBootstrap_EmptyPlan(t *testing.T) {
	d := python.New()
	root := t.TempDir()
	ctx := &domain.ProjectContext{Root: root, Language: "python"}

	gf, err := d.GenerateBootstrap(&domain.GenerationPlan{}, ctx)
	if err != nil {
		t.Fatalf("GenerateBootstrap empty plan: %v", err)
	}
	if gf != nil {
		t.Errorf("GenerateBootstrap with no test files must return nil, got %+v", gf)
	}
}

// backfill / AC: GenerateBootstrap creates a conftest.py with sys.path entries for new test files
func TestDriver_GenerateBootstrap_CreatesFresh(t *testing.T) {
	d := python.New()
	root := t.TempDir()
	ctx := &domain.ProjectContext{Root: root, Language: "python"}
	plan := &domain.GenerationPlan{
		Files: []domain.GeneratedFile{
			{
				AbsPath: filepath.Join(root, "tests", "src", "api", "test_client.py"),
				Role:    domain.RoleTestFile,
				Action:  domain.ActionCreate,
			},
		},
	}

	gf, err := d.GenerateBootstrap(plan, ctx)
	if err != nil {
		t.Fatalf("GenerateBootstrap: %v", err)
	}
	if gf == nil {
		t.Fatal("GenerateBootstrap must return a GeneratedFile when new paths are needed")
	}
	if !strings.HasSuffix(gf.AbsPath, "conftest.py") {
		t.Errorf("AbsPath = %q, want suffix 'conftest.py'", gf.AbsPath)
	}
	if gf.Role != domain.RoleBootstrap {
		t.Errorf("Role = %q, want %q", gf.Role, domain.RoleBootstrap)
	}
	if !strings.Contains(gf.Content, "src/api") {
		t.Error("conftest content must include the derived source directory 'src/api'")
	}
}

// ── ValidateFile ──────────────────────────────────────────────────────────────

// backfill / AC: ValidateFile returns a warning when a pytest-mock file imports unittest.mock directly
func TestDriver_ValidateFile_WarnsMismatch(t *testing.T) {
	t.Parallel()
	d := python.New()
	content := "import pytest\nfrom unittest.mock import MagicMock\n\ndef test_foo():\n    pass\n"
	issues := d.ValidateFile("pytest", "pytest-mock", content)
	if len(issues) == 0 {
		t.Error("expected at least one issue for unittest.mock in pytest-mock context")
	}
}

// backfill / AC: ValidateFile returns error when a unittest.mock file uses mocker (pytest-mock fixture)
func TestDriver_ValidateFile_ErrorsOnMockerInUnittestMock(t *testing.T) {
	t.Parallel()
	d := python.New()
	content := "import pytest\n\ndef test_foo(mocker):\n    mocker.patch('mod.thing')\n"
	issues := d.ValidateFile("pytest", "unittest.mock", content)
	hasError := false
	for _, issue := range issues {
		if issue.Severity == domain.SeverityError {
			hasError = true
		}
	}
	if !hasError {
		t.Error("expected at least one error-severity issue when mocker is used in unittest.mock context")
	}
}

// backfill / AC: ValidateFile returns nil for an unknown (framework, mockLib) pair
func TestDriver_ValidateFile_UnknownPairReturnsNil(t *testing.T) {
	t.Parallel()
	d := python.New()
	issues := d.ValidateFile("nose2", "mock", "def test_x(): pass")
	if issues != nil {
		t.Errorf("ValidateFile with unknown pair should return nil, got %v", issues)
	}
}

// backfill / AC: ValidateFile returns no issues for a clean pytest+pytest-mock file
func TestDriver_ValidateFile_CleanFile(t *testing.T) {
	t.Parallel()
	d := python.New()
	content := "import pytest\n\ndef test_happy_path(mocker):\n    mocker.patch('mod.Foo')\n    assert True\n"
	issues := d.ValidateFile("pytest", "pytest-mock", content)
	if len(issues) != 0 {
		t.Errorf("expected no issues for clean pytest-mock file, got %v", issues)
	}
}

// backfill / AC: ValidateFile warns when a pytest test file contains no test_ functions
func TestDriver_ValidateFile_WarnsMissingTestFunctions(t *testing.T) {
	t.Parallel()
	d := python.New()
	content := "import pytest\n\ndef helper(): pass\n"
	issues := d.ValidateFile("pytest", "", content)
	if len(issues) == 0 {
		t.Error("expected a warning when no test_ functions are found in a pytest file")
	}
}
