package golang

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
	r.SetDefault(&stdlibAdapter{})
	r.Register(&testifyAdapter{})
	r.Register(&gomockAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared render data ────────────────────────────────────────────────────────

type renderData struct {
	PackageName string
	ImportPath  string
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
	return renderData{
		PackageName: packageName(analysis.SourcePath),
		ImportPath:  analysis.ModulePath,
		Members:     analysis.PublicAPI,
		LLMBodies:   opts.LLMBodies,
	}
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render go test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"title":  strings.Title, //nolint:staticcheck
	"indent": indentLines,
}

// ── 1. stdlib testing (default) ───────────────────────────────────────────────

type stdlibAdapter struct{}

func (a *stdlibAdapter) Framework() string   { return "testing" }
func (a *stdlibAdapter) MockLibrary() string { return "interfaces" }
func (a *stdlibAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "testing",
		TestFileSuffix: "_test.go",
		TestFuncPrefix: "Test",
	}
}

func (a *stdlibAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(stdlibTmpl, newRenderData(analysis, opts))
}

var stdlibTmpl = template.Must(template.New("go-stdlib").Funcs(sharedFuncs).Parse(`package {{ .PackageName }}_test

import (
	"testing"
)

{{ range .Members }}
{{ if eq .Kind "function" -}}
func Test{{ title .Name }}(t *testing.T) {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
		// TODO: add input/output fields
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("not implemented")
		})
	}
{{ end -}}
}

{{ else if eq .Kind "class" -}}
{{ $className := .Name -}}
{{ range ($.PublicMethods .) -}}
func Test{{ title $className }}_{{ title .Name }}(t *testing.T) {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
		// TODO: add input/output fields
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Skip("not implemented")
		})
	}
{{ end -}}
}

{{ end -}}
{{ end -}}
{{ end }}`))

// ── 2. testing + testify ──────────────────────────────────────────────────────

type testifyAdapter struct{}

func (a *testifyAdapter) Framework() string   { return "testing" }
func (a *testifyAdapter) MockLibrary() string { return "testify" }
func (a *testifyAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "testing",
		TestFileSuffix: "_test.go",
		TestFuncPrefix: "Test",
	}
}

func (a *testifyAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(testifyTmpl, newRenderData(analysis, opts))
}

var testifyTmpl = template.Must(template.New("go-testify").Funcs(sharedFuncs).Parse(`package {{ .PackageName }}_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

{{ range .Members }}
{{ if eq .Kind "function" -}}
func Test{{ title .Name }}(t *testing.T) {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
		// TODO: add input/output fields
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// TODO: arrange

			// Act
			// got := {{ .Name }}(...)

			// Assert
			assert.True(t, true) // TODO: replace
		})
	}
{{ end -}}
}

{{ else if eq .Kind "class" -}}
// Mock{{ .Name }} is a testify mock for {{ .Name }}.
// Replace with a real interface once one is defined.
type Mock{{ .Name }} struct {
	mock.Mock
}

{{ $className := .Name -}}
{{ range ($.PublicMethods .) -}}
func Test{{ title $className }}_{{ title .Name }}(t *testing.T) {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// sut := New{{ $className }}(...)

			// Act
			// got := sut.{{ .Name }}(...)

			// Assert
			assert.True(t, true) // TODO: replace
		})
	}
{{ end -}}
}

{{ end -}}
{{ end -}}
{{ end }}`))

// ── 3. testing + gomock ───────────────────────────────────────────────────────

type gomockAdapter struct{}

func (a *gomockAdapter) Framework() string   { return "testing" }
func (a *gomockAdapter) MockLibrary() string { return "gomock" }
func (a *gomockAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "testing",
		TestFileSuffix: "_test.go",
		TestFuncPrefix: "Test",
	}
}

func (a *gomockAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return renderTmpl(gomockTmpl, newRenderData(analysis, opts))
}

var gomockTmpl = template.Must(template.New("go-gomock").Funcs(sharedFuncs).Parse(`package {{ .PackageName }}_test

import (
	"testing"

	"go.uber.org/mock/gomock"
)

{{ range .Members }}
{{ if eq .Kind "function" -}}
func Test{{ title .Name }}(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// mockDep := NewMockIDependency(ctrl)
			// mockDep.EXPECT().Method().Return(...)

			// Act
			// got := {{ .Name }}(mockDep, ...)

			// Assert
			if got := true; !got {
				t.Errorf("Test{{ title .Name }}: unexpected result")
			}
		})
	}
{{ end -}}
}

{{ else if eq .Kind "class" -}}
// Run: go generate ./... to regenerate mocks for interfaces.
// //go:generate mockgen -source={{ $.PackageName }}.go -destination=mock_{{ $.PackageName }}_test.go -package={{ $.PackageName }}_test

{{ $className := .Name -}}
{{ range ($.PublicMethods .) -}}
func Test{{ title $className }}_{{ title .Name }}(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 1 }}
{{ else }}
	tests := []struct {
		name string
	}{
		{name: "TODO"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			// mockDep := NewMockIDependency(ctrl)
			// sut := New{{ $className }}(mockDep)

			// Act
			// got := sut.{{ .Name }}(...)

			// Assert
			if got := true; !got {
				t.Errorf("{{ $className }}.{{ .Name }}: unexpected result")
			}
		})
	}
{{ end -}}
}

{{ end -}}
{{ end -}}
{{ end }}`))
