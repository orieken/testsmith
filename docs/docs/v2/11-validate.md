# Feature: validate — Check Tests Against Adapter Conventions

## What It Does

`testsmith validate` scans existing test files and reports mismatches against the adapter that is currently configured (or auto-detected) for the project. It answers the question: *"Do my existing tests follow the conventions of the framework I'm supposed to be using?"*

Common issues detected:

- JUnit 4 imports or annotations in a JUnit 5 project (and vice versa)
- `jest.` API calls in a Vitest test file (and vice versa)
- `mocker.patch` usage in a file that should use `unittest.mock`
- Missing testify import when the `testify` adapter is selected

Each issue has a **severity**:
- `error` — definite incompatibility (wrong framework import, mismatched annotation)
- `warning` — possible issue (deprecated API, mixed patterns)
- `info` — advisory (using a different library than the default)

---

## CLI

```
testsmith validate [flags]

Flags:
  --lang <lang>       Override auto-detected language
  --path <dir>        Restrict validation to test files under this directory
  --workspace <name>  Validate only this workspace (name or path)
  --verbose, -v       Print ✓ for each clean file in addition to issues
```

Exit code is **1** when any `error`-severity issues are found (suitable for CI gating).

### Workspace Mode

When `workspaces:` are configured, all workspaces are validated and each prints a labelled section header. Use `--workspace <name>` to limit to one workspace.

---

## Example Output

```
$ testsmith validate

── workspace: api (services/api) ──
  ✗ src/test/java/PaymentTest.java
      [error  ] junit4-import-in-junit5: found 'import org.junit.Test' — use org.junit.jupiter.api.Test
      [error  ] junit4-runwith-in-junit5: found '@RunWith' — use @ExtendWith

── workspace: frontend (services/frontend) ──
  ✗ src/utils.test.ts
      [warning] jest-spy-in-vitest: found 'jest.' — use vi. for Vitest projects

Checked 6 file(s): 2 error(s), 1 warning(s) across 2 file(s)
```

Clean run:

```
$ testsmith validate --verbose
Language: go | Adapter: testing + interfaces

  ✓ internal/services/payment_test.go
  ✓ internal/services/user_test.go

Checked 2 file(s): 0 error(s), 0 warning(s) across 0 file(s)
```

---

## Go Implementation

### `ValidationIssue` and `Severity`

```go
// internal/domain/validator.go
type Severity string

const (
    SeverityError   Severity = "error"
    SeverityWarning Severity = "warning"
    SeverityInfo    Severity = "info"
)

type ValidationIssue struct {
    Rule     string
    Severity Severity
    Message  string
}
```

### `TextValidator` Builder

All built-in validators are constructed using the `validation.TextValidator` fluent builder:

```go
// internal/validation/text.go
func New() *TextValidator

// Require fires an issue when pattern is NOT found in the content.
func (v *TextValidator) Require(id, pattern, message string, sev domain.Severity) *TextValidator

// Forbid fires an issue when pattern IS found in the content.
func (v *TextValidator) Forbid(id, pattern, message string, sev domain.Severity) *TextValidator

// Validate runs all rules against content and returns issues found.
func (v *TextValidator) Validate(content string) []domain.ValidationIssue
```

Example — Java JUnit 5 validator:

```go
func junit5Validator() *validation.TextValidator {
    return validation.New().
        Forbid("junit4-import-in-junit5",
            `import org\.junit\.Test`,
            "found 'import org.junit.Test' — use org.junit.jupiter.api.Test",
            domain.SeverityError).
        Forbid("junit4-runwith-in-junit5",
            `@RunWith`,
            "found '@RunWith' — use @ExtendWith",
            domain.SeverityError)
}
```

### `ValidateFile` on `LanguageDriver`

Each driver implements:

```go
ValidateFile(framework, mockLib, content string) []domain.ValidationIssue
```

The driver selects the correct `TextValidator` based on the `framework` and `mockLib` arguments (which come from the selected `TestAdapter`) and calls `Validate(content)`.

### File Discovery

`validate` uses `discoverValidationFiles` (an alias of `discoverTestFiles` in `cmd/testsmith/testfiles.go`) — the same walk used by `migrate`.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/validate.go` | Cobra subcommand, workspace routing, output formatting |
| `cmd/testsmith/testfiles.go` | Shared `discoverValidationFiles` helper |
| `internal/domain/validator.go` | `ValidationIssue`, `Severity` constants |
| `internal/validation/text.go` | `TextValidator` fluent builder |
| `internal/drivers/<lang>/validators.go` | Per-language validator implementations |
