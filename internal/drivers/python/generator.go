package python

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/orieken/testsmith/internal/domain"
)


// generateTestFile produces a test scaffold for the given analysis using the
// selected adapter (pytest+pytest-mock by default, overrideable via config).
func generateTestFile(analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	testPath, err := deriveTestPath(analysis.SourcePath, analysis.Project)
	if err != nil {
		return nil, err
	}

	content, err := selectAdapter(analysis.Project).GenerateTestFile(analysis, opts)
	if err != nil {
		return nil, err
	}

	return &domain.GeneratedFile{
		AbsPath: testPath,
		Content: content,
		Role:    domain.RoleTestFile,
	}, nil
}

// generateFixture produces or updates tests/fixtures/<dep>_fixture.py.
func generateFixture(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	cfg := analysis.Project.LanguageConfig()
	fixtureDir := filepath.Join(analysis.Project.Root, cfg["fixture_dir"])
	fixturePath := filepath.Join(fixtureDir, dep+"_fixture.py")

	// Collect sub-modules of this dependency from the analysis.
	subModules := collectSubModules(dep, analysis.Imports.External)

	// If the fixture already exists, merge rather than overwrite.
	if existing, err := os.ReadFile(fixturePath); err == nil {
		merged, err := mergeFixture(string(existing), dep, subModules)
		if err != nil {
			return nil, err
		}
		return &domain.GeneratedFile{
			AbsPath: fixturePath,
			Content: merged,
			Role:    domain.RoleFixture,
		}, nil
	}

	content, err := renderFixtureFile(dep, subModules)
	if err != nil {
		return nil, err
	}

	return &domain.GeneratedFile{
		AbsPath: fixturePath,
		Content: content,
		Role:    domain.RoleFixture,
	}, nil
}

// ---- fixture file template --------------------------------------------------

var fixtureFileTmpl = template.Must(template.New("fixture").Funcs(template.FuncMap{
	"subAttr": func(module, root string) string {
		after, _ := strings.CutPrefix(module, root+".")
		return strings.ReplaceAll(after, ".", ".")
	},
}).Parse(`"""Shared mock fixtures for the {{ .Dep }} external dependency."""
import pytest


@pytest.fixture
def mock_{{ .Dep }}(mocker):
    """Mock for {{ .Dep }} and its sub-modules."""
    mock = mocker.Mock()
{{ range .SubModules }}{{ if ne . $.Dep }}    mock.{{ subAttr . $.Dep }} = mocker.Mock()
{{ end }}{{ end }}
    mocker.patch.dict("sys.modules", {
        "{{ .Dep }}": mock,
{{ range .SubModules }}{{ if ne . $.Dep }}        "{{ . }}": mock.{{ subAttr . $.Dep }},
{{ end }}{{ end }}    })
    return mock
`))

type fixtureData struct {
	Dep        string
	SubModules []string
}

func renderFixtureFile(dep string, subModules []string) (string, error) {
	var buf bytes.Buffer
	if err := fixtureFileTmpl.Execute(&buf, fixtureData{Dep: dep, SubModules: subModules}); err != nil {
		return "", fmt.Errorf("render fixture file: %w", err)
	}
	return buf.String(), nil
}

// mergeFixture appends any sub-modules not already present in the existing fixture content.
func mergeFixture(existing, dep string, subModules []string) (string, error) {
	var newModules []string
	for _, mod := range subModules {
		if !strings.Contains(existing, `"`+mod+`"`) {
			newModules = append(newModules, mod)
		}
	}
	if len(newModules) == 0 {
		return existing, nil
	}

	// Inject new mock attributes before the closing mocker.patch.dict call.
	var mockLines, dictLines strings.Builder
	for _, mod := range newModules {
		after := strings.TrimPrefix(mod, dep+".")
		attr := strings.ReplaceAll(after, ".", ".")
		mockLines.WriteString(fmt.Sprintf("    mock.%s = mocker.Mock()\n", attr))
		dictLines.WriteString(fmt.Sprintf("        %q: mock.%s,\n", mod, attr))
	}

	out := existing
	// Insert mock attribute lines before "mocker.patch.dict".
	if idx := strings.Index(out, `    mocker.patch.dict`); idx != -1 {
		out = out[:idx] + mockLines.String() + out[idx:]
	}
	// Insert dict entries before the closing })
	if idx := strings.LastIndex(out, "    })"); idx != -1 {
		out = out[:idx] + dictLines.String() + out[idx:]
	}
	return out, nil
}

// ---- helpers ----------------------------------------------------------------

// fixtureParamNames returns the unique root package names of external imports,
// used as pytest fixture parameter names (mock_<name>).
func fixtureParamNames(external []domain.ImportInfo) []string {
	seen := make(map[string]bool)
	var out []string
	for _, imp := range external {
		root := rootPkg(imp.Module)
		if !seen[root] {
			seen[root] = true
			out = append(out, root)
		}
	}
	return out
}

// collectSubModules returns all sub-module paths for a given root dependency.
func collectSubModules(dep string, external []domain.ImportInfo) []string {
	seen := map[string]bool{dep: true}
	out := []string{dep}
	for _, imp := range external {
		if imp.Module == dep || strings.HasPrefix(imp.Module, dep+".") {
			if !seen[imp.Module] {
				seen[imp.Module] = true
				out = append(out, imp.Module)
			}
		}
	}
	return out
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
