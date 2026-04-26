# Feature: watch — Auto-regenerate on File Changes

## What It Does

`testsmith watch` starts a long-running process that monitors the project's source directories for file system events. When a source file is created or modified, it automatically runs the `generate` pipeline for that file.

The watcher uses a **debounce** interval (default 500 ms) to coalesce rapid saves (e.g. format-on-save followed by a manual save) into a single regeneration run. Only source files (not test files, not fixture files) trigger regeneration.

---

## CLI

```
testsmith watch [flags]

Flags:
  --llm           Enable LLM body generation on watched changes
  --debounce <ms> Debounce interval in milliseconds (default: 500)
  --verbose       Print each file system event received
```

Press `Ctrl+C` to stop watching.

---

## Example Session

```
$ testsmith watch
Watching /Users/alice/projects/myapp/src ...
  ✓ src/services/payment.py changed — regenerated tests/src/services/test_payment.py
  ✓ src/models/user.py created  — created tests/src/models/test_user.py
^C Watch stopped.
```

---

## Go Implementation

The watcher is implemented using `github.com/fsnotify/fsnotify`, which provides cross-platform file system event notifications without polling.

```go
// internal/watch/watcher.go
type Watcher struct {
    driver     domain.LanguageDriver
    pipeline   *generation.GenerationPipeline
    debounce   time.Duration
    excludeDirs []string
}

func New(driver domain.LanguageDriver, pipeline *generation.GenerationPipeline, opts WatchOpts) *Watcher
func (w *Watcher) Start(rootDir string) error  // blocks until ctx cancelled
func (w *Watcher) Stop()
```

### Debounce Strategy

Events are collected into a `map[string]time.Time` (path → last-seen). A single ticker goroutine fires every `debounce/2` ms and flushes paths whose last event is older than `debounce`. This avoids spawning a new timer per event (which can exhaust file descriptors on large projects).

### Excluded Paths

Files in `ExcludeDirs` (`.git`, `node_modules`, `__pycache__`, etc.) are registered with the `fsnotify` watcher but immediately filtered before pipeline dispatch, matching the same exclusion list used by `generate --all`.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/watch.go` | Cobra subcommand, signal handling |
| `internal/watch/watcher.go` | fsnotify integration, debounce logic |
| `internal/generation/pipeline.go` | Reused generate pipeline |
