package kotlin

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
	r.SetDefault(&junit5Adapter{})
	r.Register(&kotestAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared helpers ────────────────────────────────────────────────────────────

type renderData struct {
	PackageName string
	ModulePath  string
	Members     []domain.PublicMember
	LLMBodies   map[string][]string
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
	pkg := packageName(analysis.ModulePath)
	return renderData{
		PackageName: pkg,
		ModulePath:  analysis.ModulePath,
		Members:     analysis.PublicAPI,
		LLMBodies:   opts.LLMBodies,
	}
}

func packageName(modulePath string) string {
	if idx := strings.LastIndexByte(modulePath, '.'); idx != -1 {
		return modulePath[:idx]
	}
	return modulePath
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render kotlin test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"indent": indentLines,
}

// ── 1. JUnit 5 + MockK (default) ─────────────────────────────────────────────

type junit5Adapter struct{}

func (a *junit5Adapter) Framework() string   { return "junit5" }
func (a *junit5Adapter) MockLibrary() string { return "mockk" }
func (a *junit5Adapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "junit5",
		TestFileSuffix: "Test.kt",
		FixtureDir:     "test",
		TestFuncPrefix: "",
	}
}

func (a *junit5Adapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(junit5Tmpl, newRenderData(analysis, opts))
}

func (a *junit5Adapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "junit5",
		"mock_library": "mockk",
		"assert_style": "assertEquals(expected, actual)",
		"mock_style":   "mockk<T>(), every { mock.method() } returns value, verify { }",
		"test_prefix":  "@Test",
	}
}

var junit5Tmpl = template.Must(template.New("kotlin-junit5").Funcs(sharedFuncs).Parse(
	`package {{ .PackageName }}

import io.mockk.mockk
import io.mockk.every
import io.mockk.verify
import org.junit.jupiter.api.Assertions.*
import org.junit.jupiter.api.Test
import org.junit.jupiter.api.DisplayName
{{ range .Members -}}
{{ if eq .Kind "function" }}
@DisplayName("{{ .Name }}")
class {{ .Name }}Test {

    @Test
    fun ` + "`" + `{{ .Name }} returns expected result` + "`" + `() {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        // TODO: arrange
        // val result = {{ .Name }}(...)
        // assertEquals(expected, result)
        TODO("implement test for {{ .Name }}")
{{ end -}}
    }
}
{{ else if eq .Kind "class" -}}
{{ $className := .Name -}}
@DisplayName("{{ $className }}")
class {{ $className }}Test {
{{ range ($.PublicMethods .) }}
    @Test
    fun ` + "`" + `{{ .Name }} behaves correctly` + "`" + `() {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        // TODO: arrange
        // val sut = {{ $className }}(...)
        // val result = sut.{{ .Name }}(...)
        // assertEquals(expected, result)
        TODO("implement test for {{ $className }}.{{ .Name }}")
{{ end -}}
    }
{{ end -}}
}
{{ end -}}
{{ end -}}
`))

// ── 2. Kotest ─────────────────────────────────────────────────────────────────

type kotestAdapter struct{}

func (a *kotestAdapter) Framework() string   { return "kotest" }
func (a *kotestAdapter) MockLibrary() string { return "mockk" }
func (a *kotestAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "kotest",
		TestFileSuffix: "Test.kt",
		FixtureDir:     "test",
		TestFuncPrefix: "",
	}
}

func (a *kotestAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(kotestTmpl, newRenderData(analysis, opts))
}

func (a *kotestAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "kotest",
		"mock_library": "mockk",
		"assert_style": "result shouldBe expected",
		"mock_style":   "mockk<T>(), every { mock.method() } returns value, verify { }",
		"test_prefix":  "FreeSpec/StringSpec",
	}
}

var kotestTmpl = template.Must(template.New("kotlin-kotest").Funcs(sharedFuncs).Parse(
	`package {{ .PackageName }}

import io.kotest.core.spec.style.FreeSpec
import io.kotest.matchers.shouldBe
import io.mockk.mockk
import io.mockk.every
import io.mockk.verify
{{ range .Members -}}
{{ if eq .Kind "function" }}
class {{ .Name }}Test : FreeSpec({

    "{{ .Name }}" - {
        "returns expected result" {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 12 }}
{{ else -}}
            // TODO: arrange
            // val result = {{ .Name }}(...)
            // result shouldBe expected
            TODO("implement test for {{ .Name }}")
{{ end -}}
        }
    }
})
{{ else if eq .Kind "class" -}}
{{ $className := .Name -}}
class {{ $className }}Test : FreeSpec({

    "{{ $className }}" - {
{{ range ($.PublicMethods .) -}}
        "{{ .Name }} behaves correctly" {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 12 }}
{{ else -}}
            // TODO: arrange
            // val sut = {{ $className }}(...)
            // val result = sut.{{ .Name }}(...)
            // result shouldBe expected
            TODO("implement test for {{ $className }}.{{ .Name }}")
{{ end -}}
        }
{{ end -}}
    }
})
{{ end -}}
{{ end -}}
`))

// ── helpers ───────────────────────────────────────────────────────────────────

func indentLines(lines []string, spaces int) string {
	prefix := strings.Repeat(" ", spaces)
	var sb strings.Builder
	for _, l := range lines {
		sb.WriteString(prefix + l + "\n")
	}
	return sb.String()
}

func normaliseBlankLines(s string) string {
	lines := strings.Split(s, "\n")
	var out []string
	blanks := 0
	for _, l := range lines {
		if strings.TrimSpace(l) == "" {
			blanks++
			if blanks <= 2 {
				out = append(out, "")
			}
		} else {
			blanks = 0
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}
