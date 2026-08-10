package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/orieken/assay/internal/agents"
	"github.com/orieken/assay/internal/config"
	"github.com/orieken/assay/internal/projectknowledge"
)

const patternsReadme = `# .assay/patterns

Drop one markdown file per reusable testing pattern here.
assay merges these files into every LLM prompt so generated tests automatically
apply your project's established solutions — no more rediscovering the same mock setup.

Run ` + "`assay learn <test-file>`" + ` to extract a pattern from an existing test file.

## File naming convention

  <what-is-mocked-or-set-up>-<how>.md

Examples:
  mock-database-sqlmock.md
  http-handler-httptest.md
  postgres-testcontainers.md
  table-driven-error-sentinel.md

## Pattern file format

Each file should answer three questions:
1. What is the pattern?
2. When does it apply?
3. Minimal code example (generic names, no business logic, under 40 lines).
`

func newInitCmd() *cobra.Command {
	var lang string
	var withAgents bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold test directories and .assay.yaml",
		Long: `Detect the project language, create the test and fixture directories,
and write a .assay.yaml with sensible defaults.

Run this once at the root of a new project before using 'assay generate'.

Examples:
  assay init
  assay init --lang python
  assay init --with-agents`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(lang, withAgents)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "hint the primary language if auto-detection fails")
	cmd.Flags().BoolVar(&withAgents, "with-agents", false, "write bundled Claude Code agent files into .claude/agents/")
	return cmd
}

func runInit(langHint string, withAgents bool) error {
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

	// Write .assay.yaml if it doesn't already exist.
	cfgPath := filepath.Join(cwd, ".assay.yaml")
	if _, err := os.Stat(cfgPath); err == nil {
		fmt.Println(".assay.yaml already exists — skipping.")
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
		fmt.Printf("\n-- .assay.yaml (dry-run) --\n%s", string(data))
		return nil
	}

	if err := os.WriteFile(cfgPath, data, 0o644); err != nil {
		return fmt.Errorf("write .assay.yaml: %w", err)
	}
	fmt.Println("  ✓ created  .assay.yaml")

	// Scaffold ASSAY.md — the project knowledge file the LLM reads for every
	// file it generates tests for. Skip if it already exists.
	knowledgePath := filepath.Join(cwd, "ASSAY.md")
	if _, err := os.Stat(knowledgePath); errors.Is(err, os.ErrNotExist) {
		if !dryRun {
			tmpl := projectknowledge.Template(detectedLang)
			if err := os.WriteFile(knowledgePath, []byte(tmpl), 0o644); err != nil {
				return fmt.Errorf("write ASSAY.md: %w", err)
			}
			fmt.Println("  ✓ created  ASSAY.md")
		} else {
			fmt.Printf("\n-- ASSAY.md (dry-run) --\n%s\n", projectknowledge.Template(detectedLang))
		}
	} else {
		fmt.Println("ASSAY.md already exists — skipping.")
	}

	// Scaffold .assay/patterns/ — the directory for captured test patterns.
	patternsDir := projectknowledge.PatternsDir(cwd)
	if _, err := os.Stat(patternsDir); errors.Is(err, os.ErrNotExist) {
		if !dryRun {
			if err := os.MkdirAll(patternsDir, 0o755); err != nil {
				return fmt.Errorf("create .assay/patterns: %w", err)
			}
			readmePath := filepath.Join(patternsDir, "README.md")
			if err := os.WriteFile(readmePath, []byte(patternsReadme), 0o644); err != nil {
				return fmt.Errorf("write patterns README: %w", err)
			}
			fmt.Println("  ✓ created  .assay/patterns/")
		} else {
			fmt.Println("  ✓ would create  .assay/patterns/")
		}
	} else {
		fmt.Println(".assay/patterns/ already exists — skipping.")
	}

	if withAgents {
		if err := writeAgents(cwd); err != nil {
			return err
		}
	}

	fmt.Printf("\nInitialised %s project. Edit ASSAY.md with your conventions, then run 'assay generate --all'.\n", detectedLang)
	return nil
}

func writeAgents(root string) error {
	agentDir := filepath.Join(root, ".claude", "agents")
	if err := os.MkdirAll(agentDir, 0o755); err != nil {
		return fmt.Errorf("create .claude/agents: %w", err)
	}

	files, err := agents.All()
	if err != nil {
		return fmt.Errorf("load bundled agents: %w", err)
	}

	for name, content := range files {
		dest := filepath.Join(agentDir, name)
		if _, statErr := os.Stat(dest); statErr == nil {
			fmt.Printf("  – skipped  .claude/agents/%s (already exists)\n", name)
			continue
		}
		if dryRun {
			fmt.Printf("  ✓ would write  .claude/agents/%s\n", name)
			continue
		}
		if err := os.WriteFile(dest, content, 0o644); err != nil {
			return fmt.Errorf("write agent %s: %w", name, err)
		}
		fmt.Printf("  ✓ created  .claude/agents/%s\n", name)
	}
	return nil
}
