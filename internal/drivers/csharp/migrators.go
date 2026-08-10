package csharp

import (
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/migration"
)

var csMigrators = []domain.Migrator{
	nunitToXunit(),
	xunitToNunit(),
}

func nunitToXunit() *migration.TextMigrator {
	return migration.New("nunit", "xunit").
		// Namespace swap.
		Add(`using NUnit\.Framework;`, `using Xunit;`).
		// Class-level attribute (xUnit discovers plain public classes).
		Add(`\[TestFixture\]\s*\n`, "").
		// Ignore before Test so [Ignore][Test] converts correctly.
		Add(`\[Ignore\("([^"]*)"\)\]`, `[Fact(Skip = "$1")]`).
		Add(`\[Ignore\]`, `[Fact(Skip = "ignored")]`).
		// [Test] alone on its line → [Fact].
		// Use \s*$ so trailing whitespace and end-of-line are handled safely.
		Add(`(?m)\[Test\]\s*$`, `[Fact]`).
		// TestCase → Theory + InlineData (single line, simple values).
		Add(`\[TestCase\(([^)]+)\)\]`, "[Theory]\n    [InlineData($1)]").
		// Lifecycle — xUnit uses constructor/IDisposable; leave a reminder.
		Add(`\[SetUp\]`, `// xUnit: move SetUp logic to the constructor`).
		Add(`\[TearDown\]`, `// xUnit: move TearDown logic to Dispose() : IDisposable`).
		Add(`\[OneTimeSetUp\]`, `// xUnit: move OneTimeSetUp to IClassFixture<T>`).
		Add(`\[OneTimeTearDown\]`, `// xUnit: move OneTimeTearDown to IClassFixture<T>.Dispose()`).
		// Assertions — note: xUnit swaps expected/actual order.
		Add(`Assert\.AreEqual\(([^,]+),\s*`, `Assert.Equal($1, `).
		Add(`Assert\.AreNotEqual\(([^,]+),\s*`, `Assert.NotEqual($1, `).
		Add(`Assert\.IsTrue\(([^)]+)\)`, `Assert.True($1)`).
		Add(`Assert\.IsFalse\(([^)]+)\)`, `Assert.False($1)`).
		Add(`Assert\.IsNull\(([^)]+)\)`, `Assert.Null($1)`).
		Add(`Assert\.IsNotNull\(([^)]+)\)`, `Assert.NotNull($1)`).
		Add(`Assert\.AreSame\(([^,]+),\s*`, `Assert.Same($1, `).
		Add(`Assert\.AreNotSame\(([^,]+),\s*`, `Assert.NotSame($1, `).
		Add(`Assert\.IsInstanceOf<([^>]+)>\(([^)]+)\)`, `Assert.IsType<$1>($2)`).
		Add(`Assert\.Throws<([^>]+)>\(`, `Assert.Throws<$1>(`)
}

func xunitToNunit() *migration.TextMigrator {
	return migration.New("xunit", "nunit").
		Add(`using Xunit;`, `using NUnit.Framework;`).
		Add(`\[Fact\(Skip\s*=\s*"([^"]*)"\)\]`, `[Ignore("$1")]`).
		// [Fact] alone on its line → [Test].
		Add(`(?m)\[Fact\]\s*$`, `[Test]`).
		Add(`\[Theory\]\s*\n\s*\[InlineData\(([^)]+)\)\]`, `[TestCase($1)]`).
		// Assertions.
		Add(`Assert\.Equal\(([^,]+),\s*`, `Assert.AreEqual($1, `).
		Add(`Assert\.NotEqual\(([^,]+),\s*`, `Assert.AreNotEqual($1, `).
		Add(`Assert\.True\(([^)]+)\)`, `Assert.IsTrue($1)`).
		Add(`Assert\.False\(([^)]+)\)`, `Assert.IsFalse($1)`).
		Add(`Assert\.Null\(([^)]+)\)`, `Assert.IsNull($1)`).
		Add(`Assert\.NotNull\(([^)]+)\)`, `Assert.IsNotNull($1)`).
		Add(`Assert\.Same\(([^,]+),\s*`, `Assert.AreSame($1, `).
		Add(`Assert\.NotSame\(([^,]+),\s*`, `Assert.AreNotSame($1, `).
		Add(`Assert\.IsType<([^>]+)>\(([^)]+)\)`, `Assert.IsInstanceOf<$1>($2)`)
}
