package main

import (
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// discoverTestFiles walks root and returns source files whose names suggest
// they are test files (contain "test" or "spec", case-insensitive).
func discoverTestFiles(root string, driver domain.LanguageDriver) ([]string, error) {
	return walkTestFiles(root, driver)
}

// discoverValidationFiles is an alias; both commands use the same heuristic.
func discoverValidationFiles(root string, driver domain.LanguageDriver) ([]string, error) {
	return walkTestFiles(root, driver)
}

func walkTestFiles(root string, driver domain.LanguageDriver) ([]string, error) {
	exts := make(map[string]bool)
	for _, e := range driver.FileExtensions() {
		exts[strings.ToLower(e)] = true
	}

	var files []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if strings.HasPrefix(base, ".") || base == "node_modules" || base == "vendor" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !exts[ext] {
			return nil
		}
		lower := strings.ToLower(filepath.Base(path))
		if strings.Contains(lower, "test") || strings.Contains(lower, "spec") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}
