# Feature: migrate — Rewrite Test Files Between Frameworks

## What It Does

`testsmith migrate` bulk-rewrites existing test files from one framework convention to another using a set of ordered regex substitutions. It is the correct tool when a project is switching test frameworks and the existing test suite needs mechanical translation (e.g. upgrading from Jest to Vitest, or from JUnit 4 to JUnit 5).

Each language driver provides a list of `Migrator` implementations. A migrator knows its source framework (`From()`) and target framework (`To()`), and applies a sequence of regex substitutions to transform import statements, API calls, annotations, and mock syntax.

---

## CLI

```
testsmith migrate [flags]

Required flags:
  --from <framework>   Source framework (e.g. jest, junit4, pytest-mock, nunit)
  --to   <framework>   Target framework (e.g. vitest, junit5, unittest-mock, xunit)

Optional flags:
  --path <dir>         Restrict migration to test files under this directory
  --lang <lang>        Override auto-detected language
  --dry-run            Print what would change without writing files
  --verbose, -v        Show each rewritten file
```

If `--from` / `--to` do not match any registered migrator for the detected language, the command exits non-zero and lists the available pairs.

---

## Available Migration Pairs

| Language | `--from` | `--to` |
|----------|---------|--------|
| TypeScript | `jest` | `vitest` |
| TypeScript | `vitest` | `jest` |
| Python | `pytest-mock` | `unittest-mock` |
| Python | `unittest-mock` | `pytest-mock` |
| Java | `junit4` | `junit5` |
| Java | `junit5` | `junit4` |
| C# | `nunit` | `xunit` |
| C# | `xunit` | `nunit` |
| Go | — | — (AST-level rewrites not supported) |

---

## Example: Jest → Vitest

```
$ testsmith migrate --from jest --to vitest --path src/
  ✓ migrated  src/services/payment.test.ts
  ✓ migrated  src/utils/format.test.ts
  · skipped   src/api/client.test.ts  (no jest. references found)

Migrated 2 file(s), skipped 1 file(s).
```

Before:

```typescript
import { describe, test, expect, jest } from '@jest/globals';

describe('payment', () => {
  test('mocks spy', () => {
    const spy = jest.fn();
    jest.clearAllMocks();
    expect(spy).toHaveBeenCalled();
  });
});
```

After:

```typescript
import { vi } from 'vitest';
import { describe, test, expect } from 'vitest';

describe('payment', () => {
  test('mocks spy', () => {
    const spy = vi.fn();
    vi.clearAllMocks();
    expect(spy).toHaveBeenCalled();
  });
});
```

---

## Go Implementation

### `Migrator` Interface

```go
// internal/domain/migrator.go
type Migrator interface {
    From() string
    To() string
    MigrateFile(content string) (string, error)
}
```

### `TextMigrator` Builder

All built-in migrators are constructed using the `migration.TextMigrator` fluent builder:

```go
// internal/migration/text.go
func New(from, to string) *TextMigrator

// Add appends a regex substitution step. Pattern is a Go RE2 expression.
// Panics on bad pattern (compile-time literals only — never user input).
func (m *TextMigrator) Add(pattern, replacement string) *TextMigrator

// InjectImport prepends importLine before the first import block in the file.
func (m *TextMigrator) InjectImport(importLine string) *TextMigrator

// MigrateFile applies all steps in order and returns the transformed content.
func (m *TextMigrator) MigrateFile(content string) (string, error)
```

Example — `jestToVitest`:

```go
func jestToVitest() *migration.TextMigrator {
    return migration.New("jest", "vitest").
        Add(`import \{[^}]*\} from '@jest/globals'`, "").
        Add(`jest\.fn\(`, "vi.fn(").
        Add(`jest\.clearAllMocks\(`, "vi.clearAllMocks(").
        // ... additional substitutions
        InjectImport(`import { vi } from 'vitest';`)
}
```

### File Discovery

`migrate` uses `discoverTestFiles` (shared with `validate`) which walks the target directory and returns files whose base name contains `test` or `spec` with a language-appropriate extension.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/migrate.go` | Cobra subcommand, migrator lookup, file loop |
| `cmd/testsmith/testfiles.go` | Shared `discoverTestFiles` helper |
| `internal/domain/migrator.go` | `Migrator` interface |
| `internal/migration/text.go` | `TextMigrator` fluent builder |
| `internal/drivers/<lang>/migrators.go` | Per-language `Migrator` implementations |
