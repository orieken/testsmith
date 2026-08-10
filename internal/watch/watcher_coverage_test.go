package watch

// watcher_coverage_test.go — coverage backfill for process, addDirs, and Start.
//
// Mode: Coverage backfill.  The code under test is trusted and not changing;
// these tests lock in existing behaviour and bring the package to ≥ 85%.
//
// Annotation convention (Go / testing): issue-ref + AC as a comment above each
// function, per shared/rules/testing-conventions.md.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/generation"
)

// ── helper driver types ───────────────────────────────────────────────────────

// analyzeErrDriver embeds fakeDriver but overrides AnalyzeFile to return a
// configurable error.  Used to exercise process's first error branch.
type analyzeErrDriver struct {
	fakeDriver
	err error
}

func (d *analyzeErrDriver) AnalyzeFile(_ string, _ *domain.ProjectContext) (*domain.SourceAnalysis, error) {
	return nil, d.err
}

// derivePathErrDriver embeds fakeDriver but overrides DeriveTestPath to return
// a configurable error, causing generation.Pipeline.Plan to fail.  Used to
// exercise process's second error branch.
type derivePathErrDriver struct {
	fakeDriver
	err error
}

func (d *derivePathErrDriver) DeriveTestPath(_ string, _ *domain.ProjectContext) (string, error) {
	return "", d.err
}

// newWatcherForDir creates a Watcher whose project root is set to dir.
// Useful for tests that need real filesystem interaction.
func newWatcherForDir(driver domain.LanguageDriver, dir string) *Watcher {
	ctx := &domain.ProjectContext{Root: dir, Language: driver.Language()}
	gen := generation.NewPipeline(driver, nil)
	return New(driver, ctx, gen, 200, false)
}

// ── process: AnalyzeFile error path ──────────────────────────────────────────

// backfill / AC: process prints the error and returns early when AnalyzeFile fails
func TestProcess_AnalyzeFileError_ExitsGracefully(t *testing.T) {
	d := &analyzeErrDriver{
		fakeDriver: fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"},
		err:        errors.New("parse error"),
	}
	w := newTestWatcher(d)

	// Must not panic; the function should print the error and return.
	w.process("/proj/payment.go")
}

// ── process: Plan error path ──────────────────────────────────────────────────

// backfill / AC: process prints the error and returns early when Plan fails
func TestProcess_PlanError_ExitsGracefully(t *testing.T) {
	d := &derivePathErrDriver{
		fakeDriver: fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"},
		err:        errors.New("cannot derive test path"),
	}
	w := newTestWatcher(d)

	// Must not panic; the function should print the error and return.
	w.process("/proj/payment.go")
}

// ── process: ActionUpdate ─────────────────────────────────────────────────────

// backfill / AC: process overwrites the existing test file when OverwriteExisting is true
func TestProcess_ActionUpdate_OverwritesExistingTestFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Use "java" so the executor skips compile verification (no Java verifier).
	d := &fakeDriver{lang: "java", extensions: []string{".java"}}
	w := newWatcherForDir(d, tmpDir)
	w.opts.OverwriteExisting = true

	srcPath := filepath.Join(tmpDir, "Payment.java")
	// fakeDriver.DeriveTestPath returns srcPath+"_test".
	testPath := srcPath + "_test"

	// Pre-create the test file so resolveAction picks ActionUpdate.
	if err := os.WriteFile(testPath, []byte("original content"), 0o644); err != nil {
		t.Fatal(err)
	}

	w.process(srcPath)

	got, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("test file not found after process: %v", err)
	}
	// fakeDriver.GenerateTestFile returns Content "// test".
	const wantContent = "// test"
	if string(got) != wantContent {
		t.Errorf("expected overwritten content %q, got %q", wantContent, string(got))
	}
}

// ── process: ActionSkip ───────────────────────────────────────────────────────

// backfill / AC: process leaves the existing test file unchanged when OverwriteExisting is false
func TestProcess_ActionSkip_LeavesExistingTestFileUnchanged(t *testing.T) {
	tmpDir := t.TempDir()

	d := &fakeDriver{lang: "java", extensions: []string{".java"}}
	w := newWatcherForDir(d, tmpDir)
	// OverwriteExisting defaults to false → resolveAction returns ActionSkip when file exists.

	srcPath := filepath.Join(tmpDir, "Payment.java")
	testPath := srcPath + "_test"

	const originalContent = "my existing tests"
	if err := os.WriteFile(testPath, []byte(originalContent), 0o644); err != nil {
		t.Fatal(err)
	}

	w.process(srcPath)

	got, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("test file not found after process: %v", err)
	}
	if string(got) != originalContent {
		t.Errorf("expected file to be unchanged (%q), got %q", originalContent, string(got))
	}
}

// ── addDirs ───────────────────────────────────────────────────────────────────

// backfill / AC: addDirs registers the root directory and every non-excluded subdirectory
func TestAddDirs_RegistersRootAndSubdirectories(t *testing.T) {
	tmpDir := t.TempDir()
	subA := filepath.Join(tmpDir, "subA")
	subA1 := filepath.Join(subA, "subA1")
	subB := filepath.Join(tmpDir, "subB")

	for _, dir := range []string{subA, subA1, subB} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// Create a regular file to exercise the !d.IsDir() early-return branch.
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main"), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	w := newTestWatcher(d)

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create fsnotify watcher: %v", err)
	}
	defer fw.Close()

	if err := w.addDirs(fw, tmpDir); err != nil {
		t.Fatalf("addDirs returned unexpected error: %v", err)
	}

	watchSet := make(map[string]bool, len(fw.WatchList()))
	for _, p := range fw.WatchList() {
		watchSet[p] = true
	}

	for _, dir := range []string{tmpDir, subA, subA1, subB} {
		if !watchSet[dir] {
			t.Errorf("expected %s to be in watchlist, got %v", dir, fw.WatchList())
		}
	}
}

// backfill / AC: addDirs skips directories whose names appear in ExcludeDirs
func TestAddDirs_SkipsExcludedDirectories(t *testing.T) {
	tmpDir := t.TempDir()
	vendor := filepath.Join(tmpDir, "vendor")
	vendorPkg := filepath.Join(vendor, "somelib")
	src := filepath.Join(tmpDir, "src")

	for _, dir := range []string{vendor, vendorPkg, src} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	d := &fakeDriver{lang: "go", extensions: []string{".go"}, suffix: "_test.go"}
	projCtx := &domain.ProjectContext{
		Root:        tmpDir,
		Language:    "go",
		ExcludeDirs: []string{"vendor"},
	}
	gen := generation.NewPipeline(d, nil)
	w := New(d, projCtx, gen, 200, false)

	fw, err := fsnotify.NewWatcher()
	if err != nil {
		t.Fatalf("create fsnotify watcher: %v", err)
	}
	defer fw.Close()

	if err := w.addDirs(fw, tmpDir); err != nil {
		t.Fatalf("addDirs returned unexpected error: %v", err)
	}

	watchSet := make(map[string]bool, len(fw.WatchList()))
	for _, p := range fw.WatchList() {
		watchSet[p] = true
	}

	// The excluded directory and its subtree must not appear in the watchlist.
	for _, excluded := range []string{vendor, vendorPkg} {
		if watchSet[excluded] {
			t.Errorf("excluded path %s was registered in the watchlist", excluded)
		}
	}
	// Non-excluded directories must appear.
	if !watchSet[src] {
		t.Errorf("expected non-excluded dir %s to be in watchlist", src)
	}
}

// ── Start ─────────────────────────────────────────────────────────────────────

// backfill / AC: Start returns nil when the context is cancelled before the first tick
func TestStart_ContextCancellationReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()

	d := &fakeDriver{lang: "java", extensions: []string{".java"}}
	projCtx := &domain.ProjectContext{Root: tmpDir, Language: "java"}
	gen := generation.NewPipeline(d, nil)
	w := New(d, projCtx, gen, 200, false)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before Start so ctx.Done() is already closed

	if err := w.Start(ctx); err != nil {
		t.Errorf("Start with pre-cancelled context: expected nil, got %v", err)
	}
}

// backfill / AC: Start flushes pending entries via the ticker when the debounce expires
func TestStart_TickerFlushesExpiredEntries(t *testing.T) {
	tmpDir := t.TempDir()

	d := &fakeDriver{lang: "java", extensions: []string{".java"}}
	projCtx := &domain.ProjectContext{Root: tmpDir, Language: "java"}
	gen := generation.NewPipeline(d, nil)
	// 2 ms debounce → ticker fires every 1 ms; entries expire in 2 ms.
	w := New(d, projCtx, gen, 2, false)

	// Inject a stale entry so the first ticker flush has something to process.
	stalePath := filepath.Join(tmpDir, "Payment.java")
	w.mu.Lock()
	w.pending[stalePath] = time.Now().Add(-100 * time.Millisecond)
	w.mu.Unlock()

	// Allow enough time for several ticks and then stop.
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	if err := w.Start(ctx); err != nil {
		t.Errorf("Start: expected nil, got %v", err)
	}

	// The stale entry must have been drained by the ticker-driven flush.
	if n := pendingLen(w); n != 0 {
		t.Errorf("expected pending map to be empty after flush, got %d entries", n)
	}
}

// backfill / AC: Start adds a newly created source file to the pending map
func TestStart_FileCreateEventQueuedInPending(t *testing.T) {
	tmpDir := t.TempDir()

	d := &fakeDriver{lang: "java", extensions: []string{".java"}}
	projCtx := &domain.ProjectContext{Root: tmpDir, Language: "java"}
	gen := generation.NewPipeline(d, nil)
	// Long debounce keeps the file in pending long enough for us to inspect it.
	w := New(d, projCtx, gen, 5_000, false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- w.Start(ctx)
	}()

	srcFile := filepath.Join(tmpDir, "Payment.java")
	// Write the source file 20 ms after scheduling so fsnotify is listening.
	time.AfterFunc(20*time.Millisecond, func() {
		_ = os.WriteFile(srcFile, []byte("// src"), 0o644)
	})

	// Poll the pending map with a channel-based ticker; no time.Sleep.
	deadline := time.After(4 * time.Second)
	poll := time.NewTicker(5 * time.Millisecond)
	defer poll.Stop()
	found := false
	for !found {
		select {
		case <-deadline:
			t.Error("timed out waiting for file-create event to appear in pending map")
			cancel()
			<-errCh
			return
		case <-poll.C:
			w.mu.Lock()
			_, found = w.pending[srcFile]
			w.mu.Unlock()
		}
	}

	cancel()
	if err := <-errCh; err != nil {
		t.Errorf("Start: unexpected error after cancel: %v", err)
	}
}
