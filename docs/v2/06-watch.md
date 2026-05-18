# Feature: watch — Auto-regenerate on File Changes

## What It Does

`testsmith watch` starts a long-running process that monitors the project's source directories for file system events. When a source file is created or modified, it automatically runs the `generate` pipeline for that file.

The watcher uses a **debounce** interval (default 500 ms) to coalesce rapid saves (e.g. format-on-save followed by a manual save) into a single regeneration run. Only source files (not test files, not fixture files) trigger regeneration.

---

## CLI

```
testsmith watch [flags]

Flags:
  --llm               Enable LLM body generation on watched changes
  --debounce <ms>     Debounce interval in milliseconds (default: 500)
  --workspace <name>  Watch only this workspace (name or path)
  --verbose, -v       Print each file system event received
```

Press `Ctrl+C` to stop watching.

### Workspace Mode

When `workspaces:` are configured, one `Watcher` goroutine is started per workspace (each with its own driver and context). All watchers share the same `signal.NotifyContext` so `Ctrl+C` stops them all simultaneously. Use `--workspace <name>` to watch a single workspace.

---

## Example Session

```
$ testsmith watch
Watching /projects/myapp/src ...
  ✓ src/services/payment.py changed — created  tests/src/services/test_payment.py
  ✓ src/models/user.py changed — updated tests/src/models/test_user.py
^C
```

Workspace mode:

```
$ testsmith watch
  starting watcher for workspace: api
  starting watcher for workspace: frontend
Watching /projects/monorepo/services/api ...
Watching /projects/monorepo/services/frontend ...
  ✓ handler.go changed — created  handler_test.go
  ✓ src/utils.ts changed — created  src/utils.test.ts
```

---

## Go Implementation

The watcher is implemented using `github.com/fsnotify/fsnotify`, which provides cross-platform file system event notifications without polling.

```go
// internal/watch/watcher.go
type Watcher struct {
    driver      domain.LanguageDriver
    ctx         *domain.ProjectContext
    pipeline    *analysis.Pipeline
    genPipeline *generation.Pipeline
    executor    *generation.Executor
    debounce    time.Duration
    excludeDirs map[string]bool
    verbose     bool
}

func New(
    driver domain.LanguageDriver,
    ctx *domain.ProjectContext,
    genPipeline *generation.Pipeline,
    debounceMs int,
    verbose bool,
) *Watcher

func (w *Watcher) Start(ctx context.Context) error  // blocks until ctx cancelled
```

### Debounce Strategy

Events are collected into a `map[string]time.Time` (path → last-seen), protected by a `sync.Mutex`. A ticker goroutine fires every `debounce/2` ms and flushes paths whose last event is older than `debounce`. This avoids spawning a new timer per event (which can exhaust file descriptors on large projects).

### Source File Detection

`isSourceFile` checks the file extension against `driver.FileExtensions()` and then excludes files that match the driver's `TestFrameworkConfig.TestFilePrefix` or `TestFileSuffix`, so test files never retrigger themselves.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/watch.go` | Cobra subcommand, workspace fan-out, signal handling |
| `internal/watch/watcher.go` | fsnotify integration, debounce logic, process loop |
| `internal/generation/pipeline.go` | Reused generate pipeline |
