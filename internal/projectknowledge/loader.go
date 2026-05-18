// Package projectknowledge reads and merges TESTSMITH.md knowledge files from
// the target project. These files let project owners inject conventions, domain
// vocabulary, and test-infrastructure facts into every LLM prompt without
// reloading them per-file.
//
// Loading is hierarchical:
//
//	<project-root>/TESTSMITH.md       — project-wide conventions (always loaded)
//	<source-dir>/TESTSMITH.md         — package-level overrides (merged below root)
//
// If neither file exists, Load/LoadForFile return an empty string and the LLM
// falls back to file-level context only.
package projectknowledge

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

const filename = "TESTSMITH.md"

// Load reads the root-level TESTSMITH.md for the given project root.
// Returns empty string when the file does not exist.
func Load(root string) string {
	return readFile(filepath.Join(root, filename))
}

// LoadForFile merges root-level and source-directory-level TESTSMITH.md files.
// The directory content is appended under a separator so the LLM sees both.
// Returns the root content alone when source directory has no override.
func LoadForFile(sourcePath, root string) string {
	return LoadForDir(filepath.Dir(sourcePath), root)
}

// LoadForDir merges root-level and directory-level TESTSMITH.md files.
// Use this when you have a directory path rather than a file path.
func LoadForDir(dir, root string) string {
	rootContent := readFile(filepath.Join(root, filename))

	if dir == root {
		return rootContent
	}

	dirContent := readFile(filepath.Join(dir, filename))
	if dirContent == "" {
		return rootContent
	}

	if rootContent == "" {
		return dirContent
	}

	var sb strings.Builder
	sb.WriteString(rootContent)
	sb.WriteString("\n\n---\n\n## Package-level conventions\n\n")
	sb.WriteString(dirContent)
	return sb.String()
}

// Template returns a language-specific starter TESTSMITH.md that testsmith init
// can write to a new project. The caller should trim or expand it to taste.
func Template(language string) string {
	base := templates["default"]
	if t, ok := templates[language]; ok {
		base = t
	}
	return base
}

func readFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ""
		}
		return "" // unreadable — treat as absent
	}
	return strings.TrimSpace(string(data))
}

// templates holds per-language starter content.
var templates = map[string]string{
	"default": `# Project Test Conventions

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
<!-- How are tests named? What structure do they follow? -->
<!-- Example: Test<Method>_<scenario>, e.g. TestProcessPayment_insufficientFunds -->

## Test infrastructure
<!-- List shared helpers, fixtures, or setup utilities available in tests. -->
<!-- Example: internal/testkit — Setup(t), NewUser(), NewPayment() -->

## Domain vocabulary
<!-- Define key domain terms so the LLM generates accurate test scenarios. -->
<!-- Example: -->
<!-- - Order: a customer's purchase request, may be pending/confirmed/cancelled -->
<!-- - Invoice: generated after an order is confirmed; immutable once issued -->

## Do not mock
<!-- List internal packages or types that should be used directly, not mocked. -->
<!-- Example: domain/ entities, internal/money value objects -->

## External dependencies
<!-- Note any third-party libraries the LLM should use in generated tests. -->
`,

	"go": `# Project Test Conventions (Go)

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
- Naming: Test<Function>_<scenario>, e.g. TestProcessPayment_insufficientFunds
- Table-driven tests preferred for multiple scenarios
- Use t.Parallel() in unit tests where safe
<!-- Add: testify/suite, plain testing, or other patterns used here -->

## Test infrastructure
<!-- Example: -->
<!-- - internal/testkit — Setup(t *testing.T), NewUser(), NewOrder() -->
<!-- - internal/testkit/db — OpenTestDB(t) returns *sql.DB against testcontainers -->

## Build tags
<!-- Example: //go:build integration — used for tests requiring external services -->

## Domain vocabulary
<!-- Define key domain terms. -->

## Do not mock
<!-- Example: domain/ entities (use real structs), internal/money (pure value objects) -->

## Mock library
<!-- testify/mock, gomock, or interfaces — matches what is already in go.mod -->
`,

	"python": `# Project Test Conventions (Python)

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
- Framework: pytest
<!-- Add: fixture scope, naming conventions, parametrize patterns -->
<!-- Example: test_<function>_<scenario>, e.g. test_process_payment_insufficient_funds -->

## Test infrastructure
<!-- Example: -->
<!-- - tests/conftest.py — shared fixtures: db_session, test_client, mock_email -->
<!-- - tests/factories/ — factory_boy factories for domain objects -->

## Domain vocabulary
<!-- Define key domain terms. -->

## Do not mock
<!-- Example: domain.models — use real objects; settings — use pytest-django settings -->

## Mock style
<!-- unittest.mock.patch, pytest-mock mocker.patch, or other -->
`,

	"typescript": `# Project Test Conventions (TypeScript)

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
- Framework: <!-- vitest / jest / mocha -->
- Naming: describe('<Unit>') / it('<scenario>'), e.g. it('returns 404 when user not found')

## Test infrastructure
<!-- Example: -->
<!-- - src/test/setup.ts — global beforeAll / afterAll hooks -->
<!-- - src/test/factories/ — createUser(), createOrder() test helpers -->

## Domain vocabulary
<!-- Define key domain terms. -->

## Do not mock
<!-- Example: shared/value-objects — pure functions, test directly -->

## Mock style
<!-- vi.mock / jest.mock / sinon.stub -->
`,

	"java": `# Project Test Conventions (Java)

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
- Framework: JUnit 5 + Mockito
- Naming: <method>_<scenario>(), e.g. processPayment_insufficientFunds_throwsException()
- Follow Arrange / Act / Assert structure with comments

## Test infrastructure
<!-- Example: -->
<!-- - src/test/.../TestConfig.java — Spring test context configuration -->
<!-- - src/test/.../TestDataBuilder.java — builder helpers for domain objects -->

## Domain vocabulary
<!-- Define key domain terms. -->

## Do not mock
<!-- Example: value objects, records — instantiate directly -->

## Integration tests
<!-- Example: classes annotated @SpringBootTest with @ActiveProfiles("test") -->
`,

	"csharp": `# Project Test Conventions (C#)

## Overview
<!-- Describe what this project does in 1–2 sentences. -->

## Test conventions
- Framework: xUnit + Moq
- Naming: <Method>_<Scenario>_<ExpectedResult>(), e.g. ProcessPayment_InsufficientFunds_ThrowsException()
- Follow Arrange / Act / Assert comment structure

## Test infrastructure
<!-- Example: -->
<!-- - Tests/Helpers/TestFixture.cs — shared IClassFixture setup -->
<!-- - Tests/Builders/ — fluent builder helpers for domain objects -->

## Domain vocabulary
<!-- Define key domain terms. -->

## Do not mock
<!-- Example: value objects, records — instantiate directly -->
`,
}
