package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage TestSmith configuration",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	return cmd
}

// ── config init ───────────────────────────────────────────────────────────────

func newConfigInitCmd() *cobra.Command {
	var (
		lang    string
		yes     bool
		outFile string
	)

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Interactive wizard to create .testsmith.yaml",
		Long: `Walk through language detection, adapter selection, and optional LLM setup,
then write a commented .testsmith.yaml ready for 'testsmith generate'.

Examples:
  testsmith config init
  testsmith config init --lang java
  testsmith config init --yes          # accept all defaults, no prompts`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigInit(lang, yes, outFile)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "override auto-detected language")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "accept all defaults without prompting")
	cmd.Flags().StringVar(&outFile, "output", ".testsmith.yaml", "destination file path")
	return cmd
}

func runConfigInit(langFlag string, yes bool, outFile string) error {
	cwd, _ := os.Getwd()
	sc := bufio.NewScanner(os.Stdin)

	printHeader()

	// ── Step 1: language ──────────────────────────────────────────────────────

	var driver domain.LanguageDriver
	var ctx *domain.ProjectContext
	var err error

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
			// Offer a pick list of all known languages.
			driver, ctx, err = pickLanguage(sc, yes, cwd)
			if err != nil {
				return err
			}
		}
	}

	detectedLang := ctx.Language
	fmt.Printf("Language:  %s\n", detectedLang)
	fmt.Printf("Root:      %s\n\n", ctx.Root)

	if !yes {
		resp := prompt(sc, fmt.Sprintf("Detected language is %q — is this correct? [Y/n]", detectedLang), "y")
		if strings.ToLower(resp) == "n" {
			driver, ctx, err = pickLanguage(sc, yes, cwd)
			if err != nil {
				return err
			}
			detectedLang = ctx.Language
		}
	}

	// ── Step 2: adapter ───────────────────────────────────────────────────────

	available, selected := driver.ListAdapters(ctx)

	fmt.Println("Available adapters:")
	for i, a := range available {
		label := fmt.Sprintf("  %d  %-30s", i+1, a.Framework()+" + "+a.MockLibrary())
		note := ""
		if a.Framework() == selected.Framework() && a.MockLibrary() == selected.MockLibrary() {
			note = "  ← auto-detected"
		}
		fmt.Printf("%s%s\n", label, note)
	}

	chosenAdapter := selected
	if !yes && len(available) > 1 {
		defaultIdx := adapterIndex(available, selected) + 1
		raw := prompt(sc, fmt.Sprintf("\nAdapter [%d]", defaultIdx), fmt.Sprintf("%d", defaultIdx))
		if idx := parseChoice(raw, len(available)); idx >= 0 {
			chosenAdapter = available[idx]
		}
	}
	fmt.Printf("\nSelected: %s + %s\n\n", chosenAdapter.Framework(), chosenAdapter.MockLibrary())

	// ── Step 3: LLM ───────────────────────────────────────────────────────────

	defaults := config.Default()
	llmEnabled := false
	llmProvider := defaults.LLM.Provider
	llmModel := defaults.LLM.Model

	if !yes {
		resp := prompt(sc, "Enable LLM body generation? [y/N]", "n")
		if strings.ToLower(resp) == "y" {
			llmEnabled = true
			fmt.Println("\n  Providers:")
			fmt.Println("    1  anthropic  (Claude)")
			fmt.Println("    2  openai")
			fmt.Println("    3  ollama     (local)")
			provRaw := prompt(sc, "  Provider [1]", "1")
			switch strings.TrimSpace(provRaw) {
			case "2":
				llmProvider = "openai"
				llmModel = "gpt-4o"
			case "3":
				llmProvider = "ollama"
				llmModel = "llama3.2"
			default:
				llmProvider = "anthropic"
				llmModel = "claude-opus-4-7"
			}
			modelRaw := prompt(sc, fmt.Sprintf("  Model [%s]", llmModel), llmModel)
			if strings.TrimSpace(modelRaw) != "" {
				llmModel = strings.TrimSpace(modelRaw)
			}
			fmt.Println()
		}
	}

	// ── Step 4: exclude dirs ─────────────────────────────────────────────────

	excludeDirs := ctx.ExcludeDirs
	if len(excludeDirs) == 0 {
		excludeDirs = defaults.ExcludeDirs
	}

	if !yes {
		fmt.Printf("Exclude dirs (comma-separated, enter to keep defaults):\n")
		fmt.Printf("  %s\n", strings.Join(excludeDirs, ", "))
		raw := prompt(sc, ">", "")
		if strings.TrimSpace(raw) != "" {
			var dirs []string
			for _, d := range strings.Split(raw, ",") {
				d = strings.TrimSpace(d)
				if d != "" {
					dirs = append(dirs, d)
				}
			}
			excludeDirs = dirs
		}
		fmt.Println()
	}

	// ── Step 5: preview + write ───────────────────────────────────────────────

	content := renderConfigYAML(detectedLang, chosenAdapter, llmEnabled, llmProvider, llmModel, excludeDirs)

	fmt.Println(divider("Preview"))
	fmt.Println(content)
	fmt.Println(divider(""))

	absOut := outFile
	if !filepath.IsAbs(outFile) {
		absOut = filepath.Join(cwd, outFile)
	}

	if _, err := os.Stat(absOut); err == nil && !yes {
		resp := prompt(sc, fmt.Sprintf("%s already exists — overwrite? [y/N]", outFile), "n")
		if strings.ToLower(resp) != "y" {
			fmt.Println("Aborted — existing file kept.")
			return nil
		}
	}

	if dryRun {
		fmt.Printf("(dry-run) would write %s\n", absOut)
		return nil
	}

	if !yes {
		resp := prompt(sc, fmt.Sprintf("Write %s? [Y/n]", outFile), "y")
		if strings.ToLower(resp) == "n" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := os.WriteFile(absOut, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", absOut, err)
	}

	fmt.Printf("\n  ✓ wrote  %s\n", absOut)
	fmt.Printf("\nRun 'testsmith generate --all' to start generating tests.\n")
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────────

func printHeader() {
	fmt.Println(strings.Repeat("─", 52))
	fmt.Println("TestSmith › Config Wizard")
	fmt.Println(strings.Repeat("─", 52))
	fmt.Println()
}

func divider(label string) string {
	if label == "" {
		return strings.Repeat("─", 52)
	}
	return fmt.Sprintf("── %s %s", label, strings.Repeat("─", 48-len(label)))
}

// prompt prints a question and reads one line of user input.
// Returns def if the user presses enter without typing anything.
func prompt(sc *bufio.Scanner, question, def string) string {
	fmt.Printf("%s ", question)
	if !sc.Scan() {
		return def
	}
	text := strings.TrimSpace(sc.Text())
	if text == "" {
		return def
	}
	return text
}

// pickLanguage presents a numbered list of all registered languages and returns
// the chosen driver and a minimal ProjectContext.
func pickLanguage(sc *bufio.Scanner, yes bool, cwd string) (domain.LanguageDriver, *domain.ProjectContext, error) {
	langs := reg.Languages()
	fmt.Println("Could not auto-detect language. Available languages:")
	for i, l := range langs {
		fmt.Printf("  %d  %s\n", i+1, l)
	}

	choice := "1"
	if !yes {
		choice = prompt(sc, "Language [1]", "1")
	}
	idx := parseChoice(choice, len(langs))
	if idx < 0 {
		idx = 0
	}

	driver, err := reg.ForLanguage(langs[idx])
	if err != nil {
		return nil, nil, err
	}
	ctx, err := driver.DetectProject(cwd)
	if err != nil {
		ctx = &domain.ProjectContext{Language: langs[idx], Root: cwd, Metadata: map[string]any{}}
	}
	return driver, ctx, nil
}

func adapterIndex(available []domain.TestAdapter, target domain.TestAdapter) int {
	for i, a := range available {
		if a.Framework() == target.Framework() && a.MockLibrary() == target.MockLibrary() {
			return i
		}
	}
	return 0
}

// parseChoice converts "2" → 1 (0-based). Returns -1 on invalid input.
func parseChoice(raw string, max int) int {
	raw = strings.TrimSpace(raw)
	n := 0
	for _, c := range raw {
		if c < '0' || c > '9' {
			return -1
		}
		n = n*10 + int(c-'0')
	}
	if n < 1 || n > max {
		return -1
	}
	return n - 1
}

// renderConfigYAML produces a human-readable, commented .testsmith.yaml.
func renderConfigYAML(
	lang string,
	adapter domain.TestAdapter,
	llmEnabled bool,
	llmProvider, llmModel string,
	excludeDirs []string,
) string {
	var sb strings.Builder

	w := func(format string, args ...any) {
		fmt.Fprintf(&sb, format+"\n", args...)
	}

	w("# .testsmith.yaml — generated by 'testsmith config init'")
	w("# Run 'testsmith generate --all' to scaffold tests for every untested source file.")
	w("")
	w("# Primary language detected in this project.")
	w("language: %s", lang)
	w("")
	w("# Framework and mock library for each language.")
	w("# Run 'testsmith adapters list' to see all available options.")
	w("languages:")
	w("  %s:", lang)
	w("    framework: %s", adapter.Framework())
	w("    mock_library: %s", adapter.MockLibrary())
	w("")
	w("# Directories excluded from source discovery and analysis.")
	w("exclude_dirs:")
	for _, d := range excludeDirs {
		w("  - %s", d)
	}
	w("")
	w("# Optional LLM integration for generating test bodies.")
	w("# Requires an API key set in the environment variable below.")
	w("llm:")
	w("  enabled: %v", llmEnabled)
	w("  provider: %s  # anthropic | openai | ollama", llmProvider)
	w("  model: %s", llmModel)

	switch llmProvider {
	case "anthropic":
		w("  api_key_env_var: ANTHROPIC_API_KEY")
	case "openai":
		w("  api_key_env_var: OPENAI_API_KEY")
	case "ollama":
		w("  base_url: http://localhost:11434")
	}

	return sb.String()
}

// ── config show ───────────────────────────────────────────────────────────────

func newConfigShowCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show the resolved configuration for the current project",
		Long: `Display the fully-resolved configuration: built-in defaults merged with any
values from .testsmith.yaml, plus the effective test adapter that would be
selected for the detected (or specified) language.

Examples:
  testsmith config show
  testsmith config show --lang java`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runConfigShow(lang)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "show effective config for this language")
	return cmd
}

func runConfigShow(langFlag string) error {
	cwd, _ := os.Getwd()

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Resolve driver + context so we can show the effective adapter.
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
			// No project found — show global config only.
			ctx = nil
		}
	}

	if ctx != nil {
		config.ApplyToContext(cfg, ctx)
	}

	// ── header ────────────────────────────────────────────────────────────────

	if cfg.ConfigPath != "" {
		fmt.Printf("Config file:  %s\n", cfg.ConfigPath)
	} else {
		fmt.Printf("Config file:  (none found — showing built-in defaults)\n")
	}
	fmt.Println()

	// ── top-level fields ──────────────────────────────────────────────────────

	showField("language", cfg.Language)
	showField("test_root", cfg.TestRoot)
	showField("fixture_dir", cfg.FixtureDir)

	fmt.Printf("  %-20s %s\n", "exclude_dirs:", formatList(cfg.ExcludeDirs))
	fmt.Println()

	// ── per-language config ───────────────────────────────────────────────────

	if len(cfg.Languages) > 0 {
		fmt.Println("  languages:")
		// Show the detected language first, then the rest alphabetically.
		ordered := orderedLangKeys(cfg.Languages, langFlag)
		for _, l := range ordered {
			lc := cfg.Languages[l]
			fmt.Printf("    %s:\n", l)
			if lc.Framework != "" {
				fmt.Printf("      %-18s %s\n", "framework:", lc.Framework)
			}
			if lc.MockLibrary != "" {
				fmt.Printf("      %-18s %s\n", "mock_library:", lc.MockLibrary)
			}
			if lc.TestRoot != "" {
				fmt.Printf("      %-18s %s\n", "test_root:", lc.TestRoot)
			}
			if lc.FixtureDir != "" {
				fmt.Printf("      %-18s %s\n", "fixture_dir:", lc.FixtureDir)
			}
		}
		fmt.Println()
	}

	// ── LLM ──────────────────────────────────────────────────────────────────

	fmt.Println("  llm:")
	fmt.Printf("    %-18s %v\n", "enabled:", cfg.LLM.Enabled)
	fmt.Printf("    %-18s %s\n", "provider:", cfg.LLM.Provider)
	fmt.Printf("    %-18s %s\n", "model:", cfg.LLM.Model)
	if cfg.LLM.APIKeyEnvVar != "" {
		fmt.Printf("    %-18s %s\n", "api_key_env_var:", cfg.LLM.APIKeyEnvVar)
	}
	fmt.Println()

	// ── effective adapter ─────────────────────────────────────────────────────

	if ctx != nil && driver != nil {
		_, selected := driver.ListAdapters(ctx)
		reason := selectionReason(cfg, ctx, selected)
		fmt.Printf("  %-20s %s + %s  (%s)\n",
			"effective adapter:", selected.Framework(), selected.MockLibrary(), reason)
		fmt.Println()
	}

	return nil
}

func showField(name, value string) {
	if value == "" {
		return
	}
	fmt.Printf("  %-20s %s\n", name+":", value)
}

func formatList(items []string) string {
	if len(items) == 0 {
		return "(none)"
	}
	if len(items) <= 4 {
		return strings.Join(items, ", ")
	}
	return strings.Join(items[:4], ", ") + fmt.Sprintf(", … (%d total)", len(items))
}

// orderedLangKeys returns language keys with the primary language first.
func orderedLangKeys(langs map[string]config.LanguageConfig, primary string) []string {
	var first, rest []string
	for k := range langs {
		if k == primary {
			first = append(first, k)
		} else {
			rest = append(rest, k)
		}
	}
	// stable sort for rest
	for i := 0; i < len(rest)-1; i++ {
		for j := i + 1; j < len(rest); j++ {
			if rest[i] > rest[j] {
				rest[i], rest[j] = rest[j], rest[i]
			}
		}
	}
	return append(first, rest...)
}
