package java_test

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/drivers/java"
)

func TestJunit4ToJunit5_ImportsAndAnnotations(t *testing.T) {
	driver := &java.Driver{}
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "junit4" && mg.To() == "junit5" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("junit4→junit5 migrator not found")
	}

	input := `import org.junit.Test;
import org.junit.Before;
import org.junit.After;
import org.junit.BeforeClass;
import org.junit.AfterClass;
import org.junit.Assert;
import org.junit.Ignore;
import org.junit.runner.RunWith;
import org.mockito.junit.MockitoJUnitRunner;

@RunWith(MockitoJUnitRunner.class)
public class PaymentTest {
    @Before public void setUp() {}
    @After public void tearDown() {}
    @BeforeClass public static void init() {}
    @AfterClass public static void cleanup() {}
    @Ignore
    @Test public void testCharge() {
        Assert.assertEquals(100, result);
        Assert.assertTrue(flag);
        Assert.assertNull(val);
    }
}
`
	got, err := m.MigrateFile(input)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}

	wants := []string{
		"import org.junit.jupiter.api.Test;",
		"import org.junit.jupiter.api.BeforeEach;",
		"import org.junit.jupiter.api.AfterEach;",
		"import org.junit.jupiter.api.BeforeAll;",
		"import org.junit.jupiter.api.AfterAll;",
		"import org.junit.jupiter.api.Assertions;",
		"import org.junit.jupiter.api.Disabled;",
		"import org.junit.jupiter.api.extension.ExtendWith;",
		"import org.mockito.junit.jupiter.MockitoExtension;",
		"@ExtendWith(MockitoExtension.class)",
		"@BeforeEach",
		"@AfterEach",
		"@BeforeAll",
		"@AfterAll",
		"@Disabled",
		"Assertions.assertEquals(",
		"Assertions.assertTrue(",
		"Assertions.assertNull(",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in output:\n%s", w, got)
		}
	}
	// Old imports/annotations should be gone.
	nots := []string{"import org.junit.Test;", "import org.junit.Before;", "@Before\n", "@RunWith"}
	for _, n := range nots {
		if strings.Contains(got, n) {
			t.Errorf("should not contain %q:\n%s", n, got)
		}
	}
}

func TestJunit5ToJunit4_RoundTrip(t *testing.T) {
	driver := &java.Driver{}
	var fwd, rev interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "junit4" && mg.To() == "junit5" {
			fwd = mg
		}
		if mg.From() == "junit5" && mg.To() == "junit4" {
			rev = mg
		}
	}
	if fwd == nil || rev == nil {
		t.Fatal("both migrators required")
	}

	// Simple file that can round-trip cleanly.
	original := `import org.junit.Test;
import org.junit.Assert;

public class FooTest {
    @Test
    public void testFoo() {
        Assert.assertEquals(1, result);
    }
}
`
	upgraded, _ := fwd.MigrateFile(original)
	downgraded, _ := rev.MigrateFile(upgraded)

	// After round-trip, key JUnit 4 constructs should be restored.
	if !strings.Contains(downgraded, "import org.junit.Test;") {
		t.Errorf("expected JUnit 4 import after round-trip:\n%s", downgraded)
	}
	if !strings.Contains(downgraded, "Assert.assertEquals(") {
		t.Errorf("expected Assert.assertEquals after round-trip:\n%s", downgraded)
	}
}
