// Package analysis contains the language-agnostic analysis pipeline.
package analysis

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/domain"
)

// Pipeline orchestrates source file analysis for a single language driver.
type Pipeline struct {
	driver domain.LanguageDriver
}

// New returns a Pipeline backed by the given driver.
func New(driver domain.LanguageDriver) *Pipeline {
	return &Pipeline{driver: driver}
}

// AnalyzeFile parses a single source file and returns its SourceAnalysis.
func (p *Pipeline) AnalyzeFile(path string, ctx *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return p.driver.AnalyzeFile(path, ctx)
}

// DiscoverUntested returns source files under root that have no corresponding
// test file according to the driver's DeriveTestPath convention.
func (p *Pipeline) DiscoverUntested(root string, ctx *domain.ProjectContext) ([]string, error) {
	sources, err := p.discoverSources(root, ctx.ExcludeDirs)
	if err != nil {
		return nil, err
	}

	var untested []string
	for _, src := range sources {
		testPath, err := p.driver.DeriveTestPath(src, ctx)
		if err != nil {
			continue
		}
		if _, err := os.Stat(testPath); errors.Is(err, os.ErrNotExist) {
			untested = append(untested, src)
		}
	}
	return untested, nil
}

// DiscoverInPath returns untested source files under the given sub-directory.
func (p *Pipeline) DiscoverInPath(dir string, ctx *domain.ProjectContext) ([]string, error) {
	sources, err := p.discoverSources(dir, ctx.ExcludeDirs)
	if err != nil {
		return nil, err
	}

	var untested []string
	for _, src := range sources {
		testPath, err := p.driver.DeriveTestPath(src, ctx)
		if err != nil {
			continue
		}
		if _, err := os.Stat(testPath); errors.Is(err, os.ErrNotExist) {
			untested = append(untested, src)
		}
	}
	return untested, nil
}

// DiscoverAndAnalyzeAll discovers all source files under root and returns their analyses.
// Used by graph and gaps pipelines.
func (p *Pipeline) DiscoverAndAnalyzeAll(root string, ctx *domain.ProjectContext) ([]*domain.SourceAnalysis, error) {
	sources, err := p.discoverSources(root, ctx.ExcludeDirs)
	if err != nil {
		return nil, err
	}

	var analyses []*domain.SourceAnalysis
	for _, src := range sources {
		a, err := p.driver.AnalyzeFile(src, ctx)
		if err != nil {
			// Non-fatal: skip unparseable files and continue.
			continue
		}
		analyses = append(analyses, a)
	}
	return analyses, nil
}

func (p *Pipeline) discoverSources(root string, excludeDirs []string) ([]string, error) {
	exts := make(map[string]bool)
	for _, e := range p.driver.FileExtensions() {
		exts[e] = true
	}

	excludeSet := make(map[string]bool, len(excludeDirs))
	for _, d := range excludeDirs {
		excludeSet[d] = true
	}

	fwCfg := p.driver.GetTestFrameworkConfig()

	var sources []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip unreadable entries
		}
		if d.IsDir() {
			if excludeSet[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if exts[ext] {
			base := d.Name()
			// Only apply suffix filter when it's more specific than a bare extension
			// (e.g. "_test.go" or ".test.ts" qualify; ".py" alone does not).
			if fwCfg.TestFileSuffix != "" &&
				filepath.Ext(fwCfg.TestFileSuffix) != fwCfg.TestFileSuffix &&
				strings.HasSuffix(base, fwCfg.TestFileSuffix) {
				return nil
			}
			if fwCfg.TestFilePrefix != "" && strings.HasPrefix(base, fwCfg.TestFilePrefix) {
				return nil
			}
			sources = append(sources, path)
		}
		return nil
	})
	return sources, err
}
