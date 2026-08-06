package projectknowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const patternsDirName = ".testsmith/patterns"

// LoadPatterns reads all .md files from <root>/.testsmith/patterns/ and returns
// them as a merged string with a per-file section header derived from the
// filename. Returns empty string when the directory does not exist or contains
// no markdown files. Files are read in directory order (typically alphabetical).
func LoadPatterns(root string) string {
	dir := filepath.Join(root, patternsDirName)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}

	var sb strings.Builder
	first := true
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || e.Name() == "README.md" {
			continue
		}
		content := readFile(filepath.Join(dir, e.Name()))
		if content == "" {
			continue
		}
		if !first {
			sb.WriteString("\n\n")
		}
		label := strings.TrimSuffix(e.Name(), ".md")
		fmt.Fprintf(&sb, "### Pattern: %s\n\n%s", label, content)
		first = false
	}
	return strings.TrimSpace(sb.String())
}

// PatternsDir returns the absolute path to the patterns directory for root.
func PatternsDir(root string) string {
	return filepath.Join(root, patternsDirName)
}
