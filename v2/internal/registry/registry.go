// Package registry maintains the set of registered LanguageDrivers and resolves
// the correct driver for a given file path or language name.
package registry

import (
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// Registry holds all registered LanguageDrivers.
type Registry struct {
	drivers []domain.LanguageDriver
}

// New returns an empty Registry.
func New() *Registry {
	return &Registry{}
}

// Register adds a driver to the registry.
func (r *Registry) Register(d domain.LanguageDriver) {
	r.drivers = append(r.drivers, d)
}

// Detect walks upward from dir and returns the first driver whose DetectProject
// succeeds, along with the resolved ProjectContext.
func (r *Registry) Detect(dir string) (domain.LanguageDriver, *domain.ProjectContext, error) {
	for _, d := range r.drivers {
		ctx, err := d.DetectProject(dir)
		if err == nil {
			return d, ctx, nil
		}
	}
	return nil, nil, domain.ErrProjectNotFound
}

// ForLanguage returns the driver registered for the given canonical language name.
func (r *Registry) ForLanguage(lang string) (domain.LanguageDriver, error) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	for _, d := range r.drivers {
		if d.Language() == lang {
			return d, nil
		}
	}
	return nil, domain.ErrNoDriverForLanguage
}

// ForFile returns the driver that claims the given file's extension.
func (r *Registry) ForFile(path string) (domain.LanguageDriver, error) {
	ext := strings.ToLower(filepath.Ext(path))
	for _, d := range r.drivers {
		for _, e := range d.FileExtensions() {
			if e == ext {
				return d, nil
			}
		}
	}
	return nil, domain.ErrNoDriverForFile
}
