package kotlin

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

var (
	rootMarkers = []string{"build.gradle.kts", "build.gradle", "pom.xml", "settings.gradle.kts", "settings.gradle"}
	stopMarkers = []string{".git", ".hg", ".svn"}
)

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, buildFile, err := findProjectRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	moduleName, framework, mockLib := readBuildFile(root, buildFile)

	return &domain.ProjectContext{
		Root:        root,
		Language:    "kotlin",
		PackageMap:  map[string]string{moduleName: root},
		ExcludeDirs: []string{"build", ".gradle", ".idea", "target", ".git"},
		Metadata: map[string]any{
			"module":     moduleName,
			"build_file": buildFile,
			"framework":  framework,
			"mock_lib":   mockLib,
		},
	}, nil
}

func findProjectRoot(startDir string) (string, string, error) {
	dir := startDir
	for {
		if dir != startDir {
			for _, stop := range stopMarkers {
				if _, err := os.Stat(filepath.Join(dir, stop)); err == nil {
					return "", "", domain.ErrProjectNotFound
				}
			}
		}

		for _, marker := range rootMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir, marker, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", "", domain.ErrProjectNotFound
}

func readBuildFile(root, buildFile string) (moduleName, framework, mockLib string) {
	moduleName = filepath.Base(root)
	framework = "junit5"
	mockLib = "mockk"

	data, err := os.ReadFile(filepath.Join(root, buildFile))
	if err != nil {
		return
	}

	content := strings.ToLower(string(data))

	switch {
	case strings.Contains(content, "io.kotest"):
		framework = "kotest"
	case strings.Contains(content, "junit-jupiter") || strings.Contains(content, "junit5"):
		framework = "junit5"
	}

	switch {
	case strings.Contains(content, "io.mockk"):
		mockLib = "mockk"
	case strings.Contains(content, "mockito"):
		mockLib = "mockito"
	}

	moduleName = parseModuleName(root, buildFile)
	return
}

func parseModuleName(root, buildFile string) string {
	// Try settings.gradle.kts or settings.gradle for rootProject.name.
	for _, settings := range []string{"settings.gradle.kts", "settings.gradle"} {
		data, err := os.ReadFile(filepath.Join(root, settings))
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(strings.NewReader(string(data)))
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if strings.Contains(line, "rootProject.name") {
				if idx := strings.Index(line, `"`); idx != -1 {
					rest := line[idx+1:]
					if end := strings.Index(rest, `"`); end != -1 {
						return rest[:end]
					}
				}
			}
		}
	}
	// Fall back to directory name.
	_ = buildFile
	return filepath.Base(root)
}
