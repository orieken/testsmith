package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/generation"
	"github.com/orieken/testsmith/internal/watch"
)

func newWatchCmd() *cobra.Command {
	var llmFlag bool
	var debounce int
	var workspace string

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for file changes and auto-regenerate tests",
		Long: `Monitor source files for changes and automatically regenerate test
scaffolds on save. Press Ctrl+C to stop.
When workspaces are configured in .testsmith.yaml, each workspace
is watched concurrently in its own goroutine.

Examples:
  testsmith watch
  testsmith watch --debounce 1000
  testsmith watch --llm
  testsmith watch --workspace api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(llmFlag, debounce, workspace)
		},
	}

	cmd.Flags().BoolVar(&llmFlag, "llm", false, "enable LLM on watched changes")
	cmd.Flags().IntVar(&debounce, "debounce", 500, "debounce interval in ms")
	cmd.Flags().StringVar(&workspace, "workspace", "", "watch only this workspace (name or path)")
	return cmd
}

func runWatch(llmFlag bool, debounceMs int, wsFilter string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	sigCtx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if len(cfg.Workspaces) > 0 {
		return runWatchWorkspaces(sigCtx, cfg, cwd, llmFlag, debounceMs, wsFilter)
	}

	driver, ctx, err := reg.Detect(cwd)
	if err != nil {
		return fmt.Errorf("detect project: %w", err)
	}
	ctx.ExcludeDirs = append(ctx.ExcludeDirs, cfg.ExcludeDirs...)
	config.ApplyToContext(cfg, ctx)

	if verbose {
		fmt.Printf("Language: %s\nProject root: %s\n", ctx.Language, ctx.Root)
	}

	bodyGen, err := buildBodyGen(llmFlag, cfg.LLM, driver)
	if err != nil {
		return fmt.Errorf("init LLM: %w", err)
	}

	genPipeline := generation.NewPipeline(driver, bodyGen)
	w := watch.New(driver, ctx, genPipeline, debounceMs, verbose)
	return w.Start(sigCtx)
}

// runWatchWorkspaces starts one watcher per workspace concurrently and blocks
// until the context is cancelled or all watchers exit.
func runWatchWorkspaces(sigCtx context.Context, cfg *config.Config, cwd string, llmFlag bool, debounceMs int, wsFilter string) error {
	type watchEntry struct {
		label   string
		watcher *watch.Watcher
	}

	var entries []watchEntry
	for i := range cfg.Workspaces {
		ws := &cfg.Workspaces[i]
		id := config.WorkspaceID(ws)
		if wsFilter != "" && id != wsFilter && ws.Path != wsFilter {
			continue
		}

		wsRoot := filepath.Join(cwd, ws.Path)
		driver, ctx, err := resolveWorkspaceDriver(ws, wsRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
			continue
		}
		ctx.ExcludeDirs = append(ctx.ExcludeDirs, cfg.ExcludeDirs...)
		config.ApplyToContext(cfg, ctx)

		llmCfg := config.WorkspaceLLM(cfg.LLM, ws)
		bodyGen, err := buildBodyGen(llmFlag, llmCfg, driver)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
			continue
		}

		genPipeline := generation.NewPipeline(driver, bodyGen)
		entries = append(entries, watchEntry{
			label:   id,
			watcher: watch.New(driver, ctx, genPipeline, debounceMs, verbose),
		})
	}

	if len(entries) == 0 {
		return fmt.Errorf("no workspaces to watch")
	}

	errc := make(chan error, len(entries))
	for _, e := range entries {
		e := e
		go func() {
			fmt.Printf("  starting watcher for workspace: %s\n", e.label)
			errc <- e.watcher.Start(sigCtx)
		}()
	}

	// Wait for all watchers to stop (either from Ctrl+C or an error).
	var first error
	for range entries {
		if err := <-errc; err != nil && first == nil {
			first = err
		}
	}
	return first
}
