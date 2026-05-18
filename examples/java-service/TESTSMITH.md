# TestSmith Project Knowledge — java-service

## Framework
- Language: Java 21 (records, sealed interfaces available)
- Test framework: JUnit 5 (Jupiter) + Mockito 5
- Assertion style: JUnit 5 `assertThrows`, `assertEquals`, `assertThat` with Hamcrest

## Conventions
- Test classes live in `src/test/java/` mirroring the main source tree
- Class name: `<Subject>Test` (e.g. `NotificationServiceTest`)
- Use `@ExtendWith(MockitoExtension.class)` for Mockito injection
- Inject mocks with `@Mock`, class-under-test with `@InjectMocks`
- `@Test` on every test method; `@DisplayName` optional but encouraged
- Use `assertThrows` for exception assertions; verify message with `.getMessage()`

## Example test structure
```java
@ExtendWith(MockitoExtension.class)
class NotificationServiceTest {

    @Mock NotificationSender sender;
    @InjectMocks NotificationService service;

    @Test
    void send_storesAndDeliversNotification() {
        Notification n = service.send("user-1", "Hello", "World");
        assertNotNull(n.id());
        verify(sender).deliver(n);
    }
}
```
