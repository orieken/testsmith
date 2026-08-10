package golang

import (
	"path/filepath"
	"strings"

	"github.com/orieken/assay/internal/domain"
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

// packageName returns the Go package name for a source file (last dir segment).
func packageName(sourcePath string) string {
	dir := filepath.Base(filepath.Dir(sourcePath))
	return strings.ReplaceAll(dir, "-", "_")
}

func indentLines(lines []string, tabs int) string {
	prefix := strings.Repeat("\t", tabs)
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
