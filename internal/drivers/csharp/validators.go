package csharp

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/validation"
)

var csValidators = []*validation.TextValidator{
	xunitValidator(),
	nunitValidator(),
	mstestValidator(),
}

func xunitValidator() *validation.TextValidator {
	return validation.New("xunit", "moq").
		Require("missing-xunit-using", `using Xunit;`,
			"xUnit namespace not found — expected 'using Xunit;'",
			domain.SeverityWarning).
		Forbid("nunit-using-in-xunit", `using NUnit\.Framework;`,
			"NUnit namespace found — run 'assay migrate --from nunit --to xunit'",
			domain.SeverityError).
		Forbid("nunit-testfixture-in-xunit", `\[TestFixture\]`,
			"[TestFixture] is NUnit — remove it; xUnit discovers plain public classes",
			domain.SeverityError).
		Forbid("nunit-setup-in-xunit", `\[SetUp\]`,
			"[SetUp] is NUnit — use the constructor for xUnit test setup",
			domain.SeverityWarning).
		Forbid("mstest-testclass-in-xunit", `\[TestClass\]`,
			"[TestClass] is MSTest — not needed for xUnit",
			domain.SeverityError)
}

func nunitValidator() *validation.TextValidator {
	return validation.New("nunit", "nsubstitute").
		Require("missing-nunit-using", `using NUnit\.Framework;`,
			"NUnit namespace not found — expected 'using NUnit.Framework;'",
			domain.SeverityWarning).
		Forbid("xunit-using-in-nunit", `using Xunit;`,
			"xUnit namespace found — run 'assay migrate --from xunit --to nunit'",
			domain.SeverityError).
		Forbid("xunit-fact-in-nunit", `\[Fact\]`,
			"[Fact] is xUnit — use [Test] for NUnit",
			domain.SeverityWarning)
}

func mstestValidator() *validation.TextValidator {
	return validation.New("mstest", "moq").
		Require("missing-mstest-using", `using Microsoft\.VisualStudio\.TestTools\.UnitTesting;`,
			"MSTest namespace not found",
			domain.SeverityWarning).
		Forbid("nunit-using-in-mstest", `using NUnit\.Framework;`,
			"NUnit namespace found in MSTest file",
			domain.SeverityError).
		Forbid("xunit-using-in-mstest", `using Xunit;`,
			"xUnit namespace found in MSTest file",
			domain.SeverityError)
}

func (d *Driver) ValidateFile(framework, mockLib, content string) []domain.ValidationIssue {
	for _, v := range csValidators {
		if v.Framework() == framework {
			return v.Validate(content)
		}
	}
	return nil
}
