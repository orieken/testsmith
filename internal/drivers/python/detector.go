package python

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

// pythonRootMarkers are files that positively identify a Python project root.
var pythonRootMarkers = []string{"pyproject.toml", "setup.py", "setup.cfg", "conftest.py"}

// searchStopMarkers halt upward traversal when no Python marker was found in a
// directory — they prevent crossing a git/VCS boundary into a parent project.
var searchStopMarkers = []string{".git", ".hg", ".svn"}

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, err := findRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	pkgMap, err := scanPackages(root)
	if err != nil {
		return nil, err
	}

	framework, mockLib := detectPythonFramework(root)

	ctx := &domain.ProjectContext{
		Root:        root,
		Language:    "python",
		PackageMap:  pkgMap,
		ExcludeDirs: []string{"__pycache__", ".venv", "venv", "build", "dist", ".eggs", ".tox"},
		Metadata:    map[string]any{"framework": framework, "mock_library": mockLib},
	}

	// Record conftest.py path if it exists.
	conftest := filepath.Join(root, "conftest.py")
	if _, err := os.Stat(conftest); err == nil {
		ctx.Metadata["conftest_path"] = conftest
	}

	return ctx, nil
}

// detectPythonFramework scans pyproject.toml and requirements files for known
// test dependencies and returns the best-matching framework+mock_library pair.
func detectPythonFramework(root string) (framework, mockLib string) {
	framework, mockLib = "pytest", "pytest-mock"

	deps := collectPythonDeps(root)
	hasPytest := deps["pytest"]
	hasPytestMock := deps["pytest-mock"]
	hasUnittest := deps["unittest"] || deps["unittest2"]

	switch {
	case hasUnittest && !hasPytest:
		return "unittest", "unittest.mock"
	case hasPytest && !hasPytestMock:
		return "pytest", "unittest.mock"
	}
	return framework, mockLib
}

// collectPythonDeps returns a set of normalized package names found in
// pyproject.toml, requirements*.txt, or setup.cfg.
func collectPythonDeps(root string) map[string]bool {
	deps := make(map[string]bool)

	candidates := []string{
		"pyproject.toml",
		"requirements.txt",
		"requirements-dev.txt",
		"requirements-test.txt",
		"setup.cfg",
	}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		for _, pkg := range []string{"pytest", "pytest-mock", "unittest", "unittest2"} {
			if strings.Contains(content, pkg) {
				deps[pkg] = true
			}
		}
	}
	return deps
}

func findRoot(startDir string) (string, error) {
	dir := startDir
	for {
		// Stop at VCS boundaries when searching ancestor directories so we do not
		// claim a parent project's root. At startDir itself we skip this check
		// because a Python project root may legitimately host .git at the same level.
		if dir != startDir {
			for _, stop := range searchStopMarkers {
				if _, err := os.Stat(filepath.Join(dir, stop)); err == nil {
					return "", domain.ErrProjectNotFound
				}
			}
		}
		for _, marker := range pythonRootMarkers {
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

// scanPackages walks root looking for Python packages (directories with __init__.py)
// and returns a map of package-name -> absolute-path.
func scanPackages(root string) (map[string]string, error) {
	pkgMap := make(map[string]string)

	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil //nolint:nilerr
		}
		name := d.Name()
		// Skip common non-package directories.
		switch name {
		case "__pycache__", ".venv", "venv", "build", "dist", ".git", "node_modules", ".eggs", ".tox":
			return filepath.SkipDir
		}

		initPath := filepath.Join(path, "__init__.py")
		if _, err := os.Stat(initPath); err == nil {
			// Strip src/ prefix common in src-layout projects.
			rel, _ := filepath.Rel(root, path)
			parts := splitPath(rel)
			if len(parts) > 0 && parts[0] == "src" {
				parts = parts[1:]
			}
			if len(parts) > 0 {
				pkgMap[parts[0]] = filepath.Join(root, filepath.Join(parts[:1]...))
			}
		}
		return nil
	})
	return pkgMap, err
}

func splitPath(p string) []string {
	var parts []string
	for {
		dir, file := filepath.Split(filepath.Clean(p))
		if file == "" || file == "." {
			break
		}
		parts = append([]string{file}, parts...)
		p = filepath.Clean(dir)
		if p == "." || p == "/" {
			break
		}
	}
	return parts
}

func deriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		return "", err
	}

	// Normalise to forward slashes for the prefix check so this works on
	// Windows (where filepath.Rel returns backslash-separated paths).
	slashed := strings.TrimPrefix(filepath.ToSlash(rel), "src/")
	rel = filepath.FromSlash(slashed)

	dir := filepath.Dir(rel)
	base := filepath.Base(rel)
	testBase := "test_" + base

	testPath := filepath.Join(ctx.Root, "tests", dir, testBase)
	return testPath, nil
}

func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		return "", err
	}

	// Normalise to forward slashes for the prefix check (Windows compatibility).
	slashed := filepath.ToSlash(rel)
	slashed = strings.TrimPrefix(slashed, "src/")
	rel = filepath.FromSlash(slashed)

	// Remove .py suffix and replace separators with dots.
	module := rel[:len(rel)-len(filepath.Ext(rel))]
	result := ""
	for _, c := range module {
		if c == filepath.Separator || c == '/' {
			result += "."
		} else {
			result += string(c)
		}
	}
	return result, nil
}
