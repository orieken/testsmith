package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/orieken/testsmith/internal/generation"
)

func newValidateCmd() *cobra.Command {
	var (
		lang      string
		path      string
		workspace string
		format    string
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Check test files against the configured adapter's conventions",
		Long: `Scan existing test files and report mismatches against the adapter that is
currently configured (or auto-detected) for the project.
When workspaces are configured in .testsmith.yaml, all workspaces are validated.

Examples:
  testsmith validate
  testsmith validate --path src/services/
  testsmith validate --lang java
  testsmith validate --format json
  testsmith validate --format junit
  testsmith validate --workspace api`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runValidate(lang, path, workspace, format)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "override auto-detected language")
	cmd.Flags().StringVar(&path, "path", "", "restrict validation to files under this directory")
	cmd.Flags().StringVar(&workspace, "workspace", "", "validate only this workspace (name or path)")
	cmd.Flags().StringVar(&format, "format", "text", "output format: text, json, junit")
	return cmd
}

func runValidate(langFlag, pathFlag, wsFilter, format string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Workspace mode.
	if len(cfg.Workspaces) > 0 && langFlag == "" && pathFlag == "" {
		return runValidateWorkspaces(cfg, cwd, wsFilter, format)
	}

	// Single-project mode.
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

	searchRoot := ctx.Root
	if pathFlag != "" {
		abs, _ := filepath.Abs(pathFlag)
		searchRoot = abs
	}

	errors := validateRoot(driver, ctx, searchRoot, format)
	if errors > 0 {
		return fmt.Errorf("validation failed: %d error(s) found", errors)
	}
	return nil
}

// runValidateWorkspaces validates all (or one filtered) workspace.
func runValidateWorkspaces(cfg *config.Config, cwd, wsFilter, format string) error {
	var grandErrors int

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
		config.ApplyToContext(cfg, ctx)

		errs := validateRoot(driver, ctx, wsRoot, format)
		grandErrors += errs
	}

	if grandErrors > 0 {
		return fmt.Errorf("validation failed: %d error(s) across workspaces", grandErrors)
	}
	return nil
}

// validateRoot runs the validation loop for one driver+root.
// It dispatches output to the requested format and returns the error count.
func validateRoot(driver domain.LanguageDriver, ctx *domain.ProjectContext, searchRoot, format string) int {
	_, selected := driver.ListAdapters(ctx)
	if selected == nil {
		fmt.Fprintf(os.Stderr, "  no adapter selected for %s project\n", ctx.Language)
		return 0
	}

	framework := selected.Framework()
	mockLib := selected.MockLibrary()

	if verbose && format == "text" {
		fmt.Printf("Language: %s | Adapter: %s + %s\n\n", ctx.Language, framework, mockLib)
	}

	files, err := discoverValidationFiles(searchRoot, driver)
	if err != nil {
		fmt.Fprintf(os.Stderr, "  discover error: %v\n", err)
		return 0
	}

	if len(files) == 0 {
		if format == "text" {
			fmt.Println("  No test files found.")
		}
		return 0
	}

	results, totalErrors := collectValidationResults(files, driver, framework, mockLib, ctx.Root)

	switch format {
	case "json":
		fmt.Print(generation.GenerateValidateJSON(results))
	case "junit":
		fmt.Print(generation.GenerateValidateJUnit(results))
	default:
		printValidationText(results, totalErrors, ctx.Root)
	}

	return totalErrors
}

// collectValidationResults reads every file, calls ValidateFile, and returns
// structured results alongside the total error count.
func collectValidationResults(
	files []string,
	driver domain.LanguageDriver,
	framework, mockLib, projectRoot string,
) ([]generation.ValidateFileResult, int) {
	results := make([]generation.ValidateFileResult, 0, len(files))
	var totalErrors int

	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ %s: %v\n", relPath(projectRoot, f), err)
			continue
		}

		issues := driver.ValidateFile(framework, mockLib, string(content))
		results = append(results, generation.ValidateFileResult{
			FilePath: relPath(projectRoot, f),
			Issues:   issues,
		})

		errs, _ := countBySeverity(issues)
		totalErrors += errs
	}

	return results, totalErrors
}

// printValidationText renders results in the existing human-readable format.
func printValidationText(results []generation.ValidateFileResult, totalErrors int, _ string) {
	var totalWarnings, filesWithIssues int

	for _, r := range results {
		errs, warns := countBySeverity(r.Issues)
		totalWarnings += warns

		if len(r.Issues) == 0 {
			if verbose {
				fmt.Printf("  ✓ %s\n", r.FilePath)
			}
			continue
		}

		filesWithIssues++
		fmt.Printf("  %s %s\n", issueIndicator(errs, warns), r.FilePath)
		for _, iss := range r.Issues {
			fmt.Printf("      [%-7s] %s: %s\n", iss.Severity, iss.Rule, iss.Message)
		}
	}

	fmt.Printf("\nChecked %d file(s): %d error(s), %d warning(s) across %d file(s)\n",
		len(results), totalErrors, totalWarnings, filesWithIssues)
}

func issueIndicator(errors, warnings int) string {
	if errors > 0 {
		return "✗"
	}
	if warnings > 0 {
		return "⚠"
	}
	return "·"
}

func countBySeverity(issues []domain.ValidationIssue) (errors, warnings int) {
	for _, iss := range issues {
		switch iss.Severity {
		case domain.SeverityError:
			errors++
		case domain.SeverityWarning:
			warnings++
		case domain.SeverityInfo:
			// info-level issues are not counted toward error/warning totals
		}
	}
	return
}
