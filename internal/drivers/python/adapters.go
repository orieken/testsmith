package python

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/orieken/testsmith/internal/domain"
)

// ── registry ─────────────────────────────────────────────────────────────────

var registry = buildRegistry()

func buildRegistry() *domain.AdapterRegistry {
	r := domain.NewAdapterRegistry()
	r.SetDefault(&pytestPytestMockAdapter{})
	r.Register(&pytestUnittestMockAdapter{})
	r.Register(&unittestAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared render data ────────────────────────────────────────────────────────

type renderData struct {
	ModuleName    string
	ModulePath    string
	Members       []domain.PublicMember
	FixtureParams []string
	LLMBodies     map[string][]string
}

func (d renderData) PublicMethods(m domain.PublicMember) []domain.MethodInfo {
	var out []domain.MethodInfo
	for _, method := range m.Methods {
		if method.IsPublic && method.Name != "__init__" {
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
		ModuleName:    analysis.ModuleName(),
		ModulePath:    analysis.ModulePath,
		Members:       analysis.PublicAPI,
		FixtureParams: fixtureParamNames(analysis.Imports.External),
		LLMBodies:     opts.LLMBodies,
	}
}

func render(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render python test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

// ── 1. pytest + pytest-mock (default) ────────────────────────────────────────

type pytestPytestMockAdapter struct{}

func (a *pytestPytestMockAdapter) Framework() string   { return "pytest" }
func (a *pytestPytestMockAdapter) MockLibrary() string { return "pytest-mock" }
func (a *pytestPytestMockAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "pytest",
		TestFilePrefix: "test_",
		TestFileSuffix: ".py",
		FixtureDir:     "tests/fixtures/",
		FixtureSuffix:  "_fixture.py",
		BootstrapFile:  "conftest.py",
		TestFuncPrefix: "test_",
	}
}

func (a *pytestPytestMockAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return render(pytestPytestMockTmpl, newRenderData(analysis, opts))
}

var pytestPytestMockTmpl = template.Must(template.New("pytest-pytestmock").Funcs(template.FuncMap{
	"join":      strings.Join,
	"title":     strings.Title, //nolint:staticcheck
	"hasPrefix": strings.HasPrefix,
	"indent":    indentLines,
}).Parse(`"""Tests for {{ .ModuleName }} module."""
import pytest
{{ range .FixtureParams }}
from tests.fixtures.{{ . }}_fixture import mock_{{ . }}
{{ end }}

{{ range .Members }}
{{ if eq .Kind "function" -}}
class Test{{ title .Name }}:
    """Tests for {{ .Name }}."""
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self{{ range $.FixtureParams }}, mock_{{ . }}{{ end }}):
        # Arrange
        # TODO: arrange

        # Act
        # result = {{ .Name }}(...)

        # Assert
        pass
{{ end }}
{{ else if eq .Kind "class" -}}
class Test{{ .Name }}:
    """Tests for {{ .Name }}."""

    @pytest.fixture(autouse=True)
    def setup(self{{ range $.FixtureParams }}, mock_{{ . }}{{ end }}):
        self.sut = {{ .Name }}({{ range $.FixtureParams }}mock_{{ . }}, {{ end }})
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self):
        # Arrange
        # TODO: arrange

        # Act
        # result = self.sut.{{ .Name }}(...)

        # Assert
        pass
{{ end }}
{{ end }}
{{ end }}
{{ end }}`))

// ── 2. pytest + unittest.mock ─────────────────────────────────────────────────

type pytestUnittestMockAdapter struct{}

func (a *pytestUnittestMockAdapter) Framework() string   { return "pytest" }
func (a *pytestUnittestMockAdapter) MockLibrary() string { return "unittest.mock" }
func (a *pytestUnittestMockAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return a.toBase().FrameworkConfig()
}
func (a *pytestUnittestMockAdapter) toBase() *pytestPytestMockAdapter {
	return &pytestPytestMockAdapter{}
}

func (a *pytestUnittestMockAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return render(pytestUnittestMockTmpl, newRenderData(analysis, opts))
}

var pytestUnittestMockTmpl = template.Must(template.New("pytest-unittestmock").Funcs(template.FuncMap{
	"join":   strings.Join,
	"title":  strings.Title, //nolint:staticcheck
	"indent": indentLines,
}).Parse(`"""Tests for {{ .ModuleName }} module."""
import pytest
from unittest.mock import MagicMock, patch
{{ range .FixtureParams }}
@pytest.fixture
def mock_{{ . }}():
    return MagicMock()
{{ end }}

{{ range .Members }}
{{ if eq .Kind "function" -}}
class Test{{ title .Name }}:
    """Tests for {{ .Name }}."""
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self{{ range $.FixtureParams }}, mock_{{ . }}{{ end }}):
        # Arrange
        # TODO: arrange

        # Act
        # result = {{ .Name }}(...)

        # Assert
        pass
{{ end }}
{{ else if eq .Kind "class" -}}
class Test{{ .Name }}:
    """Tests for {{ .Name }}."""
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self{{ range $.FixtureParams }}, mock_{{ . }}{{ end }}):
        # Arrange
        sut = {{ $.Members | len | printf "" }}{{ .Name | printf "" }}{{ $.ModuleName | printf "" }}
        # sut = <ClassName>({{ range $.FixtureParams }}mock_{{ . }}, {{ end }})

        # Act
        # result = sut.{{ .Name }}(...)

        # Assert
        pass
{{ end }}
{{ end }}
{{ end }}
{{ end }}`))

// ── 3. unittest + unittest.mock ───────────────────────────────────────────────

type unittestAdapter struct{}

func (a *unittestAdapter) Framework() string   { return "unittest" }
func (a *unittestAdapter) MockLibrary() string { return "unittest.mock" }
func (a *unittestAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "unittest",
		TestFilePrefix: "test_",
		TestFileSuffix: ".py",
		FixtureDir:     "tests/",
		FixtureSuffix:  "_test.py",
		BootstrapFile:  "",
		TestFuncPrefix: "test_",
	}
}

func (a *unittestAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	return render(unittestTmpl, newRenderData(analysis, opts))
}

var unittestTmpl = template.Must(template.New("unittest").Funcs(template.FuncMap{
	"join":   strings.Join,
	"title":  strings.Title, //nolint:staticcheck
	"indent": indentLines,
}).Parse(`"""Tests for {{ .ModuleName }} module."""
import unittest
from unittest.mock import MagicMock, patch

{{ range .Members }}
{{ if eq .Kind "function" -}}
class Test{{ title .Name }}(unittest.TestCase):
    """Tests for {{ .Name }}."""
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self):
        # Arrange
        # TODO: arrange

        # Act
        # result = {{ .Name }}(...)

        # Assert
        self.assertTrue(True)  # TODO: replace
{{ end }}
{{ else if eq .Kind "class" -}}
class Test{{ .Name }}(unittest.TestCase):
    """Tests for {{ .Name }}."""

    def setUp(self):
        {{ range $.FixtureParams -}}
        self.mock_{{ . }} = MagicMock()
        {{ end -}}
        self.sut = None  # TODO: initialise {{ .Name }}
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 4 }}
{{ else }}
    def test_{{ .Name }}(self):
        # Arrange
        # TODO: arrange

        # Act
        # result = self.sut.{{ .Name }}(...)

        # Assert
        self.assertTrue(True)  # TODO: replace
{{ end }}
{{ end }}
{{ end }}
{{ end }}

if __name__ == "__main__":
    unittest.main()`))

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

func (a *pytestPytestMockAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":         "pytest",
		"mock_library":      "pytest-mock",
		"assert_style":      "assert value == expected",
		"mock_style":        "mocker.patch('module.ClassName')",
		"fixture_decorator": "@pytest.fixture",
	}
}

func (a *pytestUnittestMockAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":         "pytest",
		"mock_library":      "unittest.mock",
		"assert_style":      "assert value == expected",
		"mock_style":        "@patch('module.ClassName')",
		"fixture_decorator": "@patch",
	}
}

func (a *unittestAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":         "unittest",
		"mock_library":      "unittest.mock",
		"assert_style":      "self.assertEqual(expected, value)",
		"mock_style":        "@patch('module.ClassName')",
		"fixture_decorator": "@patch",
	}
}
