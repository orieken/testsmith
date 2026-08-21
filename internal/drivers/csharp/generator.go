package csharp

import (
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

func csNamespace(modulePath string) string {
	if idx := strings.LastIndex(modulePath, "."); idx != -1 {
		return modulePath[:idx]
	}
	return modulePath
}

func indentLines(lines []string, spaces int) string {
	prefix := strings.Repeat("    ", spaces)
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
