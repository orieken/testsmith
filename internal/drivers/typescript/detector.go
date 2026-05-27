package typescript

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/orieken/testsmith/internal/domain"
)

var rootMarkers = []string{"package.json", "tsconfig.json", "tsconfig.base.json"}
var stopMarkers = []string{".git", ".hg", ".svn"}

type packageJSON struct {
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
}

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, err := findRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	pkgMap, framework, mockLib := readPackageJSON(root)

	return &domain.ProjectContext{
		Root:        root,
		Language:    "typescript",
		PackageMap:  pkgMap,
		ExcludeDirs: []string{"node_modules", "dist", "build", ".next", "coverage", ".turbo"},
		Metadata:    map[string]any{"framework": framework, "mock_library": mockLib},
	}, nil
}

func findRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if dir != startDir {
			for _, stop := range stopMarkers {
				if _, err := os.Stat(filepath.Join(dir, stop)); err == nil {
					return "", domain.ErrProjectNotFound
				}
			}
		}
		for _, marker := range rootMarkers {
			if _, err := os.Stat(filepath.Join(dir, marker)); err == nil {
				return dir, nil
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", domain.ErrProjectNotFound
}

// readPackageJSON returns the full dependency map and the detected test framework+mock_library.
func readPackageJSON(root string) (pkgMap map[string]string, framework, mockLib string) {
	pkgMap = make(map[string]string)
	framework = "jest"
	mockLib = "jest"

	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}

	var pkg packageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return
	}

	allDeps := make(map[string]bool)
	for name := range pkg.Dependencies {
		pkgMap[name] = name
		allDeps[name] = true
	}
	for name := range pkg.DevDependencies {
		pkgMap[name] = name
		allDeps[name] = true
	}

	switch {
	case allDeps["vitest"]:
		framework, mockLib = "vitest", "vitest"
	case allDeps["mocha"]:
		framework = "mocha"
		if allDeps["sinon"] {
			mockLib = "sinon"
		}
	}
	return
}

// deriveTestPath co-locates the test file alongside the source file.
// src/services/payment.ts → src/services/payment.test.ts
func deriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	ext := filepath.Ext(sourcePath)
	base := sourcePath[:len(sourcePath)-len(ext)]
	return base + ".test" + ext, nil
}

func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(rel)
	return rel[:len(rel)-len(ext)], nil
}
