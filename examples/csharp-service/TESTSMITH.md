# TestSmith Project Knowledge — csharp-service

## Framework
- Language: C# 12 / .NET 8 (nullable enabled, implicit usings)
- Test framework: xUnit 2.x
- Mocks: Moq 4.x (`Mock<T>`, `.Setup()`, `.Verify()`)
- Assertion style: xUnit `Assert.Equal`, `Assert.Throws<T>`, `Assert.True/False`

## Conventions
- Test classes: `<Subject>Tests` in `tests/Subscriptions.Tests/`
- Use `[Fact]` for single-case tests and `[Theory] + [InlineData]` for parametrised cases
- Name pattern: `MethodName_StateUnderTest_ExpectedBehavior`
- Arrange/Act/Assert sections separated by blank lines with `// Arrange` comments
- Use `Assert.Throws<ExceptionType>(() => ...)` for exception assertions

## Example test structure
```csharp
public class SubscriptionServiceTests
{
    private readonly SubscriptionService _sut = new();

    [Fact]
    public void Subscribe_NewTenant_ReturnsActiveSubscription()
    {
        // Arrange / Act
        var sub = _sut.Subscribe("tenant-1", Plan.Pro);

        // Assert
        Assert.True(sub.IsActive);
        Assert.Equal("tenant-1", sub.TenantId);
    }
}
```
