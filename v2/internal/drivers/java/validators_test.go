package java_test

import (
	"testing"

	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/drivers/java"
)

func TestValidateFile_Junit5_CleanFile(t *testing.T) {
	driver := &java.Driver{}
	content := `import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Assertions;

public class PaymentTest {
    @Test
    public void testCharge() {
        Assertions.assertEquals(100, result);
    }
}
`
	issues := driver.ValidateFile("junit5", "mockito", content)
	for _, iss := range issues {
		if iss.Severity == domain.SeverityError {
			t.Errorf("unexpected error in clean file: [%s] %s", iss.Rule, iss.Message)
		}
	}
}

func TestValidateFile_Junit5_DetectsJunit4Import(t *testing.T) {
	driver := &java.Driver{}
	content := `import org.junit.Test;
import org.junit.Assert;

public class PaymentTest {
    @RunWith(MockitoJUnitRunner.class)
    @Test public void testCharge() {
        Assert.assertEquals(100, result);
    }
}
`
	issues := driver.ValidateFile("junit5", "mockito", content)
	errorRules := issueRules(issues, domain.SeverityError)
	if !contains(errorRules, "junit4-import-in-junit5") {
		t.Errorf("expected junit4-import-in-junit5 error, got: %v", errorRules)
	}
	if !contains(errorRules, "junit4-runwith-in-junit5") {
		t.Errorf("expected junit4-runwith-in-junit5 error, got: %v", errorRules)
	}
}

func TestValidateFile_Junit4_DetectsJunit5Import(t *testing.T) {
	driver := &java.Driver{}
	content := `import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.Assertions;
`
	issues := driver.ValidateFile("junit4", "mockito", content)
	rules := issueRules(issues, domain.SeverityError)
	if !contains(rules, "junit5-import-in-junit4") {
		t.Errorf("expected junit5-import-in-junit4 error, got: %v", rules)
	}
}

func TestValidateFile_UnknownFramework_ReturnsNil(t *testing.T) {
	driver := &java.Driver{}
	issues := driver.ValidateFile("unknown", "unknown", "anything")
	if issues != nil {
		t.Errorf("expected nil for unknown framework, got %v", issues)
	}
}

// helpers

func issueRules(issues []domain.ValidationIssue, sev domain.Severity) []string {
	var rules []string
	for _, iss := range issues {
		if iss.Severity == sev {
			rules = append(rules, iss.Rule)
		}
	}
	return rules
}

func contains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}
