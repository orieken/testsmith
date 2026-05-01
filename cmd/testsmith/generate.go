package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
	"github.com/orieken/testsmith/internal/llm/factory"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var (
		all       bool
		path      string
		llmFlag   bool
		overwrite bool
		lang      string
		workers   int
		workspace string
	)

	cmd := &cobra.Command{
		Use:   "generate [file]",
		Short: "Generate test scaffolds for source files",
		Long: `Generate test scaffolds for one file, a directory, or the entire project.
When workspaces are configured in .testsmith.yaml, --all processes every workspace.

Examples:
  testsmith generate src/payment.py
  testsmith generate --all
  testsmith generate --all --workers 8
  testsmith generate --all --workspace api
  testsmith generate --path src/services/
  testsmith generate src/payment.py --llm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(args, all, path, llmFlag, overwrite, lang, workspace, workers)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "generate tests for every untested source file")
	cmd.Flags().StringVar(&path, "path", "", "generate tests for untested files under this directory")
	cmd.Flags().BoolVar(&llmFlag, "llm", false, "use LLM to generate test bodies")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "regenerate even if test file already exists")
	cmd.Flags().StringVar(&lang, "lang", "", "override auto-detected language")
	cmd.Flags().StringVar(&workspace, "workspace", "", "process only this workspace (name or path)")
	cmd.Flags().IntVar(&workers, "workers", runtime.NumCPU(), "number of parallel workers (1 = sequential)")

	return cmd
}

func runGenerate(args []string, all bool, pathFlag string, llmFlag bool, overwrite bool, lang, wsFilter string, workers int) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Workspace mode: --all with workspaces configured.
	if all && len(cfg.Workspaces) > 0 {
		return runGenerateWorkspaces(cfg, cwd, wsFilter, llmFlag, overwrite, workers)
	}

	// Single-project mode.
	var driver domain.LanguageDriver
	var ctx *domain.ProjectContext

	if lang != "" {
		driver, err = reg.ForLanguage(lang)
		if err != nil {
			return err
		}
		ctx, err = driver.DetectProject(cwd)
	} else {
		driver, ctx, err = reg.Detect(cwd)
	}
	if err != nil {
		return fmt.Errorf("detect project: %w", err)
	}

	if verbose {
		fmt.Printf("Language: %s\nProject root: %s\nWorkers: %d\n", ctx.Language, ctx.Root, workers)
	}

	ctx.ExcludeDirs = append(ctx.ExcludeDirs, cfg.ExcludeDirs...)
	config.ApplyToContext(cfg, ctx)

	pipeline := analysis.New(driver)
	bodyGen, err := buildBodyGen(llmFlag, cfg.LLM, driver)
	if err != nil {
		return err
	}

	genPipeline := generation.NewPipeline(driver, bodyGen)
	executor := &generation.Executor{}
	opts := domain.GenerateOpts{DryRun: dryRun, OverwriteExisting: overwrite}

	var files []string
	switch {
	case all:
		files, err = pipeline.DiscoverUntested(ctx.Root, ctx)
	case pathFlag != "":
		abs, _ := filepath.Abs(pathFlag)
		files, err = pipeline.DiscoverInPath(abs, ctx)
	case len(args) > 0:
		abs, _ := filepath.Abs(args[0])
		files = []string{abs}
	default:
		return fmt.Errorf("provide a source file, --all, or --path")
	}
	if err != nil {
		return err
	}

	if len(files) == 0 {
		fmt.Println("No files found to process.")
		return nil
	}

	created, skipped, failed := processFiles(
		context.Background(), files, generation.ClampWorkers(workers, len(files)),
		pipeline, genPipeline, executor, ctx, opts,
	)

	fmt.Printf("\nProcessed %d file(s): %d created/updated, %d skipped, %d failed\n",
		len(files), created, skipped, failed)

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed", failed)
	}
	return nil
}

// runGenerateWorkspaces iterates cfg.Workspaces and generates tests for each.
func runGenerateWorkspaces(cfg *config.Config, cwd, wsFilter string, llmFlag, overwrite bool, workers int) error {
	var totalFiles, totalCreated, totalSkipped, totalFailed int

	for i := range cfg.Workspaces {
		ws := &cfg.Workspaces[i]
		id := config.WorkspaceID(ws)

		if wsFilter != "" && id != wsFilter && ws.Path != wsFilter {
			continue
		}

		wsRoot := filepath.Join(cwd, ws.Path)
		fmt.Printf("\n── workspace: %s (%s) ──\n", id, ws.Path)

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

		pipeline := analysis.New(driver)
		genPipeline := generation.NewPipeline(driver, bodyGen)
		executor := &generation.Executor{}
		opts := domain.GenerateOpts{DryRun: dryRun, OverwriteExisting: overwrite}

		files, err := pipeline.DiscoverUntested(wsRoot, ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s discover: %v\n", id, err)
			continue
		}

		if len(files) == 0 {
			fmt.Printf("  · no untested files\n")
			continue
		}

		created, skipped, failed := processFiles(
			context.Background(), files, generation.ClampWorkers(workers, len(files)),
			pipeline, genPipeline, executor, ctx, opts,
		)
		totalFiles += len(files)
		totalCreated += created
		totalSkipped += skipped
		totalFailed += failed
	}

	fmt.Printf("\nTotal — %d file(s): %d created/updated, %d skipped, %d failed\n",
		totalFiles, totalCreated, totalSkipped, totalFailed)

	if totalFailed > 0 {
		return fmt.Errorf("%d file(s) failed across workspaces", totalFailed)
	}
	return nil
}

// resolveWorkspaceDriver returns the driver and context for a workspace.
func resolveWorkspaceDriver(ws *config.WorkspaceConfig, wsRoot string) (domain.LanguageDriver, *domain.ProjectContext, error) {
	if ws.Language != "" {
		driver, err := reg.ForLanguage(ws.Language)
		if err != nil {
			return nil, nil, err
		}
		ctx, err := driver.DetectProject(wsRoot)
		if err != nil {
			ctx = &domain.ProjectContext{Language: ws.Language, Root: wsRoot, Metadata: map[string]any{}}
		}
		return driver, ctx, nil
	}
	driver, ctx, err := reg.Detect(wsRoot)
	if err != nil {
		return nil, nil, fmt.Errorf("auto-detect language: %w", err)
	}
	return driver, ctx, nil
}

// buildBodyGen constructs a BodyGenerator when llmFlag is true, else returns nil.
func buildBodyGen(llmFlag bool, llmCfg config.LLMConfig, driver domain.LanguageDriver) (domain.BodyGenerator, error) {
	if !llmFlag {
		return nil, nil
	}
	llmCfg.Enabled = true
	bg, err := factory.Build(llmCfg, driver)
	if err != nil {
		return nil, fmt.Errorf("init LLM: %w", err)
	}
	return bg, nil
}

// fileResult carries the outcome of processing a single source file.
type fileResult struct {
	srcPath string
	results []generation.Result
	err     error
}

// processFiles fans out file processing across workers goroutines and streams
// status lines to stdout as results arrive. Returns counters for the summary.
func processFiles(
	ctx context.Context,
	files []string,
	workers int,
	pipeline *analysis.Pipeline,
	genPipeline *generation.Pipeline,
	executor *generation.Executor,
	projCtx *domain.ProjectContext,
	opts domain.GenerateOpts,
) (created, skipped, failed int) {
	work := make(chan string)
	results := make(chan fileResult, workers)

	// Fan out: send files to workers.
	go func() {
		for _, f := range files {
			work <- f
		}
		close(work)
	}()

	// Workers: analyze → plan → execute.
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for f := range work {
				results <- processOne(ctx, f, pipeline, genPipeline, executor, projCtx, opts)
			}
		}()
	}

	// Close results once all workers finish.
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect and print results.
	var mu sync.Mutex
	for r := range results {
		mu.Lock()
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", filepath.Base(r.srcPath), r.err)
			failed++
		} else {
			for _, res := range r.results {
				rel, _ := filepath.Rel(projCtx.Root, res.AbsPath)
				switch res.Action {
				case domain.ActionCreate:
					fmt.Printf("  ✓ created  %s\n", rel)
					created++
				case domain.ActionUpdate:
					fmt.Printf("  ✓ updated  %s\n", rel)
					created++
				case domain.ActionSkip:
					if verbose {
						fmt.Printf("  · skipped  %s\n", rel)
					}
					skipped++
				}
			}
		}
		mu.Unlock()
	}
	return
}

// processOne runs the full analyze → plan → execute pipeline for one file.
func processOne(
	ctx context.Context,
	srcPath string,
	pipeline *analysis.Pipeline,
	genPipeline *generation.Pipeline,
	executor *generation.Executor,
	projCtx *domain.ProjectContext,
	opts domain.GenerateOpts,
) fileResult {
	a, err := pipeline.AnalyzeFile(srcPath, projCtx)
	if err != nil {
		return fileResult{srcPath: srcPath, err: err}
	}
	plan, err := genPipeline.Plan(ctx, a, opts)
	if err != nil {
		return fileResult{srcPath: srcPath, err: err}
	}
	res, err := executor.Execute(plan)
	if err != nil {
		return fileResult{srcPath: srcPath, err: err}
	}
	return fileResult{srcPath: srcPath, results: res}
}

