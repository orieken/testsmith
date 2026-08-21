package java

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

var javaValidators = []*validation.TextValidator{
	junit5Validator(),
	junit4Validator(),
	testngValidator(),
}

func junit5Validator() *validation.TextValidator {
	return validation.New("junit5", "mockito").
		Require("missing-junit5-import", `import org\.junit\.jupiter`,
			"JUnit 5 import not found — expected 'import org.junit.jupiter.*'",
			domain.SeverityWarning).
		Forbid("junit4-import-in-junit5", `import org\.junit\.Test;`,
			"JUnit 4 import detected — run 'assay migrate --from junit4 --to junit5'",
			domain.SeverityError).
		Forbid("junit4-runwith-in-junit5", `@RunWith\(`,
			"@RunWith is JUnit 4 — use @ExtendWith for JUnit 5",
			domain.SeverityError).
		Forbid("junit4-assert-in-junit5", `Assert\.assert`,
			"JUnit 4 Assert class detected — use Assertions from JUnit 5",
			domain.SeverityWarning).
		Forbid("junit4-lifecycle-in-junit5", `@Before\b`,
			"@Before is JUnit 4 — use @BeforeEach for JUnit 5",
			domain.SeverityError)
}

func junit4Validator() *validation.TextValidator {
	return validation.New("junit4", "mockito").
		Require("missing-junit4-import", `import org\.junit\.Test;`,
			"JUnit 4 import not found — expected 'import org.junit.Test'",
			domain.SeverityWarning).
		Forbid("junit5-import-in-junit4", `import org\.junit\.jupiter`,
			"JUnit 5 import detected — run 'assay migrate --from junit5 --to junit4'",
			domain.SeverityError).
		Forbid("junit5-extendwith-in-junit4", `@ExtendWith\(`,
			"@ExtendWith is JUnit 5 — use @RunWith for JUnit 4",
			domain.SeverityError)
}

func testngValidator() *validation.TextValidator {
	return validation.New("testng", "mockito").
		Require("missing-testng-import", `import org\.testng`,
			"TestNG import not found — expected 'import org.testng.*'",
			domain.SeverityWarning).
		Forbid("junit-import-in-testng", `import org\.junit`,
			"JUnit import found in TestNG test file",
			domain.SeverityWarning)
}

func (d *Driver) ValidateFile(framework, mockLib, content string) []domain.ValidationIssue {
	for _, v := range javaValidators {
		if v.Framework() == framework {
			return v.Validate(content)
		}
	}
	return nil
}
