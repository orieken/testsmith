package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

func newPruneCmd() *cobra.Command {
	var confirm bool
	var workspace string

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Identify and remove unused fixture files",
		Long: `Scan all source files, then find fixture files that no longer correspond
to any active external dependency. By default runs as a dry-run.
When workspaces are configured in .testsmith.yaml, all workspaces are pruned.

Examples:
  testsmith prune           # list unused fixtures (dry-run)
  testsmith prune --confirm # actually delete them
  testsmith prune --workspace api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(confirm, workspace)
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "actually delete (default: dry-run)")
	cmd.Flags().StringVar(&workspace, "workspace", "", "prune only this workspace (name or path)")
	return cmd
}

func runPrune(confirm bool, wsFilter string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) > 0 {
		return runPruneWorkspaces(cfg, cwd, confirm, wsFilter)
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

	return pruneOne(driver, ctx, ctx.Root, confirm)
}

func runPruneWorkspaces(cfg *config.Config, cwd string, confirm bool, wsFilter string) error {
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

		if err := pruneOne(driver, ctx, wsRoot, confirm); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
		}
	}
	return nil
}

// pruneOne runs the full prune cycle for one driver+root.
func pruneOne(driver domain.LanguageDriver, ctx *domain.ProjectContext, root string, confirm bool) error {
	pipeline := analysis.New(driver)
	analyses, err := pipeline.DiscoverAndAnalyzeAll(root, ctx)
	if err != nil {
		return fmt.Errorf("analyse project: %w", err)
	}

	used := generation.ScanUsedDependencies(analyses)

	driverCfg := driver.GetTestFrameworkConfig()
	fixtureDir := fixtureRootFor(driverCfg, root)

	fixtures, err := generation.ScanExistingFixtures(fixtureDir, driverCfg)
	if err != nil {
		return fmt.Errorf("scan fixtures: %w", err)
	}

	unused := generation.IdentifyUnused(used, fixtures)

	if len(unused) == 0 {
		fmt.Println("  No unused fixtures found.")
		return nil
	}

	isDryRun := dryRun || !confirm
	results := generation.PruneFixtures(unused, isDryRun)

	var deleted []string
	for _, r := range results {
		switch r.Action {
		case "deleted":
			fmt.Printf("  ✓ deleted  %s_fixture\n", r.DepName)
			deleted = append(deleted, r.DepName)
		case "skipped":
			fmt.Printf("  · would delete  %s_fixture\n", r.DepName)
		case "error":
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", r.DepName, r.Err)
		}
	}

	if len(deleted) > 0 {
		modified, err := generation.UpdateTestImports(root, deleted)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: update imports: %v\n", err)
		}
		for _, f := range modified {
			rel, _ := filepath.Rel(root, f)
			fmt.Printf("  · commented out stale imports in %s\n", rel)
		}
	}

	if isDryRun {
		fmt.Printf("  %d unused fixture(s) found. Run with --confirm to delete.\n", len(unused))
	} else {
		fmt.Printf("  Pruned %d fixture(s).\n", len(deleted))
	}
	return nil
}

// fixtureRootFor resolves the absolute fixture directory path for a project root,
// using the driver's TestFrameworkConfig. Falls back to root when not set.
func fixtureRootFor(cfg domain.TestFrameworkConfig, root string) string {
	if cfg.FixtureDir == "" {
		return ""
	}
	return filepath.Join(root, cfg.FixtureDir)
}
