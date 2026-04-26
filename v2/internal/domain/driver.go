// Package domain contains the pure types and interfaces that form
// the core of TestSmith v2. Nothing in this package imports from
// the adapter or infrastructure layers.
package domain

import "context"

// LanguageDriver is the single extension point for adding language support.
// Implement this interface to teach TestSmith how to analyse and generate
// tests for a new language or test framework.
//
// All methods operate on abstract domain types. No language-specific
// types (Python AST nodes, Java PSI, etc.) ever cross this boundary.
type LanguageDriver interface {
	// Language returns the canonical lowercase identifier, e.g. "python", "typescript", "go", "java".
	Language() string

	// FileExtensions returns the source file suffixes this driver handles, e.g. [".py"], [".ts", ".tsx"].
	FileExtensions() []string

	// DetectProject walks up from dir looking for project-root markers and returns a
	// fully-populated ProjectContext. Returns ErrProjectNotFound when no root is detected.
	DetectProject(dir string) (*ProjectContext, error)

	// AnalyzeFile parses a single source file and returns the complete SourceAnalysis.
	// Implementations must NOT perform I/O beyond reading the given file path.
	AnalyzeFile(path string, ctx *ProjectContext) (*SourceAnalysis, error)

	// ClassifyDependency categorises one import as Stdlib, Internal, or External.
	ClassifyDependency(dep ImportInfo, ctx *ProjectContext) DependencyCategory

	// DeriveTestPath computes the destination path for the test file that corresponds
	// to the given source file, using the driver's framework conventions.
	DeriveTestPath(sourcePath string, ctx *ProjectContext) (string, error)

	// DeriveModulePath computes the importable module/package path for a source file,
	// e.g. "src/services/payment.py" -> "services.payment" (Python, src-layout).
	DeriveModulePath(sourcePath string, ctx *ProjectContext) (string, error)

	// GenerateTestFile produces a test scaffold for the given analysis.
	// The returned GeneratedFile contains the destination path and content.
	// Implementations must NOT write to disk.
	GenerateTestFile(analysis *SourceAnalysis, opts GenerateOpts) (*GeneratedFile, error)

	// GenerateFixture produces a mock/fake helper file for the given external dependency.
	// Returns nil, nil when the driver has no fixture concept (e.g. Go uses interfaces).
	GenerateFixture(dep string, analysis *SourceAnalysis, opts GenerateOpts) (*GeneratedFile, error)

	// GenerateBootstrap produces or updates the framework bootstrap file
	// (e.g. conftest.py for pytest, jest.setup.ts for Jest).
	// Returns nil, nil when no bootstrap file is needed.
	GenerateBootstrap(plan *GenerationPlan, ctx *ProjectContext) (*GeneratedFile, error)

	// GetTestFrameworkConfig returns static metadata about how this driver's
	// test framework names files, fixtures, and test functions.
	GetTestFrameworkConfig() TestFrameworkConfig

	// BodyGenerationPrompt returns a language/framework-specific prompt template
	// used by the LLM BodyGenerator adapter. The template receives a BodyPromptData value.
	BodyGenerationPrompt() string

	// LLMContext returns key-value pairs that are merged into the LLM prompt context,
	// allowing drivers to inject language-specific vocabulary (e.g. "assert" vs "expect").
	LLMContext() map[string]string
}

// BodyGenerator is the optional LLM adapter interface. The generation pipeline
// accepts nil and falls back to TODO stubs when no generator is wired in.
type BodyGenerator interface {
	GenerateBodies(ctx context.Context, req BodyGenRequest) ([]BodyGenResult, error)
}
