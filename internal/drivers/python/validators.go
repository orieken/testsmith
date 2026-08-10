package python

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

var pyValidators = []*validation.TextValidator{
	pytestMockValidator(),
	unittestMockValidator(),
	pytestValidator(),
	unittestValidator(),
}

func pytestMockValidator() *validation.TextValidator {
	return validation.New("pytest", "pytest-mock").
		Forbid("unittest-mock-in-pytest-mock", `from unittest\.mock import`,
			"file imports unittest.mock but adapter is pytest-mock — consider using mocker fixture",
			domain.SeverityWarning)
}

func unittestMockValidator() *validation.TextValidator {
	return validation.New("pytest", "unittest.mock").
		Forbid("mocker-in-unittest-mock", `\bmocker\.`,
			"file uses mocker (pytest-mock) but adapter is unittest.mock — run 'assay migrate --from pytest-mock --to unittest.mock'",
			domain.SeverityError)
}

func pytestValidator() *validation.TextValidator {
	return validation.New("pytest", "").
		Require("no-test-functions", `def test_`,
			"no test_ functions found — pytest tests must be named test_*",
			domain.SeverityWarning)
}

func unittestValidator() *validation.TextValidator {
	return validation.New("unittest", "unittest.mock").
		Require("no-testcase-class", `unittest\.TestCase`,
			"no TestCase class found — unittest tests should subclass unittest.TestCase",
			domain.SeverityWarning).
		Forbid("mocker-in-unittest", `\bmocker\.`,
			"file uses mocker (pytest-mock) but adapter is unittest",
			domain.SeverityError)
}

func (d *Driver) ValidateFile(framework, mockLib, content string) []domain.ValidationIssue {
	for _, v := range pyValidators {
		if v.Framework() == framework && v.MockLibrary() == mockLib {
			return v.Validate(content)
		}
	}
	return nil
}
