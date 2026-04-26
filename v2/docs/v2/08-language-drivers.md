# Feature: Language Drivers — Polyglot Support

## What It Does

A **language driver** is a self-contained implementation of the `domain.LanguageDriver` interface that teaches TestSmith how to work with a specific programming language and test framework. Adding a new language means adding one new driver package — no changes to the core pipelines.

---

## Auto-Detection

When no `--lang` flag is given, the `DriverRegistry` probes every registered driver's `DetectProject()` against the current working directory. The first driver that finds a root marker wins.

| Language | Winning Marker | Test Framework |
|----------|---------------|----------------|
| Python | `pyproject.toml`, `setup.py`, `setup.cfg` | pytest |
| TypeScript/JS | `package.json` | Jest / Vitest (auto-detected) |
| Go | `go.mod` | `testing` (stdlib) |
| Java | `pom.xml`, `build.gradle` | JUnit 5 |
| C# | `*.sln`, `*.csproj` | xUnit (default), NUnit, MSTest |

If multiple drivers match (e.g. a monorepo with both `package.json` and `go.mod`), the most specific marker wins, or the user specifies `--lang`.

---

## Python Driver (`internal/drivers/python/`)

| Concern | Approach |
|---------|----------|
| Project root | Walk up for `pyproject.toml` → `setup.py` → `setup.cfg` → `.git` → `conftest.py` |
| Package map | Scan for `__init__.py`; strip `src/` prefix for src-layout projects |
| Import parsing | go-tree-sitter + Python grammar; `import_statement` + `import_from_statement` nodes |
| Stdlib detection | Embedded Go map of Python 3.11 stdlib names (600+ modules) |
| Public API | tree-sitter: `function_definition` at module scope, `class_definition` + methods |
| Test path | `{test_root}/{rel_source}/test_{stem}.py` |
| Test framework | pytest — class-based `TestFoo`, method-based `test_*` |
| Fixture strategy | `mocker.patch.dict("sys.modules", {...})` in `tests/fixtures/{dep}_fixture.py` |
| Bootstrap file | `conftest.py` with `pytest_configure` hook and `paths_to_add` |

---

## TypeScript Driver (`internal/drivers/typescript/`)

| Concern | Approach |
|---------|----------|
| Project root | `package.json` → `tsconfig.json` → `.git` |
| Package map | Parse `package.json` `name` and `paths` in `tsconfig.json` for internal aliases |
| Import parsing | go-tree-sitter + TypeScript grammar; `import_statement`, `export_statement` |
| Stdlib detection | Embedded list of Node.js built-in module names (60+ entries) + `node:` prefix stripping |
| Public API | Exported `function_declaration`, `class_declaration`, `interface_declaration` |
| Test path | Adjacent `.test.ts` file or `__tests__/{stem}.test.ts` (detected from project convention) |
| Test framework | Jest (default) or Vitest (detected from `package.json` devDependencies) |
| Fixture strategy | `jest.mock(...)` / `vi.mock(...)` calls in `__mocks__/` directory |
| Bootstrap file | `jest.setup.ts` / `vitest.setup.ts` |

---

## Go Driver (`internal/drivers/golang/`)

| Concern | Approach |
|---------|----------|
| Project root | `go.mod` → `.git` |
| Package map | Parse `module` directive from `go.mod` as the internal prefix |
| Import parsing | Native `go/ast` + `go/parser` — no tree-sitter needed |
| Stdlib detection | Embedded list of Go 1.22 stdlib package paths (100+ entries) |
| Public API | Exported `FuncDecl` (capital first letter) and `TypeSpec` with `StructType` / `InterfaceType` |
| Test path | `{source_stem}_test.go` co-located in the same directory and package |
| Test framework | `testing` package — `func TestXxx(t *testing.T)`, table-driven pattern |
| Fixture strategy | Interface-based mocks; optionally generate `testify/mock` stubs |
| Bootstrap file | None (Go test setup via `TestMain` if needed) |

---

## Java Driver (`internal/drivers/java/`)

| Concern | Approach |
|---------|----------|
| Project root | `pom.xml` → `build.gradle` → `.git` |
| Package map | Parse `groupId`/`artifactId` from `pom.xml`; infer from Gradle `settings.gradle` |
| Import parsing | go-tree-sitter + Java grammar; `import_declaration` nodes |
| Stdlib detection | Embedded map of `java.*`, `javax.*`, `jakarta.*`, `sun.*` prefixes |
| Public API | `class_declaration`, `interface_declaration` with `public` modifier; `method_declaration` |
| Test path | `src/test/java/{package}/Test{SourceName}.java` |
| Test framework | JUnit 5 — `@Test`, `@ExtendWith(MockitoExtension.class)` |
| Fixture strategy | Mockito `@Mock` fields + `when(...).thenReturn(...)` |
| Bootstrap file | None standard; JUnit base class generated if needed |

---

## Go Implementation: DriverRegistry

```go
// internal/registry/registry.go
type DriverRegistry struct {
    drivers []domain.LanguageDriver
}

func (r *DriverRegistry) Register(d domain.LanguageDriver)

// Detect returns the first driver whose DetectProject succeeds, or ErrNoDriverForFile.
func (r *DriverRegistry) Detect(dir string) (domain.LanguageDriver, *domain.ProjectContext, error)

// ForLanguage returns the driver registered for the given language name.
func (r *DriverRegistry) ForLanguage(lang string) (domain.LanguageDriver, error)

// ForFile returns the driver that claims a given file extension.
func (r *DriverRegistry) ForFile(path string) (domain.LanguageDriver, error)
```

Drivers are registered in `cmd/testsmith/main.go` composition root:

```go
reg := registry.New()
reg.Register(python.New())
reg.Register(typescript.New())
reg.Register(golang.New())
reg.Register(java.New())
reg.Register(csharp.New())
```

---

## C# Driver (`internal/drivers/csharp/`)

| Concern | Approach |
|---------|----------|
| Project root | Walk up for `*.sln` → `*.csproj` → `.git` |
| Package map | Parse `<RootNamespace>` and `<AssemblyName>` from `.csproj`; infer namespace from directory structure |
| Import parsing | go-tree-sitter + C# grammar; `using_directive` nodes |
| Stdlib detection | Embedded prefix set: `System`, `Microsoft`, `Windows`, `mscorlib` |
| Public API | `class_declaration`, `interface_declaration`, `method_declaration` with `public` modifier |
| Test path | Separate `*.Tests` project convention: `{ProjectName}.Tests/{rel_source}/{Name}Tests.cs` |
| Test framework | xUnit (default); NUnit or MSTest if detected in `.csproj` `<PackageReference>` |
| Fixture strategy | Moq constructor injection — `Mock<IFoo>` declared per test class; no shared fixture files |
| Bootstrap file | None — xUnit test discovery is assembly-based; no setup file required |
| Test naming | `{Name}Tests.cs`, `[Fact]` single tests, `[Theory]` + `[InlineData]` for parameterised |
| Mock strategy | `new Mock<IDependency>()`, `.Setup(x => x.Method()).Returns(value)`, `.Object` for SUT construction |

### C# Test Structure (xUnit example)

```csharp
public class PaymentProcessorTests
{
    private readonly Mock<IStripeClient> _mockStripe;
    private readonly PaymentProcessor _sut;

    public PaymentProcessorTests()
    {
        _mockStripe = new Mock<IStripeClient>();
        _sut = new PaymentProcessor(_mockStripe.Object);
    }

    [Fact]
    public async Task Charge_WithValidToken_ReturnsSucceeded()
    {
        // Arrange
        _mockStripe.Setup(x => x.CreateChargeAsync(It.IsAny<ChargeRequest>()))
                   .ReturnsAsync(new ChargeResult { Status = "succeeded" });
        // Act
        var result = await _sut.ChargeAsync(amount: 1000, currency: "usd", token: "tok_test");
        // Assert
        Assert.Equal("succeeded", result.Status);
    }

    [Theory]
    [InlineData(0)]
    [InlineData(-1)]
    public async Task Charge_WithInvalidAmount_ThrowsArgumentException(int amount)
    {
        // Act & Assert
        await Assert.ThrowsAsync<ArgumentException>(() =>
            _sut.ChargeAsync(amount, "usd", "tok_test"));
    }
}
```

### Project Layout Convention

C# projects typically separate source and test into sibling projects:

```
MyApp/
├── MyApp.sln
├── MyApp/                        ← source project
│   ├── MyApp.csproj
│   └── Services/
│       └── PaymentProcessor.cs
└── MyApp.Tests/                  ← test project (generated by testsmith init)
    ├── MyApp.Tests.csproj        ← references MyApp + xUnit + Moq
    └── Services/
        └── PaymentProcessorTests.cs
```

`testsmith init` creates the `*.Tests` project and wires the `<ProjectReference>` in the test `.csproj`.

---

## Adding a New Language Driver

1. Create `internal/drivers/<lang>/driver.go` implementing `domain.LanguageDriver`.
2. Add tree-sitter grammar queries under `internal/drivers/<lang>/queries/*.scm`.
3. Add prompt template under `internal/drivers/<lang>/prompts/generate_body.tmpl`.
4. Register the driver in `cmd/testsmith/main.go`.
5. Add testdata samples under `testdata/<lang>/`.

No other files need to change.
