package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
)

func newMigrateCmd() *cobra.Command {
	var (
		from string
		to   string
		lang string
		path string
	)

	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Rewrite test files from one framework convention to another",
		Long: `Apply pattern-based transformation rules to migrate existing test files
between test framework conventions.

Examples:
  assay migrate --from jest --to vitest
  assay migrate --from junit4 --to junit5
  assay migrate --from pytest-mock --to unittest.mock
  assay migrate --from nunit --to xunit --dry-run
  assay migrate --from jest --to vitest --path src/components/`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMigrate(from, to, lang, path)
		},
	}

	cmd.Flags().StringVar(&from, "from", "", "source framework or mock library (required)")
	cmd.Flags().StringVar(&to, "to", "", "target framework or mock library (required)")
	cmd.Flags().StringVar(&lang, "lang", "", "override auto-detected language")
	cmd.Flags().StringVar(&path, "path", "", "restrict migration to files under this directory")
	_ = cmd.MarkFlagRequired("from")
	_ = cmd.MarkFlagRequired("to")

	return cmd
}

func runMigrate(from, to, langFlag, pathFlag string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	var driver domain.LanguageDriver
	var ctx *domain.ProjectContext

	if langFlag != "" {
		driver, err = reg.ForLanguage(langFlag)
		if err != nil {
			return err
		}
		ctx, err = driver.DetectProject(cwd)
		if err != nil {
			ctx = &domain.ProjectContext{Language: langFlag, Root: cwd, Metadata: map[string]any{}}
		}
	} else {
		driver, ctx, err = reg.Detect(cwd)
		if err != nil {
			return fmt.Errorf("detect project: %w", err)
		}
	}

	config.ApplyToContext(cfg, ctx)

	migrator := findMigrator(driver, from, to)
	if migrator == nil {
		return fmt.Errorf("no migration rule found for %q → %q in %s projects\n\nAvailable migrations:\n%s",
			from, to, ctx.Language, listAvailableMigrations(driver))
	}

	searchRoot := ctx.Root
	if pathFlag != "" {
		abs, _ := filepath.Abs(pathFlag)
		searchRoot = abs
	}

	files, err := discoverTestFiles(searchRoot, driver)
	if err != nil {
		return fmt.Errorf("discover test files: %w", err)
	}

	if len(files) == 0 {
		fmt.Println("No test files found to migrate.")
		return nil
	}

	if verbose {
		fmt.Printf("Language: %s | Migration: %s → %s | Files: %d\n\n",
			ctx.Language, from, to, len(files))
	}

	var migrated, skipped, failed int
	for _, f := range files {
		original, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", relPath(ctx.Root, f), err)
			failed++
			continue
		}

		result, err := migrator.MigrateFile(string(original))
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", relPath(ctx.Root, f), err)
			failed++
			continue
		}

		if result == string(original) {
			if verbose {
				fmt.Printf("  · unchanged  %s\n", relPath(ctx.Root, f))
			}
			skipped++
			continue
		}

		if dryRun {
			fmt.Printf("  ~ would migrate  %s\n", relPath(ctx.Root, f))
			migrated++
			continue
		}

		if err := os.WriteFile(f, []byte(result), 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", relPath(ctx.Root, f), err)
			failed++
			continue
		}

		fmt.Printf("  ✓ migrated  %s\n", relPath(ctx.Root, f))
		migrated++
	}

	fmt.Printf("\nProcessed %d file(s): %d migrated, %d unchanged, %d failed\n",
		len(files), migrated, skipped, failed)

	if failed > 0 {
		return fmt.Errorf("%d file(s) failed", failed)
	}
	return nil
}

// findMigrator returns the first migrator whose From/To match (case-insensitive).
func findMigrator(driver domain.LanguageDriver, from, to string) domain.Migrator {
	from = strings.ToLower(strings.TrimSpace(from))
	to = strings.ToLower(strings.TrimSpace(to))
	for _, m := range driver.ListMigrators() {
		if strings.ToLower(m.From()) == from && strings.ToLower(m.To()) == to {
			return m
		}
	}
	return nil
}

// listAvailableMigrations formats a human-readable list of available (from → to) pairs.
func listAvailableMigrations(driver domain.LanguageDriver) string {
	ms := driver.ListMigrators()
	if len(ms) == 0 {
		return "  (none)"
	}
	var sb strings.Builder
	for _, m := range ms {
		fmt.Fprintf(&sb, "  %s → %s\n", m.From(), m.To())
	}
	return sb.String()
}
