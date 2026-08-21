package csharp_test

import (
	"strings"
	"testing"

	"github.com/orieken/assay/internal/drivers/csharp"
)

func TestNunitToXunit_AttributesAndAssertions(t *testing.T) {
	driver := &csharp.Driver{}
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "nunit" && mg.To() == "xunit" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("nunit→xunit migrator not found")
	}

	input := `using NUnit.Framework;

[TestFixture]
public class PaymentTests
{
    [SetUp]
    public void SetUp() {}

    [TearDown]
    public void TearDown() {}

    [Test]
    public void Charge_ShouldReturnTrue()
    {
        Assert.AreEqual(100, result);
        Assert.IsTrue(flag);
        Assert.IsNull(val);
        Assert.IsNotNull(other);
    }

    [Ignore("not ready")]
    [Test]
    public void Pending() {}
}
`
	got, err := m.MigrateFile(input)
	if err != nil {
		t.Fatalf("MigrateFile: %v", err)
	}

	wants := []string{
		"using Xunit;",
		"[Fact]",
		"Assert.Equal(100,",
		"Assert.True(flag)",
		"Assert.Null(val)",
		"Assert.NotNull(other)",
		`[Fact(Skip = "not ready")]`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in output:\n%s", w, got)
		}
	}
	if strings.Contains(got, "using NUnit.Framework;") {
		t.Errorf("NUnit namespace should be replaced:\n%s", got)
	}
	if strings.Contains(got, "[TestFixture]") {
		t.Errorf("[TestFixture] should be removed:\n%s", got)
	}
}

func TestXunitToNunit_ReplacesFactAndAssertions(t *testing.T) {
	driver := &csharp.Driver{}
	var m interface{ MigrateFile(string) (string, error) }
	for _, mg := range driver.ListMigrators() {
		if mg.From() == "xunit" && mg.To() == "nunit" {
			m = mg
		}
	}
	if m == nil {
		t.Fatal("xunit→nunit migrator not found")
	}

	input := `using Xunit;

public class PaymentTests
{
    [Fact]
    public void Charge_Works()
    {
        Assert.Equal(100, result);
        Assert.True(flag);
        Assert.Null(val);
    }
}
`
	got, _ := m.MigrateFile(input)

	wants := []string{
		"using NUnit.Framework;",
		"[Test]",
		"Assert.AreEqual(100,",
		"Assert.IsTrue(flag)",
		"Assert.IsNull(val)",
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("expected %q in output:\n%s", w, got)
		}
	}
}
