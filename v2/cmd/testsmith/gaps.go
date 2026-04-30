package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
	"github.com/spf13/cobra"
)

func newGapsCmd() *cobra.Command {
	var output string
	var top int
	var workspace string

	cmd := &cobra.Command{
		Use:   "gaps",
		Short: "Analyse and report test coverage gaps",
		Long: `Scan all source files, detect missing or skeletal tests, and write a
prioritised Markdown report ranking which files need attention most.
When workspaces are configured in .testsmith.yaml, all workspaces are analysed.

Examples:
  testsmith gaps
  testsmith gaps --output report.md
  testsmith gaps --top 10
  testsmith gaps --workspace api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGaps(output, top, workspace)
		},
	}

	cmd.Flags().StringVar(&output, "output", "testsmith_coverage_report.md", "output Markdown file")
	cmd.Flags().IntVar(&top, "top", 0, "show only top N gaps (0 = show all)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "analyse only this workspace (name or path)")
	return cmd
}

func runGaps(output string, top int, wsFilter string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) > 0 {
		return runGapsWorkspaces(cfg, cwd, output, top, wsFilter)
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

	gaps, totalSources, err := collectGaps(driver, ctx, ctx.Root)
	if err != nil {
		return err
	}

	return writeReport(gaps, totalSources, top, output, ctx.Root)
}

// runGapsWorkspaces analyses all (or one filtered) workspace and merges results
// into a single report.
func runGapsWorkspaces(cfg *config.Config, cwd, output string, top int, wsFilter string) error {
	var allGaps []domain.CoverageGap
	var totalSources int

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

		gaps, n, err := collectGaps(driver, ctx, wsRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
			continue
		}
		fmt.Printf("  %d source(s), %d gap(s)\n", n, len(gaps))
		allGaps = append(allGaps, gaps...)
		totalSources += n
	}

	return writeReport(allGaps, totalSources, top, output, cwd)
}

// collectGaps analyses all source files under root and returns the prioritised
// gaps plus the total source file count.
func collectGaps(driver domain.LanguageDriver, ctx *domain.ProjectContext, root string) ([]domain.CoverageGap, int, error) {
	pipeline := analysis.New(driver)
	analyses, err := pipeline.DiscoverAndAnalyzeAll(root, ctx)
	if err != nil {
		return nil, 0, fmt.Errorf("analyse project: %w", err)
	}
	gaps, err := generation.GapsForAnalyses(analyses, driver)
	if err != nil {
		return nil, 0, fmt.Errorf("compute gaps: %w", err)
	}
	return gaps, len(analyses), nil
}

// writeReport applies the --top limit, renders the Markdown, and either prints
// (dry-run) or writes the output file.
func writeReport(gaps []domain.CoverageGap, totalSources, top int, output, root string) error {
	if top > 0 && len(gaps) > top {
		gaps = gaps[:top]
	}

	report := generation.GenerateReport(gaps, totalSources)

	if dryRun {
		fmt.Print(report)
		return nil
	}

	absOutput, _ := filepath.Abs(output)
	if err := os.WriteFile(absOutput, []byte(report), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	rel, _ := filepath.Rel(root, absOutput)
	fmt.Printf("\nCoverage report written to %s (%d gap(s) found)\n", rel, len(gaps))
	return nil
}
