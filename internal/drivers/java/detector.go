package java

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

var rootMarkers = []string{"pom.xml", "build.gradle", "build.gradle.kts"}
var stopMarkers = []string{".git", ".hg", ".svn"}

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, err := findRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	basePkg := detectBasePackage(root)
	buildSystem := detectBuildSystem(root)
	framework := detectJavaFramework(root)

	return &domain.ProjectContext{
		Root:        root,
		Language:    "java",
		PackageMap:  map[string]string{basePkg: root},
		ExcludeDirs: []string{"target", "build", ".gradle", ".mvn", ".git", "node_modules"},
		Metadata:    map[string]any{"base_package": basePkg, "build_system": buildSystem, "framework": framework, "mock_library": "mockito"},
	}, nil
}

// detectJavaFramework reads build files to determine which JUnit version (or
// TestNG / Spring Boot Test) the project uses.
func detectJavaFramework(root string) string {
	candidates := []string{"pom.xml", "build.gradle", "build.gradle.kts"}
	for _, name := range candidates {
		data, err := os.ReadFile(filepath.Join(root, name))
		if err != nil {
			continue
		}
		content := strings.ToLower(string(data))
		switch {
		case strings.Contains(content, "spring-boot-starter-test"):
			return "springboot"
		case strings.Contains(content, "testng"):
			return "testng"
		case strings.Contains(content, "junit-vintage-engine") ||
			strings.Contains(content, "junit4") ||
			strings.Contains(content, "junit:junit"):
			return "junit4"
		}
	}
	return "junit5"
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

func detectBuildSystem(root string) string {
	if _, err := os.Stat(filepath.Join(root, "pom.xml")); err == nil {
		return "maven"
	}
	return "gradle"
}

// detectBasePackage reads pom.xml for <groupId>, falling back to scanning source files.
func detectBasePackage(root string) string {
	if pkg := readGroupIDFromPom(filepath.Join(root, "pom.xml")); pkg != "" {
		return pkg
	}
	return scanBasePackageFromSources(root)
}

func readGroupIDFromPom(pomPath string) string {
	data, err := os.ReadFile(pomPath)
	if err != nil {
		return ""
	}
	content := string(data)
	const tag = "<groupId>"
	idx := strings.Index(content, tag)
	if idx == -1 {
		return ""
	}
	start := idx + len(tag)
	end := strings.Index(content[start:], "</groupId>")
	if end == -1 {
		return ""
	}
	return strings.TrimSpace(content[start : start+end])
}

func scanBasePackageFromSources(root string) string {
	var pkg string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil || d.IsDir() || !strings.HasSuffix(path, ".java") {
			return nil
		}
		if p := readPackageDecl(path); p != "" {
			pkg = p
			return filepath.SkipAll
		}
		return nil
	})
	// Return the top two components of the first package found.
	parts := strings.SplitN(pkg, ".", 3)
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return pkg
}

func readPackageDecl(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "package ") {
			return strings.TrimSuffix(strings.TrimPrefix(line, "package "), ";")
		}
	}
	return ""
}

// deriveTestPath maps src/main/java/… → src/test/java/…Test.java.
// Falls back to co-location with a Test suffix.
func deriveTestPath(sourcePath string, ctx *domain.ProjectContext) (string, error) {
	rel, err := filepath.Rel(ctx.Root, sourcePath)
	if err != nil {
		return "", err
	}

	rel = filepath.ToSlash(rel)
	if strings.HasPrefix(rel, "src/main/java/") {
		testRel := strings.Replace(rel, "src/main/java/", "src/test/java/", 1)
		base := testRel[:len(testRel)-len(".java")]
		return filepath.Join(ctx.Root, filepath.FromSlash(base+"Test.java")), nil
	}

	// Co-located fallback.
	base := sourcePath[:len(sourcePath)-len(".java")]
	return base + "Test.java", nil
}

func deriveModulePath(sourcePath string, _ *domain.ProjectContext) (string, error) {
	pkg := readPackageDecl(sourcePath)
	if pkg == "" {
		base := filepath.Base(sourcePath)
		return base[:len(base)-len(".java")], nil
	}
	className := filepath.Base(sourcePath)
	className = className[:len(className)-len(".java")]
	return pkg + "." + className, nil
}
