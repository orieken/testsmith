package python

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/assay/internal/domain"
)

const conftestTemplate = `import sys
from pathlib import Path

# Paths managed by Assay — do not edit this list manually.
paths_to_add = [
]


def pytest_configure(config):
    for p in paths_to_add:
        path = Path(__file__).parent / p
        sys.path.insert(0, str(path))
`

// generateBootstrap creates or updates conftest.py with the paths required
// for the test files in the plan to import their source modules.
func generateBootstrap(plan *domain.GenerationPlan, ctx *domain.ProjectContext) (*domain.GeneratedFile, error) {
	conftestPath := filepath.Join(ctx.Root, "conftest.py")

	// Collect paths we need to add based on the test files in the plan.
	required := computeRequiredPaths(plan, ctx)
	if len(required) == 0 {
		return nil, nil
	}

	// Read existing conftest or start fresh.
	var existing string
	if data, err := os.ReadFile(conftestPath); err == nil {
		existing = string(data)
	} else {
		existing = conftestTemplate
	}

	updated, changed := injectPaths(existing, required)
	if !changed {
		return &domain.GeneratedFile{
			AbsPath: conftestPath,
			Content: existing,
			Action:  domain.ActionSkip,
			Role:    domain.RoleBootstrap,
		}, nil
	}

	return &domain.GeneratedFile{
		AbsPath: conftestPath,
		Content: updated,
		Action:  domain.ActionUpdate,
		Role:    domain.RoleBootstrap,
	}, nil
}

// computeRequiredPaths derives the set of relative paths that conftest.py
// must add to sys.path so that test imports resolve.
func computeRequiredPaths(plan *domain.GenerationPlan, ctx *domain.ProjectContext) []string {
	seen := make(map[string]bool)
	var paths []string

	for _, f := range plan.Files {
		if f.Role != domain.RoleTestFile || f.Action == domain.ActionSkip {
			continue
		}
		// src/services/payment.py -> add "src" to sys.path
		rel, err := filepath.Rel(ctx.Root, f.AbsPath)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		// The source directory is the tests/ mirror reversed back to src/.
		// e.g. tests/src/services/test_payment.py -> src/services
		srcDir := mirrorTestPathToSrcDir(rel, ctx.Root)
		if srcDir != "" && !seen[srcDir] {
			seen[srcDir] = true
			paths = append(paths, srcDir)
		}
		// Always ensure tests/ and tests/fixtures/ are on the path.
		for _, p := range []string{"tests", "tests/fixtures"} {
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	return paths
}

// injectPaths inserts any missing entries into the paths_to_add list in conftest.py.
// Returns the updated content and whether any change was made.
func injectPaths(content string, required []string) (string, bool) {
	// Find paths already present.
	existing := extractExistingPaths(content)
	var missing []string
	for _, p := range required {
		found := false
		for _, e := range existing {
			if e == p {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, p)
		}
	}
	if len(missing) == 0 {
		return content, false
	}

	// Build injection lines.
	var newEntries strings.Builder
	for _, p := range missing {
		newEntries.WriteString(fmt.Sprintf("    %q,\n", p))
	}

	// Find the closing bracket of paths_to_add and insert before it.
	listStart := strings.Index(content, "paths_to_add = [")
	if listStart == -1 {
		// paths_to_add not found — append a new declaration before pytest_configure.
		hookIdx := strings.Index(content, "def pytest_configure")
		if hookIdx == -1 {
			content += "\npaths_to_add = [\n" + newEntries.String() + "]\n"
		} else {
			insert := "paths_to_add = [\n" + newEntries.String() + "]\n\n"
			content = content[:hookIdx] + insert + content[hookIdx:]
		}
		return content, true
	}

	closingBracket := strings.Index(content[listStart:], "\n]")
	if closingBracket == -1 {
		return content, false
	}
	insertAt := listStart + closingBracket + 1 // position of the \n before ]
	content = content[:insertAt] + newEntries.String() + content[insertAt:]
	return content, true
}

// extractExistingPaths parses the quoted strings inside paths_to_add = [...].
func extractExistingPaths(content string) []string {
	var paths []string
	start := strings.Index(content, "paths_to_add = [")
	if start == -1 {
		return paths
	}
	end := strings.Index(content[start:], "\n]")
	if end == -1 {
		return paths
	}
	block := content[start : start+end]
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		line = strings.Trim(line, `",`)
		line = strings.Trim(line, `'`)
		if line != "" && !strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "paths_to_add") {
			paths = append(paths, line)
		}
	}
	return paths
}

// mirrorTestPathToSrcDir converts a test file path back to its source directory.
// e.g. "tests/src/services/test_payment.py" -> "src/services"
func mirrorTestPathToSrcDir(testRel, root string) string {
	_ = root
	// Strip "tests/" prefix.
	after, ok := strings.CutPrefix(testRel, "tests/")
	if !ok {
		return ""
	}
	// Strip the test filename to get the directory.
	dir := filepath.Dir(after)
	if dir == "." {
		return ""
	}
	return filepath.ToSlash(dir)
}
