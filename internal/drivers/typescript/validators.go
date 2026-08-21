package typescript

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

var tsValidators = []*validation.TextValidator{
	jestValidator(),
	vitestValidator(),
}

func jestValidator() *validation.TextValidator {
	return validation.New("jest", "jest").
		Forbid("vitest-api-in-jest", `\bvi\.`,
			"file uses vi.* (Vitest API) but adapter is Jest — run 'assay migrate --from vitest --to jest'",
			domain.SeverityError).
		Forbid("vitest-import-in-jest", `from\s+['"]vitest['"]`,
			"file imports from 'vitest' but adapter is Jest",
			domain.SeverityError)
}

func vitestValidator() *validation.TextValidator {
	return validation.New("vitest", "vitest").
		Forbid("jest-api-in-vitest", `\bjest\.`,
			"file uses jest.* (Jest API) but adapter is Vitest — run 'assay migrate --from jest --to vitest'",
			domain.SeverityWarning).
		Forbid("jest-import-in-vitest", `from\s+['"]@jest/globals['"]`,
			"file imports from '@jest/globals' but adapter is Vitest",
			domain.SeverityError)
}

func (d *Driver) ValidateFile(framework, mockLib, content string) []domain.ValidationIssue {
	for _, v := range tsValidators {
		if v.Framework() == framework && v.MockLibrary() == mockLib {
			return v.Validate(content)
		}
	}
	return nil
}
