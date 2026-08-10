# Framework & Mock Library Matrix

Assay ships 18 built-in adapters across 5 languages. Each adapter is independently selectable via config or auto-detected from project files.

## Built-in adapters

| Language   | Framework    | Mock Library    | Key / config value            | Default |
|------------|--------------|-----------------|-------------------------------|---------|
| Python     | pytest       | pytest-mock     | `pytest` / `pytest-mock`      | ✓       |
| Python     | pytest       | unittest.mock   | `pytest` / `unittest.mock`    |         |
| Python     | unittest     | unittest.mock   | `unittest` / `unittest.mock`  |         |
| TypeScript | jest         | jest            | `jest` / `jest`               | ✓       |
| TypeScript | vitest       | vitest          | `vitest` / `vitest`           |         |
| TypeScript | mocha        | sinon           | `mocha` / `sinon`             |         |
| Go         | testing      | interfaces      | `testing` / `interfaces`      | ✓       |
| Go         | testing      | testify         | `testing` / `testify`         |         |
| Go         | testing      | gomock          | `testing` / `gomock`          |         |
| Java       | junit5       | mockito         | `junit5` / `mockito`          | ✓       |
| Java       | junit4       | mockito         | `junit4` / `mockito`          |         |
| Java       | testng       | mockito         | `testng` / `mockito`          |         |
| Java       | springboot   | mockito         | `springboot` / `mockito`      |         |
| C#         | xunit        | moq             | `xunit` / `moq`               | ✓       |
| C#         | nunit        | moq             | `nunit` / `moq`               |         |
| C#         | mstest       | moq             | `mstest` / `moq`              |         |
| C#         | xunit        | nsubstitute     | `xunit` / `nsubstitute`       |         |
| C#         | nunit        | nsubstitute     | `nunit` / `nsubstitute`       |         |

## Auto-detection heuristics

Assay reads project files to select the best adapter automatically:

| Language   | Signal file(s)                   | Detected as               |
|------------|----------------------------------|---------------------------|
| Python     | `pytest-mock` in deps            | pytest + pytest-mock      |
| Python     | `pytest` only (no pytest-mock)   | pytest + unittest.mock    |
| Python     | `unittest` only, no pytest       | unittest + unittest.mock  |
| TypeScript | `vitest` in package.json         | vitest + vitest           |
| TypeScript | `mocha` + `sinon` in package.json| mocha + sinon             |
| Go         | `go.uber.org/mock` in go.mod     | testing + gomock          |
| Go         | `github.com/stretchr/testify`    | testing + testify         |
| Java       | `spring-boot-starter-test`       | springboot + mockito      |
| Java       | `testng` in pom.xml/build.gradle | testng + mockito          |
| Java       | `junit-vintage-engine` or junit4 | junit4 + mockito          |
| C#         | `NUnit` in .csproj               | nunit + moq               |
| C#         | `MSTest.TestFramework`           | mstest + moq              |
| C#         | `NSubstitute` in .csproj         | (same framework) + nsubstitute |

Config overrides always win over auto-detection — see `.assay.yaml` configuration below.

## Overriding via config

```yaml
# .assay.yaml
languages:
  python:
    framework: unittest
    mock_library: unittest.mock
  typescript:
    framework: vitest
    mock_library: vitest
  go:
    framework: testing
    mock_library: gomock
  java:
    framework: junit5
    mock_library: mockito
  csharp:
    framework: nunit
    mock_library: nsubstitute
```

## Contributing a new adapter

### 1. Implement the interface

```go
// internal/drivers/<language>/adapters.go

type myFrameworkAdapter struct{}

func (a *myFrameworkAdapter) Framework() string   { return "myframework" }
func (a *myFrameworkAdapter) MockLibrary() string { return "mylib" }

func (a *myFrameworkAdapter) FrameworkConfig() domain.TestFrameworkConfig {
    return domain.TestFrameworkConfig{
        Name:           "myframework",
        TestFileSuffix: "_mytest.go", // language-specific convention
        TestFuncPrefix: "TestWith",
    }
}

func (a *myFrameworkAdapter) GenerateTestFile(
    analysis *domain.SourceAnalysis,
    opts domain.GenerateOpts,
) (string, error) {
    return renderTmpl(myFrameworkTmpl, newRenderData(analysis, opts))
}

var myFrameworkTmpl = template.Must(template.New("myframework").Funcs(sharedFuncs).Parse(`
// your template here — use {{ .Members }}, {{ $.PublicMethods . }}, {{ $.BodyFor .Name }}
`))
```

### 2. Register it

In the same file's `buildRegistry()`:

```go
func buildRegistry() *domain.AdapterRegistry {
    r := domain.NewAdapterRegistry()
    r.SetDefault(&existingDefaultAdapter{})
    r.Register(&myFrameworkAdapter{}) // ← add this
    return r
}
```

### 3. Add auto-detection (optional)

In the driver's `detector.go`, extend the detection function to recognise the new framework's dependency:

```go
// e.g. for Go
case strings.Contains(content, "github.com/myorg/myframework"):
    return "myframework"
```

### 4. Write tests

Create or extend `adapters_test.go`:

```go
func TestMyFrameworkAdapter_Imports(t *testing.T) {
    a := &myFrameworkAdapter{}
    out, err := a.GenerateTestFile(makeAnalysis("function"), domain.GenerateOpts{})
    if err != nil {
        t.Fatal(err)
    }
    if !strings.Contains(out, "myframework") {
        t.Error("missing framework import")
    }
}
```

### Template helpers available

| Helper         | Description                                          |
|----------------|------------------------------------------------------|
| `$.PublicMethods .` | Returns only `IsPublic == true` methods for a member |
| `$.BodyFor .Name`   | Returns LLM-generated lines for a method, or nil    |
| `indent $body N`    | Indents each line with N tabs (Go) or N spaces (others) |
| `title .Name`       | Title-cases the first letter (Go/Java only)         |
| `join slice sep`    | Joins a string slice (TypeScript only)              |

### Naming conventions

- `Framework()` — lowercase, no spaces: `"jest"`, `"junit5"`, `"xunit"`, `"pytest"`
- `MockLibrary()` — lowercase, hyphenated where conventional: `"pytest-mock"`, `"unittest.mock"`, `"nsubstitute"`
- Config keys map 1:1 to these return values

### Registry lookup

`AdapterRegistry.Select(framework, mockLibrary)` uses this precedence:

1. Exact match on `"framework/mockLibrary"` key
2. Framework-only match (first registered adapter for that framework)
3. Default adapter (set via `SetDefault`)
