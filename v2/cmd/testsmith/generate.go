package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
	"github.com/spf13/cobra"
)

func newGenerateCmd() *cobra.Command {
	var (
		all       bool
		path      string
		llmFlag   bool
		overwrite bool
		lang      string
	)

	cmd := &cobra.Command{
		Use:   "generate [file]",
		Short: "Generate test scaffolds for source files",
		Long: `Generate test scaffolds for one file, a directory, or the entire project.

Examples:
  testsmith generate src/payment.py
  testsmith generate --all
  testsmith generate --path src/services/
  testsmith generate src/payment.py --llm`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGenerate(args, all, path, llmFlag, overwrite, lang)
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "generate tests for every untested source file")
	cmd.Flags().StringVar(&path, "path", "", "generate tests for untested files under this directory")
	cmd.Flags().BoolVar(&llmFlag, "llm", false, "use LLM to generate test bodies")
	cmd.Flags().BoolVar(&overwrite, "overwrite", false, "regenerate even if test file already exists")
	cmd.Flags().StringVar(&lang, "lang", "", "override auto-detected language")

	return cmd
}

func runGenerate(args []string, all bool, pathFlag string, llmFlag bool, overwrite bool, lang string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Resolve driver.
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
		fmt.Printf("Language: %s\nProject root: %s\n", ctx.Language, ctx.Root)
	}

	// Apply config exclude dirs.
	ctx.ExcludeDirs = append(ctx.ExcludeDirs, cfg.ExcludeDirs...)

	pipeline := analysis.New(driver)
	genPipeline := generation.NewPipeline(driver, nil) // LLM wired in Phase 3
	executor := &generation.Executor{}

	opts := domain.GenerateOpts{
		DryRun:            dryRun,
		OverwriteExisting: overwrite,
	}

	// Collect files to process.
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

	// Process each file.
	var created, skipped, failed int
	for _, f := range files {
		a, err := pipeline.AnalyzeFile(f, ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		plan, err := genPipeline.Plan(context.Background(), a, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		results, err := executor.Execute(plan)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", filepath.Base(f), err)
			failed++
			continue
		}

		for _, r := range results {
			rel, _ := filepath.Rel(ctx.Root, r.AbsPath)
			switch r.Action {
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

	fmt.Printf("\nProcessed %d file(s): %d created/updated, %d skipped, %d failed\n",
		len(files), created, skipped, failed)

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed", failed)
	}
	return nil
}
