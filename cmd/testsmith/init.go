package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/projectknowledge"
)

func newInitCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold test directories and .testsmith.yaml",
		Long: `Detect the project language, create the test and fixture directories,
and write a .testsmith.yaml with sensible defaults.

Run this once at the root of a new project before using 'testsmith generate'.

Examples:
  testsmith init
  testsmith init --lang python`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(lang)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "hint the primary language if auto-detection fails")
	return cmd
}

func runInit(langHint string) error {
	cwd, _ := os.Getwd()

	// Try auto-detection; fall back to the user hint.
	var detectedLang string
	var testRoot, fixtureDir string

	if langHint != "" {
		detectedLang = langHint
	} else if driver, _, err := reg.Detect(cwd); err == nil {
		detectedLang = driver.Language()
	} else {
		return fmt.Errorf("could not detect project language — use --lang to specify")
	}

	defaults := config.Default()
	if lc, ok := defaults.Languages[detectedLang]; ok {
		testRoot = lc.TestRoot
		fixtureDir = lc.FixtureDir
	} else {
		testRoot = defaults.TestRoot
		fixtureDir = defaults.FixtureDir
	}

	// Create directories.
	for _, dir := range []string{testRoot, fixtureDir} {
		if dir == "" {
			continue
		}
		abs := filepath.Join(cwd, dir)
		if err := os.MkdirAll(abs, 0o755); err != nil {
			return fmt.Errorf("create directory %s: %w", dir, err)
		}
		fmt.Printf("  ✓ created  %s\n", dir)
	}

	// Write .testsmith.yaml if it doesn't already exist.
	cfgPath := filepath.Join(cwd, ".testsmith.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Println(".testsmith.yaml already exists — skipping.")
		return nil
	}

	cfg := &config.Config{
		Language:    detectedLang,
		TestRoot:    testRoot,
		FixtureDir:  fixtureDir,
		ExcludeDirs: defaults.ExcludeDirs,
		LLM: config.LLMConfig{
			Enabled:      false,
			Provider:     defaults.LLM.Provider,
			Model:        defaults.LLM.Model,
			APIKeyEnvVar: defaults.LLM.APIKeyEnvVar,
		},
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if dryRun {
		fmt.Printf("\n-- .testsmith.yaml (dry-run) --\n%s", string(data))
		return nil
	}

	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("write .testsmith.yaml: %w", err)
	}
	fmt.Println("  ✓ created  .testsmith.yaml")

	// Scaffold TESTSMITH.md — the project knowledge file the LLM reads for every
	// file it generates tests for. Skip if it already exists.
	knowledgePath := filepath.Join(cwd, "TESTSMITH.md")
	if _, err := os.Stat(knowledgePath); errors.Is(err, os.ErrNotExist) {
		if !dryRun {
			tmpl := projectknowledge.Template(detectedLang)
			if err := os.WriteFile(knowledgePath, []byte(tmpl), 0o644); err != nil {
				return fmt.Errorf("write TESTSMITH.md: %w", err)
			}
			fmt.Println("  ✓ created  TESTSMITH.md")
		} else {
			fmt.Printf("\n-- TESTSMITH.md (dry-run) --\n%s\n", projectknowledge.Template(detectedLang))
		}
	} else {
		fmt.Println("TESTSMITH.md already exists — skipping.")
	}

	fmt.Printf("\nInitialised %s project. Edit TESTSMITH.md with your conventions, then run 'testsmith generate --all'.\n", detectedLang)
	return nil
}
