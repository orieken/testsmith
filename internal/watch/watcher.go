// Package watch provides a debounced file system watcher that triggers
// the generation pipeline when source files are created or modified.
package watch

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

// Handler is called with the absolute path of a changed source file.
type Handler func(path string, a *domain.SourceAnalysis) error

// Watcher monitors source directories and triggers the generation pipeline
// on debounced write events.
type Watcher struct {
	driver      domain.LanguageDriver
	ctx         *domain.ProjectContext
	pipeline    *analysis.Pipeline
	genPipeline *generation.Pipeline
	executor    *generation.Executor
	opts        domain.GenerateOpts
	debounce    time.Duration
	excludeDirs map[string]bool
	verbose     bool

	mu      sync.Mutex
	pending map[string]time.Time // path -> last event time
}

// New returns a Watcher ready to be started.
func New(
	driver domain.LanguageDriver,
	ctx *domain.ProjectContext,
	genPipeline *generation.Pipeline,
	debounceMs int,
	verbose bool,
) *Watcher {
	excludeSet := make(map[string]bool, len(ctx.ExcludeDirs))
	for _, d := range ctx.ExcludeDirs {
		excludeSet[d] = true
	}

	return &Watcher{
		driver:      driver,
		ctx:         ctx,
		pipeline:    analysis.New(driver),
		genPipeline: genPipeline,
		executor:    generation.NewVerifiedExecutor(ctx.Language),
		opts:        domain.GenerateOpts{},
		debounce:    time.Duration(debounceMs) * time.Millisecond,
		excludeDirs: excludeSet,
		verbose:     verbose,
		pending:     make(map[string]time.Time),
	}
}

// Start begins watching ctx.Root and blocks until ctx is cancelled or an
// unrecoverable error occurs.
func (w *Watcher) Start(ctx context.Context) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create watcher: %w", err)
	}
	defer watcher.Close()

	// Register every source directory under the project root.
	if err := w.addDirs(watcher, w.ctx.Root); err != nil {
		return err
	}

	fmt.Printf("Watching %s ...\n", w.ctx.Root)

	// Ticker flushes pending events every debounce/2.
	ticker := time.NewTicker(w.debounce / 2)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil

		case event, ok := <-watcher.Events:
			if !ok {
				return nil
			}
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				if w.isSourceFile(event.Name) {
					w.mu.Lock()
					w.pending[event.Name] = time.Now()
					w.mu.Unlock()
				}
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return nil
			}
			fmt.Printf("  watch error: %v\n", err)

		case now := <-ticker.C:
			w.flush(now)
		}
	}
}

// flush processes all pending paths whose last event is older than the debounce window.
func (w *Watcher) flush(now time.Time) {
	w.mu.Lock()
	var ready []string
	for path, last := range w.pending {
		if now.Sub(last) >= w.debounce {
			ready = append(ready, path)
			delete(w.pending, path)
		}
	}
	w.mu.Unlock()

	for _, path := range ready {
		w.process(path)
	}
}

func (w *Watcher) process(path string) {
	a, err := w.pipeline.AnalyzeFile(path, w.ctx)
	if err != nil {
		fmt.Printf("  ✗ %s: %v\n", filepath.Base(path), err)
		return
	}

	// Keep the dep index current so subsequent generations of files that import
	// this module receive its up-to-date public API signatures.
	w.genPipeline.UpdateDepEntry(a.ModulePath, a)

	plan, err := w.genPipeline.Plan(context.Background(), a, w.opts)
	if err != nil {
		fmt.Printf("  ✗ %s: %v\n", filepath.Base(path), err)
		return
	}

	results, _ := w.executor.Execute(plan)

	for _, r := range results {
		rel, _ := filepath.Rel(w.ctx.Root, r.AbsPath)
		switch r.Action {
		case domain.ActionCreate:
			fmt.Printf("  ✓ %s changed — created  %s\n", filepath.Base(path), rel)
		case domain.ActionUpdate:
			fmt.Printf("  ✓ %s changed — updated  %s\n", filepath.Base(path), rel)
		case domain.ActionSkip:
			// already up-to-date; nothing to report
		}
	}
}

// addDirs recursively adds all non-excluded directories to the watcher.
func (w *Watcher) addDirs(watcher *fsnotify.Watcher, root string) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil //nolint:nilerr
		}
		if !d.IsDir() {
			return nil
		}
		if w.excludeDirs[d.Name()] {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
}

// isSourceFile returns true when the file extension is claimed by the driver
// and the file is not itself a test file.
func (w *Watcher) isSourceFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	base := filepath.Base(path)

	for _, e := range w.driver.FileExtensions() {
		if e == ext {
			// Exclude test files from triggering re-generation.
			cfg := w.driver.GetTestFrameworkConfig()
			if cfg.TestFilePrefix != "" && strings.HasPrefix(base, cfg.TestFilePrefix) {
				return false
			}
			if cfg.TestFileSuffix != "" && strings.HasSuffix(base, cfg.TestFileSuffix) {
				return false
			}
			return true
		}
	}
	return false
}
