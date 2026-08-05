package kotlin

import (
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
		AbsPath:  testPath,
		Content:  content,
		Role:     domain.RoleTestFile,
		Language: "kotlin",
	}, nil
}
