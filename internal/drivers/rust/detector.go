package rust

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

var stopMarkers = []string{".git", ".hg", ".svn"}

func detectProject(startDir string) (*domain.ProjectContext, error) {
	root, crateName, err := findCrateRoot(startDir)
	if err != nil {
		return nil, domain.ErrProjectNotFound
	}

	edition := detectEdition(filepath.Join(root, "Cargo.toml"))
	testFramework := detectTestFramework(root)

	return &domain.ProjectContext{
		Root:        root,
		Language:    "rust",
		PackageMap:  map[string]string{crateName: root},
		ExcludeDirs: []string{"target", ".git", "node_modules"},
		Metadata: map[string]any{
			"crate":      crateName,
			"edition":    edition,
			"framework":  testFramework,
			"mock_style": "mockall",
		},
	}, nil
}

// findCrateRoot walks up from startDir looking for Cargo.toml.
func findCrateRoot(startDir string) (string, string, error) {
	dir := startDir
	for {
		if dir != startDir {
			for _, stop := range stopMarkers {
				if _, err := os.Stat(filepath.Join(dir, stop)); err == nil {
					return "", "", domain.ErrProjectNotFound
				}
			}
		}

		cargo := filepath.Join(dir, "Cargo.toml")
		if data, err := os.ReadFile(cargo); err == nil {
			name := parseCrateName(data)
			if name != "" {
				return dir, name, nil
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

func parseCrateName(data []byte) string {
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inPackage := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[package]" {
			inPackage = true
			continue
		}
		if inPackage && strings.HasPrefix(line, "[") {
			break
		}
		if inPackage && strings.HasPrefix(line, "name") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`)
			}
		}
	}
	return ""
}

func detectEdition(cargoPath string) string {
	data, err := os.ReadFile(cargoPath)
	if err != nil {
		return "2021"
	}
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	inPackage := false
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "[package]" {
			inPackage = true
			continue
		}
		if inPackage && strings.HasPrefix(line, "[") {
			break
		}
		if inPackage && strings.HasPrefix(line, "edition") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.Trim(strings.TrimSpace(parts[1]), `"`)
			}
		}
	}
	return "2021"
}

func detectTestFramework(root string) string {
	data, err := os.ReadFile(filepath.Join(root, "Cargo.toml"))
	if err != nil {
		return "test"
	}
	content := strings.ToLower(string(data))
	switch {
	case strings.Contains(content, "rstest"):
		return "rstest"
	case strings.Contains(content, "proptest"):
		return "proptest"
	default:
		return "test"
	}
}
