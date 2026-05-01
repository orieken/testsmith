package java

import (
	"strings"
	"testing"

	"github.com/orieken/testsmith/internal/domain"
)

func makeJavaAnalysis() *domain.SourceAnalysis {
	return &domain.SourceAnalysis{
		SourcePath: "/proj/src/main/java/com/example/OrderService.java",
		ModulePath: "com.example.OrderService",
		PublicAPI: []domain.PublicMember{
			{
				Name: "OrderService",
				Kind: "class",
				Methods: []domain.MethodInfo{
					{Name: "createOrder", IsPublic: true},
					{Name: "cancelOrder", IsPublic: true},
					{Name: "validateInternal", IsPublic: false},
				},
			},
		},
		Project: &domain.ProjectContext{Language: "java", Metadata: map[string]any{}},
	}
}

func javaCtxWith(framework string) *domain.ProjectContext {
	return &domain.ProjectContext{
		Language: "java",
		Metadata: map[string]any{"framework": framework, "mock_library": "mockito"},
	}
}

// ── selectAdapter ─────────────────────────────────────────────────────────────

func TestJavaSelectAdapter_DefaultIsJUnit5(t *testing.T) {
	a := selectAdapter(nil)
	if a.Framework() != "junit5" {
		t.Errorf("expected junit5, got %q", a.Framework())
	}
}

func TestJavaSelectAdapter_JUnit4(t *testing.T) {
	a := selectAdapter(javaCtxWith("junit4"))
	if a.Framework() != "junit4" {
		t.Errorf("expected junit4, got %q", a.Framework())
	}
}

func TestJavaSelectAdapter_TestNG(t *testing.T) {
	a := selectAdapter(javaCtxWith("testng"))
	if a.Framework() != "testng" {
		t.Errorf("expected testng, got %q", a.Framework())
	}
}

func TestJavaSelectAdapter_SpringBoot(t *testing.T) {
	a := selectAdapter(javaCtxWith("springboot"))
	if a.Framework() != "springboot" {
		t.Errorf("expected springboot, got %q", a.Framework())
	}
}

func TestJavaSelectAdapter_UnknownFallsBackToDefault(t *testing.T) {
	a := selectAdapter(javaCtxWith("junit3"))
	if a.Framework() != "junit5" {
		t.Errorf("expected junit5 fallback, got %q", a.Framework())
	}
}

// ── JUnit 5 + Mockito adapter ─────────────────────────────────────────────────

func TestJUnit5Adapter_Imports(t *testing.T) {
	a := &junit5MockitoAdapter{}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "org.junit.jupiter.api.Test") {
		t.Error("missing JUnit 5 import")
	}
	if !strings.Contains(out, "MockitoExtension") {
		t.Error("missing Mockito extension")
	}
	if !strings.Contains(out, "@InjectMocks") {
		t.Error("missing @InjectMocks")
	}
}

func TestJUnit5Adapter_TestMethods(t *testing.T) {
	a := &junit5MockitoAdapter{}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "createOrder_shouldWork") {
		t.Error("missing test for createOrder")
	}
	if !strings.Contains(out, "@DisplayName") {
		t.Error("missing @DisplayName")
	}
	if strings.Contains(out, "validateInternal") {
		t.Error("private method must not appear in test")
	}
}

// ── JUnit 4 + Mockito adapter ─────────────────────────────────────────────────

func TestJUnit4Adapter_RunWith(t *testing.T) {
	a := &junit4MockitoAdapter{}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@RunWith(MockitoJUnitRunner.class)") {
		t.Error("missing @RunWith")
	}
	if !strings.Contains(out, "public class OrderServiceTest") {
		t.Error("missing public class")
	}
}

// ── TestNG + Mockito adapter ──────────────────────────────────────────────────

func TestTestNGAdapter_Annotations(t *testing.T) {
	a := &testngMockitoAdapter{}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "org.testng.annotations.Test") {
		t.Error("missing TestNG import")
	}
	if !strings.Contains(out, "@BeforeMethod") {
		t.Error("missing @BeforeMethod")
	}
	if !strings.Contains(out, "MockitoAnnotations.openMocks") {
		t.Error("missing openMocks call")
	}
}

// ── Spring Boot adapter ───────────────────────────────────────────────────────

func TestSpringBootAdapter_Annotations(t *testing.T) {
	a := &springBootMockitoAdapter{}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), domain.GenerateOpts{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "@SpringBootTest") {
		t.Error("missing @SpringBootTest")
	}
	if !strings.Contains(out, "@Autowired") {
		t.Error("missing @Autowired")
	}
}

// ── LLM body injection ────────────────────────────────────────────────────────

func TestJUnit5Adapter_InjectsLLMBody(t *testing.T) {
	a := &junit5MockitoAdapter{}
	opts := domain.GenerateOpts{
		LLMBodies: map[string][]string{
			"createOrder": {"Order order = sut.createOrder(req);", "assertNotNull(order);"},
		},
	}
	out, err := a.GenerateTestFile(makeJavaAnalysis(), opts)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "Order order = sut.createOrder(req)") {
		t.Error("LLM body not injected")
	}
}
