package java

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/orieken/assay/internal/domain"
)

// ── registry ──────────────────────────────────────────────────────────────────

var registry = buildRegistry()

func buildRegistry() *domain.AdapterRegistry {
	r := domain.NewAdapterRegistry()
	r.SetDefault(&junit5MockitoAdapter{})
	r.Register(&junit4MockitoAdapter{})
	r.Register(&testngMockitoAdapter{})
	r.Register(&springBootMockitoAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared render data ────────────────────────────────────────────────────────

type renderData struct {
	Package   string
	Members   []domain.PublicMember
	LLMBodies map[string][]string
}

func (d renderData) PublicMethods(m domain.PublicMember) []domain.MethodInfo {
	var out []domain.MethodInfo
	for _, method := range m.Methods {
		if method.IsPublic {
			out = append(out, method)
		}
	}
	return out
}

func (d renderData) BodyFor(name string) []string {
	if d.LLMBodies == nil {
		return nil
	}
	return d.LLMBodies[name]
}

func newRenderData(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) renderData {
	return renderData{
		Package:   javaPackage(analysis.ModulePath),
		Members:   analysis.PublicAPI,
		LLMBodies: opts.LLMBodies,
	}
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render java test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"title":  strings.Title, //nolint:staticcheck
	"indent": indentLines,
}

// ── 1. JUnit 5 + Mockito (default) ───────────────────────────────────────────

type junit5MockitoAdapter struct{}

func (a *junit5MockitoAdapter) Framework() string   { return "junit5" }
func (a *junit5MockitoAdapter) MockLibrary() string { return "mockito" }
func (a *junit5MockitoAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit5",
		TestFileSuffix: "Test.java",
		TestFuncPrefix: "@Test",
	}
}

func (a *junit5MockitoAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(junit5Tmpl, newRenderData(analysis, opts))
}

var junit5Tmpl = template.Must(template.New("junit5-mockito").Funcs(sharedFuncs).Parse(`package {{ .Package }};

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.DisplayName;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
{{ range .Members }}
{{ if eq .Kind "class" -}}
class {{ .Name }}Test {

    // @Mock
    // private IDependency dependency;

    @InjectMocks
    private {{ .Name }} sut;

    @BeforeEach
    void setUp() {
        // TODO: additional setup if needed
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    @Test
    @DisplayName("{{ .Name }} should work correctly")
    void {{ .Name }}_shouldWork() {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: call sut.{{ .Name }}(...)

        // Assert
        assertNotNull(sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 2. JUnit 4 + Mockito ──────────────────────────────────────────────────────

type junit4MockitoAdapter struct{}

func (a *junit4MockitoAdapter) Framework() string   { return "junit4" }
func (a *junit4MockitoAdapter) MockLibrary() string { return "mockito" }
func (a *junit4MockitoAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit4",
		TestFileSuffix: "Test.java",
		TestFuncPrefix: "@Test",
	}
}

func (a *junit4MockitoAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(junit4Tmpl, newRenderData(analysis, opts))
}

var junit4Tmpl = template.Must(template.New("junit4-mockito").Funcs(sharedFuncs).Parse(`package {{ .Package }};

import org.junit.Test;
import org.junit.Before;
import org.junit.runner.RunWith;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.junit.MockitoJUnitRunner;
import static org.junit.Assert.*;
import static org.mockito.Mockito.*;

@RunWith(MockitoJUnitRunner.class)
{{ range .Members }}
{{ if eq .Kind "class" -}}
public class {{ .Name }}Test {

    // @Mock
    // private IDependency dependency;

    @InjectMocks
    private {{ .Name }} sut;

    @Before
    public void setUp() {
        // TODO: additional setup if needed
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    @Test
    public void {{ .Name }}_shouldWork() {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: call sut.{{ .Name }}(...)

        // Assert
        assertNotNull(sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 3. TestNG + Mockito ───────────────────────────────────────────────────────

type testngMockitoAdapter struct{}

func (a *testngMockitoAdapter) Framework() string   { return "testng" }
func (a *testngMockitoAdapter) MockLibrary() string { return "mockito" }
func (a *testngMockitoAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "testng",
		TestFileSuffix: "Test.java",
		TestFuncPrefix: "@Test",
	}
}

func (a *testngMockitoAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(testngTmpl, newRenderData(analysis, opts))
}

var testngTmpl = template.Must(template.New("testng-mockito").Funcs(sharedFuncs).Parse(`package {{ .Package }};

import org.testng.annotations.Test;
import org.testng.annotations.BeforeMethod;
import org.mockito.InjectMocks;
import org.mockito.Mock;
import org.mockito.MockitoAnnotations;
import static org.testng.Assert.*;
import static org.mockito.Mockito.*;

{{ range .Members }}
{{ if eq .Kind "class" -}}
public class {{ .Name }}Test {

    // @Mock
    // private IDependency dependency;

    @InjectMocks
    private {{ .Name }} sut;

    private AutoCloseable mocks;

    @BeforeMethod
    public void setUp() {
        mocks = MockitoAnnotations.openMocks(this);
    }
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    @Test
    public void {{ .Name }}_shouldWork() {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: call sut.{{ .Name }}(...)

        // Assert
        assertNotNull(sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── 4. Spring Boot Test + Mockito ─────────────────────────────────────────────

type springBootMockitoAdapter struct{}

func (a *springBootMockitoAdapter) Framework() string   { return "springboot" }
func (a *springBootMockitoAdapter) MockLibrary() string { return "mockito" }
func (a *springBootMockitoAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "springboot",
		TestFileSuffix: "Test.java",
		TestFuncPrefix: "@Test",
	}
}

func (a *springBootMockitoAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(springBootTmpl, newRenderData(analysis, opts))
}

var springBootTmpl = template.Must(template.New("springboot-mockito").Funcs(sharedFuncs).Parse(`package {{ .Package }};

import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.BeforeEach;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.test.mock.mockito.MockBean;
import org.springframework.beans.factory.annotation.Autowired;
import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.Mockito.*;

@SpringBootTest
{{ range .Members }}
{{ if eq .Kind "class" -}}
class {{ .Name }}Test {

    @Autowired
    private {{ .Name }} sut;

    // @MockBean
    // private IDependency dependency;
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name }}
    @Test
    void {{ .Name }}_shouldWork() {
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
        // Arrange
        // TODO: arrange

        // Act
        // TODO: call sut.{{ .Name }}(...)

        // Assert
        assertNotNull(sut);
{{ end }}
    }
{{ end }}
}
{{ end -}}
{{ end }}`))

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

func (a *junit5MockitoAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":       "JUnit 5",
		"mock_library":    "Mockito",
		"assert_style":    "Assertions.assertEquals(expected, actual)",
		"mock_style":      "@Mock + @ExtendWith(MockitoExtension.class), when(x).thenReturn(y)",
		"test_annotation": "@Test",
	}
}

func (a *junit4MockitoAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":       "JUnit 4",
		"mock_library":    "Mockito",
		"assert_style":    "Assert.assertEquals(expected, actual)",
		"mock_style":      "@Mock + @RunWith(MockitoJUnitRunner.class), when(x).thenReturn(y)",
		"test_annotation": "@Test",
	}
}

func (a *testngMockitoAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":       "TestNG",
		"mock_library":    "Mockito",
		"assert_style":    "Assert.assertEquals(actual, expected)",
		"mock_style":      "@Mock + MockitoAnnotations.openMocks(this), when(x).thenReturn(y)",
		"test_annotation": "@Test",
	}
}

func (a *springBootMockitoAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":       "Spring Boot Test",
		"mock_library":    "Mockito",
		"assert_style":    "Assertions.assertEquals(expected, actual)",
		"mock_style":      "@MockBean, when(x).thenReturn(y)",
		"test_annotation": "@SpringBootTest",
	}
}
