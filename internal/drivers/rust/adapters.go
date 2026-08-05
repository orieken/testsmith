package rust

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/orieken/testsmith/internal/domain"
)

// ── registry ──────────────────────────────────────────────────────────────────

var registry = buildRegistry()

func buildRegistry() *domain.AdapterRegistry {
	r := domain.NewAdapterRegistry()
	r.SetDefault(&builtinAdapter{})
	r.Register(&rstestAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared helpers ────────────────────────────────────────────────────────────

type renderData struct {
	ModulePath string
	Members    []domain.PublicMember
	LLMBodies  map[string][]string
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
		ModulePath: analysis.ModulePath,
		Members:    analysis.PublicAPI,
		LLMBodies:  opts.LLMBodies,
	}
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render rust test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"snakeCase": toSnakeCase,
	"indent":    indentLines,
}

// ── 1. built-in #[test] (default) ────────────────────────────────────────────

type builtinAdapter struct{}

func (a *builtinAdapter) Framework() string   { return "test" }
func (a *builtinAdapter) MockLibrary() string { return "mockall" }
func (a *builtinAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "test",
		TestFileSuffix: "_test.rs",
		FixtureDir:     "tests",
		TestFuncPrefix: "test_",
	}
}

func (a *builtinAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(builtinTmpl, newRenderData(analysis, opts))
}

func (a *builtinAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "test",
		"mock_library": "mockall",
		"assert_style": "assert_eq!(actual, expected)",
		"mock_style":   "#[automock] on trait, use MockMyTrait in tests",
		"test_prefix":  "test_",
	}
}

var builtinTmpl = template.Must(template.New("rust-builtin").Funcs(sharedFuncs).Parse(
	`#[cfg(test)]
mod tests {
    use super::*;
{{ range .Members -}}
{{ if eq .Kind "function" }}
    #[test]
    fn test_{{ snakeCase .Name }}() {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        // TODO: arrange
        // let result = {{ .Name }}(...);
        // assert_eq!(result, expected);
        todo!("implement test_{{ snakeCase .Name }}")
{{ end -}}
    }
{{ else if eq .Kind "class" -}}
{{ $typeName := .Name -}}
{{ range ($.PublicMethods .) }}
    #[test]
    fn test_{{ snakeCase $typeName }}_{{ snakeCase .Name }}() {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        // TODO: arrange
        // let sut = {{ $typeName }}::new(...);
        // let result = sut.{{ snakeCase .Name }}(...);
        // assert_eq!(result, expected);
        todo!("implement test_{{ snakeCase $typeName }}_{{ snakeCase .Name }}")
{{ end -}}
    }
{{ end -}}
{{ end -}}
{{ end -}}
}
`))

// ── 2. rstest ─────────────────────────────────────────────────────────────────

type rstestAdapter struct{}

func (a *rstestAdapter) Framework() string   { return "rstest" }
func (a *rstestAdapter) MockLibrary() string { return "mockall" }
func (a *rstestAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "rstest",
		TestFileSuffix: "_test.rs",
		FixtureDir:     "tests",
		TestFuncPrefix: "test_",
	}
}

func (a *rstestAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(rstestTmpl, newRenderData(analysis, opts))
}

func (a *rstestAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "rstest",
		"mock_library": "mockall",
		"assert_style": "assert_eq!(actual, expected)",
		"mock_style":   "#[automock] on trait, use MockMyTrait in tests",
		"test_prefix":  "test_",
	}
}

var rstestTmpl = template.Must(template.New("rust-rstest").Funcs(sharedFuncs).Parse(
	`use rstest::rstest;

#[cfg(test)]
mod tests {
    use super::*;
    use rstest::*;
{{ range .Members -}}
{{ if eq .Kind "function" }}
    #[rstest]
    #[case::happy_path(/* input */, /* expected */)]
    fn test_{{ snakeCase .Name }}(#[case] _input: (), #[case] _expected: ()) {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        todo!("implement test_{{ snakeCase .Name }}")
{{ end -}}
    }
{{ else if eq .Kind "class" -}}
{{ $typeName := .Name -}}
{{ range ($.PublicMethods .) }}
    #[rstest]
    fn test_{{ snakeCase $typeName }}_{{ snakeCase .Name }}() {
{{ $body := $.BodyFor .Name -}}
{{ if $body -}}
{{ indent $body 8 }}
{{ else -}}
        todo!("implement test_{{ snakeCase $typeName }}_{{ snakeCase .Name }}")
{{ end -}}
    }
{{ end -}}
{{ end -}}
{{ end -}}
}
`))

// ── helpers ───────────────────────────────────────────────────────────────────

func toSnakeCase(s string) string {
	var out strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				out.WriteByte('_')
			}
			out.WriteRune(r + 32)
		} else {
			out.WriteRune(r)
		}
	}
	return out.String()
}

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
