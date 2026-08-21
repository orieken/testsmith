package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/domain"
)

func newAdaptersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "adapters",
		Short: "Manage and inspect test framework adapters",
	}
	cmd.AddCommand(newAdaptersListCmd())
	return cmd
}

func newAdaptersListCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available adapters for the current (or specified) language",
		Long: `Show all built-in test framework adapters for the detected language,
highlighting which one is currently selected and why.

Examples:
  assay adapters list
  assay adapters list --lang java`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAdaptersList(lang)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "show adapters for this language instead of auto-detecting")
	return cmd
}

func runAdaptersList(langFlag string) error {
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
			// No project found — use a minimal context so selection still works.
			ctx = &domain.ProjectContext{Language: langFlag, Metadata: map[string]any{}}
		}
	} else {
		driver, ctx, err = reg.Detect(cwd)
		if err != nil {
			return fmt.Errorf("detect project: %w", err)
		}
	}

	// Apply config overrides so selection reflects what the user has set.
	config.ApplyToContext(cfg, ctx)

	available, selected := driver.ListAdapters(ctx)

	selectionNote := selectionReason(cfg, ctx, selected)

	fmt.Printf("Language:  %s\n", ctx.Language)
	fmt.Printf("Root:      %s\n\n", ctx.Root)

	// Column widths.
	const (
		colAdapter = 32
		colFW      = 14
		colMock    = 18
	)

	header := fmt.Sprintf("  %-*s  %-*s  %-*s  %s",
		colAdapter, "ADAPTER",
		colFW, "FRAMEWORK",
		colMock, "MOCK LIBRARY",
		"ACTIVE",
	)
	fmt.Println(header)
	fmt.Println("  " + strings.Repeat("─", len(header)-2))

	for _, a := range available {
		adapterLabel := a.Framework() + " + " + a.MockLibrary()
		marker := ""
		if a.Framework() == selected.Framework() && a.MockLibrary() == selected.MockLibrary() {
			marker = "✓  " + selectionNote
		}
		fmt.Printf("  %-*s  %-*s  %-*s  %s\n",
			colAdapter, adapterLabel,
			colFW, a.Framework(),
			colMock, a.MockLibrary(),
			marker,
		)
	}

	fmt.Printf("\nTo override, add to .assay.yaml:\n")
	fmt.Printf("  languages:\n")
	fmt.Printf("    %s:\n", ctx.Language)
	fmt.Printf("      framework: %s\n", selected.Framework())
	fmt.Printf("      mock_library: %s\n", selected.MockLibrary())

	return nil
}

// selectionReason returns a human-readable note explaining why the given adapter
// was selected (config override, auto-detected, or default).
func selectionReason(cfg *config.Config, ctx *domain.ProjectContext, selected domain.TestAdapter) string {
	if cfg.ConfigPath != "" {
		langCfg, hasLangCfg := cfg.Languages[ctx.Language]
		if hasLangCfg && (langCfg.Framework != "" || langCfg.MockLibrary != "") {
			return "from config"
		}
	}
	fw, _ := ctx.Metadata["framework"].(string)
	ml, _ := ctx.Metadata["mock_library"].(string)
	if fw == selected.Framework() && ml == selected.MockLibrary() && (fw != "" || ml != "") {
		return "auto-detected"
	}
	return "default"
}
