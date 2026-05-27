package typescript

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

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

func generateMock(dep string, analysis *domain.SourceAnalysis, opts domain.GenerateOpts) (*domain.GeneratedFile, error) {
	cfg := analysis.Project.LanguageConfig()
	mockDir := filepath.Join(analysis.Project.Root, cfg["fixture_dir"])
	if mockDir == "" {
		mockDir = filepath.Join(analysis.Project.Root, "__mocks__")
	}
	mockPath := filepath.Join(mockDir, dep+".ts")

	content := renderMockFile(dep)

	return &domain.GeneratedFile{
		AbsPath: mockPath,
		Content: content,
		Role:    domain.RoleFixture,
	}, nil
}

// ---- helpers -----------------------------------------------------------------

func moduleImportPath(testPath, sourcePath string) string {
	dir := filepath.Dir(testPath)
	rel, err := filepath.Rel(dir, sourcePath)
	if err != nil {
		rel = filepath.Base(sourcePath)
	}
	// Strip extension.
	ext := filepath.Ext(rel)
	rel = rel[:len(rel)-len(ext)]
	// Ensure leading ./
	rel = filepath.ToSlash(rel)
	if !strings.HasPrefix(rel, ".") {
		rel = "./" + rel
	}
	return rel
}

// ---- mock file template ------------------------------------------------------

func renderMockFile(dep string) string {
	safe := strings.ReplaceAll(dep, "-", "_")
	safe = strings.ReplaceAll(safe, "/", "_")
	safe = strings.ReplaceAll(safe, "@", "")

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("// Auto-generated Jest manual mock for '%s'\n", dep))
	sb.WriteString(fmt.Sprintf("const mock%s = {\n", titleCase(safe)))
	sb.WriteString("  // Add mock implementations here\n")
	sb.WriteString("};\n\n")
	sb.WriteString(fmt.Sprintf("export default mock%s;\n", titleCase(safe)))
	sb.WriteString(fmt.Sprintf("module.exports = mock%s;\n", titleCase(safe)))
	return sb.String()
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
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
