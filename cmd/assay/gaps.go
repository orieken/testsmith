package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orieken/assay/internal/analysis"
	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
	"github.com/orieken/assay/internal/generation"
)

func newGapsCmd() *cobra.Command {
	var output string
	var top int
	var workspace string
	var format string
	var check bool

	cmd := &cobra.Command{
		Use:   "gaps",
		Short: "Analyse and report test coverage gaps",
		Long: `Scan all source files, detect missing or skeletal tests, and write a
prioritised report ranking which files need attention most.
When workspaces are configured in .assay.yaml, all workspaces are analysed.

Examples:
  assay gaps
  assay gaps --output report.md
  assay gaps --format json
  assay gaps --format junit --output results.xml
  assay gaps --top 10
  assay gaps --check
  assay gaps --workspace api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGaps(output, top, workspace, format, check)
		},
	}

	cmd.Flags().StringVar(&output, "output", "", "output file (default: assay_coverage_report.md for markdown, stdout for json/junit)")
	cmd.Flags().IntVar(&top, "top", 0, "show only top N gaps (0 = show all)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "analyse only this workspace (name or path)")
	cmd.Flags().StringVar(&format, "format", "markdown", "output format: markdown, json, junit")
	cmd.Flags().BoolVar(&check, "check", false, "exit non-zero if any gaps are found (useful in CI)")
	return cmd
}

func runGaps(output string, top int, wsFilter, format string, check bool) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) > 0 {
		return runGapsWorkspaces(cfg, cwd, output, top, wsFilter, format, check)
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

	if err := writeReport(gaps, totalSources, top, output, ctx.Root, format); err != nil {
		return err
	}

	if check && len(gaps) > 0 {
		return fmt.Errorf("check failed: %d coverage gap(s) found", len(gaps))
	}
	return nil
}

// runGapsWorkspaces analyses all (or one filtered) workspace and merges results
// into a single report.
func runGapsWorkspaces(cfg *config.Config, cwd, output string, top int, wsFilter, format string, check bool) error {
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

	if err := writeReport(allGaps, totalSources, top, output, cwd, format); err != nil {
		return err
	}

	if check && len(allGaps) > 0 {
		return fmt.Errorf("check failed: %d coverage gap(s) found", len(allGaps))
	}
	return nil
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

// writeReport applies the --top limit, renders the report in the requested
// format, and either prints to stdout or writes to the output file.
func writeReport(gaps []domain.CoverageGap, totalSources, top int, output, root, format string) error {
	if top > 0 && len(gaps) > top {
		gaps = gaps[:top]
	}

	var report string
	switch format {
	case "json":
		report = generation.GenerateReportJSON(gaps, totalSources)
	case "junit":
		report = generation.GenerateReportJUnit(gaps, totalSources)
	default:
		report = generation.GenerateReport(gaps, totalSources)
	}

	// json/junit default to stdout; markdown defaults to a file.
	if output == "" && format != "markdown" {
		fmt.Print(report)
		return nil
	}

	if dryRun {
		fmt.Print(report)
		return nil
	}

	dest := output
	if dest == "" {
		dest = "assay_coverage_report.md"
	}

	absOutput, _ := filepath.Abs(dest)
	if err := os.WriteFile(absOutput, []byte(report), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	rel, _ := filepath.Rel(root, absOutput)
	fmt.Printf("\nReport written to %s (%d gap(s) found)\n", rel, len(gaps))
	return nil
}
