package golang

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, modName, err := findModuleRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	pkgMap := scanPackages(root, modName)
	mockLib := detectGoMockLib(filepath.Join(root, "go.mod"))

	return &domain.ProjectContext{
		Root:        root,
		Language:    "go",
		PackageMap:  pkgMap,
		ExcludeDirs: []string{"vendor", "testdata", ".git"},
		Metadata:    map[string]any{"module": modName, "framework": "testing", "mock_library": mockLib},
	}, nil
}

// detectGoMockLib reads go.mod and returns the mock library name based on
// which testing helpers are required.
func detectGoMockLib(gomodPath string) string {
	data, err := os.ReadFile(gomodPath)
	if err != nil {
		return "interfaces"
	}
	content := string(data)
	switch {
	case strings.Contains(content, "go.uber.org/mock"):
		return "gomock"
	case strings.Contains(content, "github.com/stretchr/testify"):
		return "testify"
	default:
		return "interfaces"
	}
}

// findModuleRoot walks up from startDir looking for go.mod and returns
// (root, moduleName, error).
func findModuleRoot(startDir string) (string, string, error) {
	dir := startDir
	for {
		modFile := filepath.Join(dir, "go.mod")
		if data, err := os.ReadFile(modFile); err == nil {
			mod := parseModuleName(data)
			if mod != "" {
				return dir, mod, nil
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

func parseModuleName(data []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// scanPackages returns a map of package-import-path → local-dir for every
// directory under root that contains at least one .go file.
func scanPackages(root, modName string) map[string]string {
	pkgMap := make(map[string]string)
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		name := d.Name()
		switch name {
		case "vendor", "testdata", ".git", "node_modules":
			return filepath.SkipDir
		}
		entries, _ := os.ReadDir(path)
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") &&
				!strings.HasSuffix(e.Name(), "_test.go") {
				rel, _ := filepath.Rel(root, path)
				if rel == "." {
					pkgMap[modName] = path
				} else {
					pkgMap[modName+"/"+filepath.ToSlash(rel)] = path
				}
				break
			}
		}
		return nil
	})
	return pkgMap
}

// deriveTestPath produces foo_test.go alongside foo.go.
func deriveTestPath(sourcePath string, _ *domain.ProjectContext) (string, error) {
	ext := filepath.Ext(sourcePath)
	base := sourcePath[:len(sourcePath)-len(ext)]
	return base + "_test" + ext, nil
}

// deriveModulePath returns the Go package import path for a source file.
func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, filepath.Dir(sourcePath))
	if err != nil {
		return "", err
	}
	mod, _ := ctx.Metadata["module"].(string)
	if rel == "." {
		return mod, nil
	}
	return mod + "/" + filepath.ToSlash(rel), nil
}
