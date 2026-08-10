package typescript

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
	r.SetDefault(&jestAdapter{})
	r.Register(&vitestAdapter{})
	r.Register(&mochaSinonAdapter{})
	return r
}

func selectAdapter(ctx *domain.ProjectContext) domain.TestAdapter {
	return registry.SelectFromContext(ctx)
}

// ── shared render data ────────────────────────────────────────────────────────

type renderData struct {
	ImportPath string
	ModuleName string
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

func newRenderData(analysis *domain.SourceAnalysis, opts domain.GenerateOpts, testPath string) renderData {
	importPath := moduleImportPath(testPath, analysis.SourcePath)
	return renderData{
		ImportPath: importPath,
		ModuleName: analysis.ModuleName(),
		Members:    analysis.PublicAPI,
		LLMBodies:  opts.LLMBodies,
	}
}

func renderTmpl(tmpl *template.Template, data renderData) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render ts test: %w", err)
	}
	return normaliseBlankLines(buf.String()), nil
}

var sharedFuncs = template.FuncMap{
	"join":   strings.Join,
	"indent": indentLines,
}

// ── 1. Jest (default) ─────────────────────────────────────────────────────────

type jestAdapter struct{}

func (a *jestAdapter) Framework() string   { return "jest" }
func (a *jestAdapter) MockLibrary() string { return "jest" }
func (a *jestAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "jest",
		TestFileSuffix: ".test.ts",
		FixtureDir:     "__mocks__/",
		BootstrapFile:  "jest.setup.ts",
		TestFuncPrefix: "it(",
	}
}

func (a *jestAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	testPath, _ := deriveTestPath(analysis.SourcePath, analysis.Project)
	return renderTmpl(jestTmpl, newRenderData(analysis, opts, testPath))
}

var jestTmpl = template.Must(template.New("jest").Funcs(sharedFuncs).Parse(`import { describe, it, expect, jest, beforeEach } from '@jest/globals';
import {
{{ range .Members }}  {{ .Name }},
{{ end -}}
} from '{{ .ImportPath }}';

{{ range .Members }}
{{ if eq .Kind "function" -}}
describe('{{ .Name }}', () => {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = {{ .Name }}(...)

    // Assert
    expect(true).toBe(true); // TODO: replace
  });
{{ end -}}
});

{{ else if eq .Kind "class" -}}
describe('{{ .Name }}', () => {
  let instance: {{ .Name }};

  beforeEach(() => {
    // TODO: initialise instance with jest.fn() mocks
    // instance = new {{ .Name }}(jest.fn(), jest.fn());
  });
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('{{ .Name }} should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = instance.{{ .Name }}(...)

    // Assert
    expect(true).toBe(true); // TODO: replace
  });
{{ end }}
{{ end -}}
});

{{ end -}}
{{ end }}`))

// ── 2. Vitest ─────────────────────────────────────────────────────────────────

type vitestAdapter struct{}

func (a *vitestAdapter) Framework() string   { return "vitest" }
func (a *vitestAdapter) MockLibrary() string { return "vitest" }
func (a *vitestAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "vitest",
		TestFileSuffix: ".test.ts",
		FixtureDir:     "__mocks__/",
		BootstrapFile:  "vitest.setup.ts",
		TestFuncPrefix: "it(",
	}
}

func (a *vitestAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	testPath, _ := deriveTestPath(analysis.SourcePath, analysis.Project)
	return renderTmpl(vitestTmpl, newRenderData(analysis, opts, testPath))
}

var vitestTmpl = template.Must(template.New("vitest").Funcs(sharedFuncs).Parse(`import { describe, it, expect, vi, beforeEach } from 'vitest';
import {
{{ range .Members }}  {{ .Name }},
{{ end -}}
} from '{{ .ImportPath }}';

{{ range .Members }}
{{ if eq .Kind "function" -}}
describe('{{ .Name }}', () => {
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = {{ .Name }}(...)

    // Assert
    expect(true).toBe(true); // TODO: replace
  });
{{ end -}}
});

{{ else if eq .Kind "class" -}}
describe('{{ .Name }}', () => {
  let instance: {{ .Name }};

  beforeEach(() => {
    // TODO: initialise instance with vi.fn() mocks
    // instance = new {{ .Name }}(vi.fn(), vi.fn());
  });
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('{{ .Name }} should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = instance.{{ .Name }}(...)

    // Assert
    expect(true).toBe(true); // TODO: replace
  });
{{ end }}
{{ end -}}
});

{{ end -}}
{{ end }}`))

// ── 3. Mocha + Sinon ──────────────────────────────────────────────────────────

type mochaSinonAdapter struct{}

func (a *mochaSinonAdapter) Framework() string   { return "mocha" }
func (a *mochaSinonAdapter) MockLibrary() string { return "sinon" }
func (a *mochaSinonAdapter) FrameworkConfig() domain.TestFrameworkConfig {
	return domain.TestFrameworkConfig{
		Name:           "mocha",
		TestFileSuffix: ".test.ts",
		FixtureDir:     "test/fixtures/",
		BootstrapFile:  "",
		TestFuncPrefix: "it(",
	}
}

func (a *mochaSinonAdapter) GenerateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (string, error) {
	testPath, _ := deriveTestPath(analysis.SourcePath, analysis.Project)
	return renderTmpl(mochaSinonTmpl, newRenderData(analysis, opts, testPath))
}

var mochaSinonTmpl = template.Must(template.New("mocha-sinon").Funcs(sharedFuncs).Parse(`import { describe, it, beforeEach, afterEach } from 'mocha';
import { expect } from 'chai';
import * as sinon from 'sinon';
import {
{{ range .Members }}  {{ .Name }},
{{ end -}}
} from '{{ .ImportPath }}';

{{ range .Members }}
{{ if eq .Kind "function" -}}
describe('{{ .Name }}', () => {
  let sandbox: sinon.SinonSandbox;

  beforeEach(() => { sandbox = sinon.createSandbox(); });
  afterEach(() => { sandbox.restore(); });
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = {{ .Name }}(...)

    // Assert
    expect(true).to.equal(true); // TODO: replace
  });
{{ end -}}
});

{{ else if eq .Kind "class" -}}
describe('{{ .Name }}', () => {
  let sandbox: sinon.SinonSandbox;
  let instance: {{ .Name }};

  beforeEach(() => {
    sandbox = sinon.createSandbox();
    // TODO: initialise instance with sinon stubs
    // const depStub = sandbox.stub() as sinon.SinonStub;
    // instance = new {{ .Name }}(depStub);
  });

  afterEach(() => { sandbox.restore(); });
{{ range ($.PublicMethods .) }}
{{ $body := $.BodyFor .Name -}}
{{ if $body }}
{{ indent $body 2 }}
{{ else }}
  it('{{ .Name }} should work correctly', () => {
    // Arrange
    // TODO: arrange

    // Act
    // const result = instance.{{ .Name }}(...)

    // Assert
    expect(true).to.equal(true); // TODO: replace
  });
{{ end }}
{{ end -}}
});

{{ end -}}
{{ end }}`))

// ── LLMVocabulary ─────────────────────────────────────────────────────────────

func (a *jestAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "jest",
		"mock_library": "jest",
		"assert_style": "expect(value).toBe(expected)",
		"mock_style":   "jest.fn() / jest.mock('module')",
		"import_style": "import { describe, it, expect } from '@jest/globals'",
	}
}

func (a *vitestAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "vitest",
		"mock_library": "vitest",
		"assert_style": "expect(value).toBe(expected)",
		"mock_style":   "vi.fn() / vi.mock('module')",
		"import_style": "import { describe, it, expect, vi } from 'vitest'",
	}
}

func (a *mochaSinonAdapter) LLMVocabulary() map[string]string {
	return map[string]string{
		"framework":    "mocha",
		"mock_library": "sinon",
		"assert_style": "expect(value).to.equal(expected)",
		"mock_style":   "sinon.stub(obj, 'method') / sinon.spy()",
		"import_style": "import { expect } from 'chai'; import sinon from 'sinon'",
	}
}
