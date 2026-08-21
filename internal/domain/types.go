package domain

import "path/filepath"

// ---- Project ----------------------------------------------------------------

// ProjectContext is the language-agnostic representation of a detected project.
// Drivers populate this; pipelines and the CLI consume it.
type ProjectContext struct {
	Root        string            // absolute path to project root
	Language    string            // canonical language identifier from the winning driver
	PackageMap  map[string]string // root-package-name -> absolute directory path
	ExcludeDirs []string          // directory names skipped during file scanning
	// Metadata holds driver-specific extras that don't fit the common fields,
	// e.g. Python: conftest_path, existing_paths; Go: module_name from go.mod.
	Metadata map[string]any
	// ProjectKnowledge is the content of the project's ASSAY.md file (if present).
	// Loaded once after DetectProject and injected into every LLM system prompt.
	ProjectKnowledge string
}

// ---- Analysis ---------------------------------------------------------------

// SourceAnalysis is the language-agnostic result of parsing one source file.
type SourceAnalysis struct {
	SourcePath string
	ModulePath string // importable path, e.g. "services.payment" or "services/payment"
	Imports    ClassifiedImports
	PublicAPI  []PublicMember
	Project    *ProjectContext
	RawSource  string // full source text, available for LLM prompts
}

// ModuleName returns the stem of the source file (e.g. "payment" from "payment.py").
func (a *SourceAnalysis) ModuleName() string {
	base := filepath.Base(a.SourcePath)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}

// ModuleName returns per-language config values stored in ProjectContext.Metadata["lang_config"].
func (p *ProjectContext) LanguageConfig() map[string]string {
	if v, ok := p.Metadata["lang_config"]; ok {
		if m, ok := v.(map[string]string); ok {
			return m
		}
	}
	return map[string]string{}
}

// ClassifiedImports groups ImportInfo values by resolved category.
type ClassifiedImports struct {
	Stdlib   []ImportInfo
	Internal []ImportInfo
	External []ImportInfo
}

// ImportInfo represents a single import statement in a language-agnostic form.
type ImportInfo struct {
	Module     string   // dotted/slashed module path as written in the source
	Names      []string // specific names imported ("from x import y, z" -> ["y","z"])
	IsFrom     bool     // true for "from x import y" / "import { y } from x" style
	Alias      string   // "import x as y" -> alias is "y"
	LineNumber int
}

// DependencyCategory classifies an import.
type DependencyCategory int

const (
	DepStdlib   DependencyCategory = iota // language built-in / standard library
	DepInternal                           // within the same project
	DepExternal                           // third-party package
)

// PublicMember is a language-agnostic exported function, class, struct, or interface.
type PublicMember struct {
	Name       string
	Kind       MemberKind
	Parameters []ParamInfo
	Methods    []MethodInfo // populated for Class / Struct / Interface kinds
	Docstring  string
}

// MemberKind classifies what a PublicMember represents.
type MemberKind string

const (
	KindFunction  MemberKind = "function"
	KindClass     MemberKind = "class"
	KindInterface MemberKind = "interface"
	KindStruct    MemberKind = "struct"
)

// ParamInfo describes a single parameter in a function or method signature.
type ParamInfo struct {
	Name     string
	TypeHint string // empty when the language lacks type annotations
}

// MethodInfo describes one method on a class, struct, or interface.
type MethodInfo struct {
	Name       string
	Parameters []ParamInfo
	IsPublic   bool
}

// ---- Generation -------------------------------------------------------------

// GenerationPlan is the complete, ordered set of file operations to perform.
// The plan is pure data — it is computed before any I/O, which makes dry-run
// a zero-cost in-memory operation and keeps the pipeline fully unit-testable.
type GenerationPlan struct {
	Files  []GeneratedFile
	DryRun bool
}

// GeneratedFile is one file that will be created or updated.
type GeneratedFile struct {
	AbsPath  string
	Content  string
	Action   FileAction
	Role     FileRole
	Language string // set by Plan(); used by Executor to select the right Verifier
}

// FileAction describes what will happen to the file on disk.
type FileAction string

const (
	ActionCreate FileAction = "created"
	ActionUpdate FileAction = "updated"
	ActionSkip   FileAction = "skipped"
)

// FileRole classifies the purpose of a generated file.
type FileRole string

const (
	RoleTestFile  FileRole = "test"
	RoleFixture   FileRole = "fixture"
	RoleBootstrap FileRole = "bootstrap" // conftest.py, jest.setup.ts, etc.
	RoleInit      FileRole = "init"      // __init__.py, package stubs
)

// GenerateOpts carries per-invocation options passed down to driver generation methods.
type GenerateOpts struct {
	DryRun            bool
	LLMBodies         map[string][]string // member name -> generated code lines
	FixtureImports    []FixtureImport
	OverwriteExisting bool
	// TestFileKnownNew skips the os.Stat existence check in resolveAction when the
	// caller already knows the test file does not exist (e.g. after DiscoverUntested).
	TestFileKnownNew bool
}

// FixtureImport describes a fixture function that a test file should receive.
type FixtureImport struct {
	Module   string
	FuncName string
}

// TestFrameworkConfig carries static metadata about a driver's test framework.
type TestFrameworkConfig struct {
	Name           string // "pytest", "jest", "vitest", "testing", "junit"
	TestFilePrefix string // "test_" (Python); empty for Go (uses suffix)
	TestFileSuffix string // ".py", "_test.go", ".test.ts"
	FixtureDir     string // "tests/fixtures", "__mocks__", "" (Go: co-located)
	FixtureSuffix  string // "_fixture.py", ".mock.ts", ""
	BootstrapFile  string // "conftest.py", "jest.setup.ts", ""
	TestFuncPrefix string // "test_", "Test", "it("
}

// ---- LLM --------------------------------------------------------------------

// BodyGenRequest is the input to a BodyGenerator call for a single member.
type BodyGenRequest struct {
	Language            string
	MemberName          string
	MemberKind          MemberKind
	SourceCode          string // full source file for context window
	Fixtures            []FixtureImport
	Framework           TestFrameworkConfig
	Extra               map[string]string // driver-injected language vocabulary
	ModulePath          string            // importable path of the source module
	DepsSignatures      string            // compact public API block of internal deps
	ExistingTestSnippet string            // style sample mined from existing test files in same package
	ProjectKnowledge    string            // content of ASSAY.md, injected as system prompt prefix
}

// BodyGenResult is the LLM output for one member.
type BodyGenResult struct {
	MemberName string
	CodeLines  []string
	TokensUsed int
}

// BodyPromptData is injected into a driver's BodyGenerationPrompt template.
type BodyPromptData struct {
	MemberName          string
	MemberKind          string
	SourceCode          string
	FixtureNames        []string
	Extra               map[string]string
	ModulePath          string
	DepsSignatures      string
	ExistingTestSnippet string
	ProjectKnowledge    string
}

// ---- Graph ------------------------------------------------------------------

// DependencyGraph is a directed graph of inter-module dependencies.
type DependencyGraph struct {
	Nodes []GraphNode
	Edges []GraphEdge
}

// GraphNode represents a single source module in the dependency graph.
type GraphNode struct {
	Name             string
	Path             string
	Package          string
	ExternalDepCount int
}

// GraphEdge is a directed dependency between two modules.
type GraphEdge struct {
	Source   string
	Target   string
	EdgeType string // "internal" or "external"
}

// ModuleMetrics holds computed coupling metrics for a single module.
type ModuleMetrics struct {
	Name                 string
	InternalDependencies int
	ExternalDependencies int
	Dependents           int
	CouplingScore        float64
}

// ---- Coverage ---------------------------------------------------------------

// CoverageStatus describes how well a source file is covered by tests.
type CoverageStatus string

const (
	CoverageNoTest       CoverageStatus = "no_test"
	CoverageSkeletonOnly CoverageStatus = "skeleton_only"
	CoveragePartial      CoverageStatus = "partial"
	CoverageFull         CoverageStatus = "covered"
)

// CoverageGap represents an untested or under-tested source file.
type CoverageGap struct {
	SourcePath       string
	Status           CoverageStatus
	PriorityScore    float64
	ExternalDeps     int
	Dependents       int
	SuggestedCommand string
}
