# Feature: Language Drivers — Polyglot Support

## What It Does

A **language driver** is a self-contained implementation of the `domain.LanguageDriver` interface that teaches TestSmith how to work with a specific programming language and test framework. Adding a new language means adding one new driver package — no changes to the core pipelines.

---

## Auto-Detection

When no `--lang` flag is given, the `DriverRegistry` probes every registered driver's `DetectProject()` against the current working directory. The first driver that finds a root marker wins.

| Language | Winning Marker | Default Test Framework |
|----------|---------------|----------------------|
| Python | `pyproject.toml`, `setup.py`, `setup.cfg` | pytest |
| TypeScript/JS | `package.json` | Jest / Vitest (auto-detected from devDependencies) |
| Go | `go.mod` | `testing` (stdlib) |
| Java | `pom.xml`, `build.gradle` | JUnit 5 |
| C# | `*.sln`, `*.csproj` | xUnit (default), NUnit, MSTest |

If multiple drivers match (e.g. a monorepo with both `package.json` and `go.mod`), configure explicit `workspaces:` in `.testsmith.yaml` or use `--lang`.

---

## `LanguageDriver` Interface

```go
// internal/domain/driver.go
type LanguageDriver interface {
    Language() string
    FileExtensions() []string
    DetectProject(dir string) (*ProjectContext, error)
    AnalyzeFile(path string, ctx *ProjectContext) (*SourceAnalysis, error)
    DeriveTestPath(srcPath string, ctx *ProjectContext) (string, error)
    GenerateTestFile(a *SourceAnalysis, bodies map[string][]string) (string, error)
    GenerateFixture(dep ImportInfo, ctx *ProjectContext) (string, error)
    GenerateBootstrap(ctx *ProjectContext) (string, error)
    BootstrapPath(ctx *ProjectContext) string
    ListAdapters(ctx *ProjectContext) ([]TestAdapter, TestAdapter)
    GetTestFrameworkConfig() TestFrameworkConfig
    // Migration support
    ListMigrators() []Migrator
    // Validation support
    ValidateFile(framework, mockLib, content string) []ValidationIssue
}
```

### `ListMigrators`

Returns the set of `Migrator` implementations the driver provides. Each migrator rewrites test file content from one framework convention to another using a fluent `TextMigrator` regex pipeline.

```go
// internal/domain/migrator.go
type Migrator interface {
    From() string                                     // e.g. "jest"
    To() string                                       // e.g. "vitest"
    MigrateFile(content string) (string, error)
}
```

### `ValidateFile`

Checks the content of a test file against the conventions of the given framework and mock library. Returns a slice of `ValidationIssue` values (empty = clean).

```go
// internal/domain/validator.go
type ValidationIssue struct {
    Rule     string
    Severity Severity   // "error" | "warning" | "info"
    Message  string
}
```

---

## Migrators by Language

| Language | Available Pairs |
|----------|----------------|
| Python | `pytest-mock` → `unittest.mock`, and reverse |
| TypeScript | `jest` → `vitest`, and reverse |
| Go | None (AST-level rewrites are too complex for regex) |
| Java | `junit4` → `junit5`, and reverse |
| C# | `nunit` → `xunit`, and reverse |

Migrators are built with the `migration.TextMigrator` fluent builder:

```go
// internal/migration/text.go
type TextMigrator struct { ... }

func (m *TextMigrator) Add(pattern, replacement string) *TextMigrator
func (m *TextMigrator) InjectImport(importLine string) *TextMigrator
func (m *TextMigrator) MigrateFile(content string) (string, error)
```

---

## Validators by Language

| Language | Rules enforced |
|----------|---------------|
| Python | pytest-mock API present/absent, `def test_` prefix, unittest.mock imports |
| TypeScript | `vi.` vs `jest.` API, `@jest/globals` vs `vitest` imports |
| Go | testify import required/forbidden based on selected adapter |
| Java | JUnit 4 vs JUnit 5 import and annotation mismatches, TestNG detection |
| C# | xUnit, NUnit, MSTest attribute and namespace consistency |

Validators are built with the `validation.TextValidator` fluent builder:

```go
// internal/validation/text.go
type TextValidator struct { ... }

func (v *TextValidator) Require(id, pattern, message string, sev domain.Severity) *TextValidator
func (v *TextValidator) Forbid(id, pattern, message string, sev domain.Severity) *TextValidator
func (v *TextValidator) Validate(content string) []domain.ValidationIssue
```

---

## Python Driver (`internal/drivers/python/`)

| Concern | Approach |
|---------|----------|
| Project root | Walk up for `pyproject.toml` → `setup.py` → `setup.cfg` → `.git` |
| Import parsing | go-tree-sitter + Python grammar |
| Public API | `function_definition` at module scope, `class_definition` + methods |
| Test path | `{test_root}/{rel_source}/test_{stem}.py` |
| Test framework | pytest — `TestFoo` classes, `test_*` methods |
| Fixture strategy | `mocker.patch.dict("sys.modules", {...})` in `tests/fixtures/{dep}_fixture.py` |

---

## TypeScript Driver (`internal/drivers/typescript/`)

| Concern | Approach |
|---------|----------|
| Project root | `package.json` → `tsconfig.json` → `.git` |
| Import parsing | go-tree-sitter + TypeScript grammar |
| Public API | Exported `function_declaration`, `class_declaration`, `interface_declaration` |
| Test path | `{stem}.test.ts` adjacent or in `__tests__/` |
| Test framework | Jest (default) or Vitest (detected from `package.json` devDependencies) |
| Fixture strategy | `jest.mock(...)` / `vi.mock(...)` in `__mocks__/` |

---

## Go Driver (`internal/drivers/golang/`)

| Concern | Approach |
|---------|----------|
| Project root | `go.mod` → `.git` |
| Import parsing | Native `go/ast` + `go/parser` — no tree-sitter |
| Public API | Exported `FuncDecl` and `TypeSpec` |
| Test path | `{source_stem}_test.go` co-located, same package |
| Test framework | `testing` package — table-driven `TestXxx(t *testing.T)` |
| Fixture strategy | Interface mocks; optionally `testify/mock` stubs |

---

## Java Driver (`internal/drivers/java/`)

| Concern | Approach |
|---------|----------|
| Project root | `pom.xml` → `build.gradle` → `.git` |
| Import parsing | go-tree-sitter + Java grammar |
| Public API | `public` `class_declaration`, `interface_declaration`, `method_declaration` |
| Test path | `src/test/java/{package}/Test{SourceName}.java` |
| Test framework | JUnit 5 — `@Test`, `@ExtendWith(MockitoExtension.class)` |
| Fixture strategy | Mockito `@Mock` fields + `when(...).thenReturn(...)` |

---

## C# Driver (`internal/drivers/csharp/`)

| Concern | Approach |
|---------|----------|
| Project root | Walk up for `*.sln` → `*.csproj` → `.git` |
| Import parsing | go-tree-sitter + C# grammar; `using_directive` nodes |
| Public API | `public` `class_declaration`, `interface_declaration`, `method_declaration` |
| Test path | `{ProjectName}.Tests/{rel_source}/{Name}Tests.cs` |
| Test framework | xUnit (default); NUnit or MSTest if detected in `.csproj` |
| Fixture strategy | Moq constructor injection — `Mock<IFoo>` per test class |

---

## DriverRegistry

```go
// internal/registry/registry.go
type Registry struct { ... }

func New() *Registry
func (r *Registry) Register(d domain.LanguageDriver)
func (r *Registry) Detect(dir string) (domain.LanguageDriver, *domain.ProjectContext, error)
func (r *Registry) ForLanguage(lang string) (domain.LanguageDriver, error)
```

Drivers are registered in `cmd/testsmith/main.go`:

```go
reg = registry.New()
reg.Register(python.New())
reg.Register(typescript.New())
reg.Register(golang.New())
reg.Register(java.New())
reg.Register(csharp.New())
```

---

## Adding a New Language Driver

1. Create `internal/drivers/<lang>/driver.go` implementing `domain.LanguageDriver`.
2. Implement `ListMigrators()` — return `nil` if no regex-level migrations make sense.
3. Implement `ValidateFile()` — use `validation.TextValidator` builders for each framework variant.
4. Add tree-sitter grammar queries under `internal/drivers/<lang>/queries/*.scm` (if applicable).
5. Register the driver in `cmd/testsmith/main.go`.
6. Add source fixtures under `testdata/<lang>/` for integration tests.

No other files need to change.
