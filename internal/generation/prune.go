package generation

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// FixtureFile describes a discovered fixture file and the dependency it mocks.
type FixtureFile struct {
	AbsPath string
	DepName string // root package name this fixture covers
}

// PruneResult is the outcome of attempting to delete one fixture.
type PruneResult struct {
	DepName string
	Action  string // "deleted" | "skipped" | "error"
	Err     error
}

// ScanUsedDependencies returns the set of root external dependency names
// that appear in at least one source analysis.
func ScanUsedDependencies(analyses []*domain.SourceAnalysis) map[string]bool {
	used := make(map[string]bool)
	for _, a := range analyses {
		for _, imp := range a.Imports.External {
			used[rootPkgName(imp.Module)] = true
		}
	}
	return used
}

// ScanExistingFixtures discovers fixture files in fixtureDir using the
// driver's TestFrameworkConfig to determine naming conventions.
func ScanExistingFixtures(fixtureDir string, cfg domain.TestFrameworkConfig) ([]FixtureFile, error) {
	if fixtureDir == "" {
		return nil, nil
	}

	entries, err := os.ReadDir(fixtureDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan fixtures: %w", err)
	}

	suffix := cfg.FixtureSuffix
	if suffix == "" {
		suffix = "_fixture.py" // sensible default for Python
	}

	var fixtures []FixtureFile
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, suffix) {
			continue
		}
		depName := strings.TrimSuffix(name, suffix)
		fixtures = append(fixtures, FixtureFile{
			AbsPath: filepath.Join(fixtureDir, name),
			DepName: depName,
		})
	}
	return fixtures, nil
}

// IdentifyUnused returns fixtures whose dependency name is not present in used.
func IdentifyUnused(used map[string]bool, fixtures []FixtureFile) []FixtureFile {
	var unused []FixtureFile
	for _, f := range fixtures {
		if !used[f.DepName] {
			unused = append(unused, f)
		}
	}
	return unused
}

// PruneFixtures deletes unused fixture files. When dryRun is true the files
// are listed but not deleted.
func PruneFixtures(unused []FixtureFile, dryRun bool) []PruneResult {
	results := make([]PruneResult, 0, len(unused))
	for _, f := range unused {
		if dryRun {
			results = append(results, PruneResult{DepName: f.DepName, Action: "skipped"})
			continue
		}
		err := os.Remove(f.AbsPath)
		if err != nil {
			results = append(results, PruneResult{DepName: f.DepName, Action: "error", Err: err})
		} else {
			results = append(results, PruneResult{DepName: f.DepName, Action: "deleted"})
		}
	}
	return results
}

// UpdateTestImports scans all test files under root and comments out import
// lines that reference any of the deleted fixture names.
func UpdateTestImports(root string, deletedNames []string) ([]string, error) {
	if len(deletedNames) == 0 {
		return nil, nil
	}

	// Build a set of substrings to match against import lines.
	patterns := make([]string, len(deletedNames))
	for i, n := range deletedNames {
		patterns[i] = n + "_fixture"
	}

	var modified []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if !isTestFile(path) {
			return nil
		}
		changed, err := commentOutImports(path, patterns)
		if err != nil {
			return nil // non-fatal
		}
		if changed {
			modified = append(modified, path)
		}
		return nil
	})
	return modified, err
}

// commentOutImports rewrites a file, commenting out any import line that
// contains one of the given patterns. Returns true if the file was changed.
func commentOutImports(path string, patterns []string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}

	var lines []string
	changed := false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if isImportLine(line) && matchesAny(line, patterns) {
			lines = append(lines, "# [testsmith-pruned] "+line)
			changed = true
		} else {
			lines = append(lines, line)
		}
	}
	f.Close()
	if err := scanner.Err(); err != nil {
		return false, err
	}

	if !changed {
		return false, nil
	}

	return true, os.WriteFile(path, []byte(strings.Join(lines, "\n")+"\n"), 0o644)
}

func isTestFile(path string) bool {
	base := filepath.Base(path)
	return strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") ||
		strings.HasSuffix(base, ".test.ts") || strings.HasSuffix(base, "_test.go") ||
		strings.HasSuffix(base, "Tests.cs") || strings.HasSuffix(base, "Test.java")
}

func isImportLine(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "import ") || strings.HasPrefix(t, "from ")
}

func matchesAny(line string, patterns []string) bool {
	for _, p := range patterns {
		if strings.Contains(line, p) {
			return true
		}
	}
	return false
}

func rootPkgName(module string) string {
	if idx := strings.IndexAny(module, "./"); idx != -1 {
		return module[:idx]
	}
	return module
}
