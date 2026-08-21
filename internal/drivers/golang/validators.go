package golang

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

var goValidators = []*validation.TextValidator{
	testifyValidator(),
	stdlibValidator(),
}

func testifyValidator() *validation.TextValidator {
	return validation.New("testing", "testify").
		Require("missing-testify-import", `"github\.com/stretchr/testify`,
			"testify not imported — adapter expects testify assertions",
			domain.SeverityWarning)
}

func stdlibValidator() *validation.TextValidator {
	return validation.New("testing", "interfaces").
		Forbid("testify-in-stdlib", `"github\.com/stretchr/testify`,
			"testify found but adapter is stdlib — consider removing the dependency",
			domain.SeverityInfo)
}

func (d *Driver) ValidateFile(framework, mockLib, content string) []domain.ValidationIssue {
	for _, v := range goValidators {
		if v.Framework() == framework && v.MockLibrary() == mockLib {
			return v.Validate(content)
		}
	}
	return nil
}
