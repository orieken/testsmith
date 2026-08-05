package csharp

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, err := findRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	rootNS := detectRootNamespace(root)
	framework, mockLib := detectCSharpFramework(root)

	return &domain.ProjectContext{
		Root:        root,
		Language:    "csharp",
		PackageMap:  map[string]string{rootNS: root},
		ExcludeDirs: []string{"bin", "obj", ".git", "node_modules", "packages"},
		Metadata:    map[string]any{"root_namespace": rootNS, "framework": framework, "mock_library": mockLib},
	}, nil
}

// detectCSharpFramework scans .csproj files for NuGet package references to
// determine the test framework and mock library in use.
func detectCSharpFramework(root string) (framework, mockLib string) {
	framework, mockLib = "xunit", "moq"

	var content strings.Builder
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".csproj") {
			return nil //nolint:nilerr
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil //nolint:nilerr
		}
		content.Write(data)
		return nil
	})

	lower := strings.ToLower(content.String())
	switch {
	case strings.Contains(lower, "mstest.testframework") ||
		strings.Contains(lower, "microsoft.visualstudio.testtools"):
		framework = "mstest"
	case strings.Contains(lower, "nunit"):
		framework = "nunit"
	}

	if strings.Contains(lower, "nsubstitute") {
		mockLib = "nsubstitute"
	}
	return
}

var csharpStopMarkers = []string{".git", ".hg", ".svn"}

func findRoot(startDir string) (string, error) {
	dir := startDir
	for {
		if dir != startDir {
			for _, stop := range csharpStopMarkers {
				if _, err := os.Stat(filepath.Join(dir, stop)); err == nil {
					return "", domain.ErrProjectNotFound
				}
			}
		}
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			name := e.Name()
			if strings.HasSuffix(name, ".csproj") || strings.HasSuffix(name, ".sln") {
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

// detectRootNamespace reads <RootNamespace> from the first .csproj found,
// then falls back to scanning .cs files for namespace declarations.
func detectRootNamespace(root string) string {
	entries, _ := os.ReadDir(root)
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".csproj") {
			if ns := readCsprojNamespace(filepath.Join(root, e.Name())); ns != "" {
				return ns
			}
		}
	}
	return scanNamespaceFromSources(root)
}

func readCsprojNamespace(csprojPath string) string {
	data, err := os.ReadFile(csprojPath)
	if err != nil {
		return ""
	}
	content := string(data)
	for _, tag := range []string{"<RootNamespace>", "<AssemblyName>"} {
		if idx := strings.Index(content, tag); idx != -1 {
			start := idx + len(tag)
			end := strings.Index(content[start:], "</")
			if end != -1 {
				return strings.TrimSpace(content[start : start+end])
			}
		}
	}
	return ""
}

func scanNamespaceFromSources(root string) string {
	var ns string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".cs") {
			return nil //nolint:nilerr
		}
		if n := readNamespaceDecl(path); n != "" {
			ns = n
			return filepath.SkipAll
		}
		return nil
	})
	// Return top-level namespace component.
	if idx := strings.IndexByte(ns, '.'); idx != -1 {
		return ns[:idx]
	}
	return ns
}

func readNamespaceDecl(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "namespace ") {
			ns := strings.TrimPrefix(line, "namespace ")
			return strings.TrimSuffix(strings.TrimSpace(ns), ";")
		}
	}
	return ""
}

// deriveTestPath: Services/PaymentService.cs → Services/PaymentServiceTests.cs
func deriveTestPath(sourcePath string, _ *domain.ProjectContext) (string, error) {
	ext := filepath.Ext(sourcePath)
	base := sourcePath[:len(sourcePath)-len(ext)]
	return base + "Tests" + ext, nil
}

func deriveModulePath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	ns := readNamespaceDecl(sourcePath)
	className := filepath.Base(sourcePath)
	className = className[:len(className)-len(".cs")]
	if ns != "" {
		return ns + "." + className, nil
	}
	rootNS, _ := ctx.Metadata["root_namespace"].(string)
	if rootNS != "" {
		return rootNS + "." + className, nil
	}
	return className, nil
}
