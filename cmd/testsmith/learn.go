package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/llm"
	"github.com/orieken/testsmith/internal/llm/factory"
	"github.com/orieken/testsmith/internal/projectknowledge"
)

const learnSystemPrompt = `You are a test-pattern extractor. Given a test file, identify the single most
reusable, non-obvious testing pattern it demonstrates — a mock setup, a fixture strategy, a
framework-specific workaround, or an unusual test structure. Do NOT describe standard patterns
that any reader of the framework documentation would already know.

Respond with exactly two sections:

filename: <kebab-case-slug>
(A descriptive filename without the .md extension. Examples: mock-database-sqlmock,
http-handler-httptest, postgres-testcontainers, table-driven-error-sentinel)

---

<pattern content>
(Markdown. Answer three questions: what is the pattern, when does it apply, and show
the minimal code example demonstrating it — stripped of business logic, with generic names.
Under 60 lines total.)`

var filenameRe = regexp.MustCompile(`(?m)^filename:\s*(.+)$`)

func newLearnCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "learn <test-file>",
		Short: "Extract a reusable test pattern from a file into .testsmith/patterns/",
		Long: `Read a corrected or hand-written test file, use the LLM to extract the
non-obvious testing pattern it demonstrates, and write the result into
.testsmith/patterns/ for human review and commit.

Requires LLM to be enabled in .testsmith.yaml (llm.enabled: true).

Examples:
  testsmith learn internal/payment/payment_test.go
  testsmith learn --dry-run src/services/user.test.ts`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLearn(args[0])
		},
	}
	return cmd
}

func runLearn(testFile string) error {
	cwd, _ := os.Getwd()

	content, err := os.ReadFile(testFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", testFile, err)
	}
	if len(content) == 0 {
		return fmt.Errorf("%s is empty", testFile)
	}

	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	provider, err := factory.BuildProvider(cfg.LLM)
	if err != nil {
		return fmt.Errorf("build LLM provider: %w", err)
	}

	userPrompt := fmt.Sprintf("Test file: %s\n\n```\n%s\n```", filepath.Base(testFile), string(content))

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := provider.Complete(ctx, llm.CompletionRequest{
		SystemPrompt: learnSystemPrompt,
		UserPrompt:   userPrompt,
		Model:        cfg.LLM.Model,
		MaxTokens:    1024,
		Temperature:  0.2,
	})
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	name, patternContent := parseLearnResponse(resp.Content)
	if name == "" {
		name = slugFromPath(testFile)
	}

	patternsDir := projectknowledge.PatternsDir(cwd)
	destPath := filepath.Join(patternsDir, name+".md")

	if dryRun {
		fmt.Printf("-- .testsmith/patterns/%s.md (dry-run) --\n%s\n", name, patternContent)
		return nil
	}

	if err := os.MkdirAll(patternsDir, 0o755); err != nil {
		return fmt.Errorf("create patterns directory: %w", err)
	}

	if _, statErr := os.Stat(destPath); statErr == nil {
		fmt.Printf("  – .testsmith/patterns/%s.md already exists — skipping. Delete it first to re-extract.\n", name)
		return nil
	}

	if err := os.WriteFile(destPath, []byte(patternContent), 0o644); err != nil {
		return fmt.Errorf("write pattern file: %w", err)
	}

	fmt.Printf("  ✓ created  .testsmith/patterns/%s.md\n", name)
	fmt.Println("  Review the file, edit if needed, then commit it.")
	return nil
}

// parseLearnResponse splits the LLM response into a filename slug and the
// markdown body. If the response does not contain the expected filename line,
// the slug is empty and the full response is returned as the body.
func parseLearnResponse(raw string) (slug, body string) {
	match := filenameRe.FindStringSubmatch(raw)
	if match == nil {
		return "", strings.TrimSpace(raw)
	}
	slug = strings.TrimSpace(match[1])
	// Strip the filename line and the following separator from the body.
	after := raw[strings.Index(raw, match[0])+len(match[0]):]
	after = strings.TrimPrefix(strings.TrimSpace(after), "---")
	return slug, strings.TrimSpace(after)
}

// slugFromPath derives a fallback pattern filename from the test file path.
func slugFromPath(path string) string {
	base := filepath.Base(path)
	// Strip known test suffixes before making the slug.
	for _, suffix := range []string{"_test.go", ".test.ts", ".test.js", ".spec.ts", ".spec.js", "Test.java", "Tests.swift"} {
		base = strings.TrimSuffix(base, suffix)
	}
	base = strings.ToLower(base)
	base = regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(base, "-")
	return strings.Trim(base, "-") + "-pattern"
}
