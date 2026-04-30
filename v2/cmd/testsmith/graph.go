package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/orieken/testsmith/internal/domain"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	var output string
	var workspace string

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Generate a dependency graph visualisation",
		Long: `Scan all source files, compute module metrics, and write a Mermaid
dependency graph plus a coupling-score table to a Markdown file.
When workspaces are configured in .testsmith.yaml, each workspace
produces a labelled section in the same report.

Examples:
  testsmith graph
  testsmith graph --output deps.md
  testsmith graph --workspace api`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(output, workspace)
		},
	}

	cmd.Flags().StringVar(&output, "output", "testsmith_graph.md", "output Markdown file")
	cmd.Flags().StringVar(&workspace, "workspace", "", "graph only this workspace (name or path)")
	return cmd
}

func runGraph(output, wsFilter string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	if len(cfg.Workspaces) > 0 {
		return runGraphWorkspaces(cfg, cwd, output, wsFilter)
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

	section, err := buildGraphSection("", ctx.Root, driver, ctx)
	if err != nil {
		return err
	}

	return writeGraphReport("# TestSmith Dependency Graph\n\n"+section, output, cwd)
}

func runGraphWorkspaces(cfg *config.Config, cwd, output, wsFilter string) error {
	var sb strings.Builder
	sb.WriteString("# TestSmith Dependency Graph\n\n")

	for i := range cfg.Workspaces {
		ws := &cfg.Workspaces[i]
		id := config.WorkspaceID(ws)
		if wsFilter != "" && id != wsFilter && ws.Path != wsFilter {
			continue
		}

		wsRoot := filepath.Join(cwd, ws.Path)
		driver, ctx, err := resolveWorkspaceDriver(ws, wsRoot)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
			continue
		}
		ctx.ExcludeDirs = append(ctx.ExcludeDirs, cfg.ExcludeDirs...)
		config.ApplyToContext(cfg, ctx)

		fmt.Fprintf(&sb, "## Workspace: %s\n\n", id)
		section, err := buildGraphSection(id, wsRoot, driver, ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  ✗ workspace %s: %v\n", id, err)
			continue
		}
		sb.WriteString(section)
	}

	return writeGraphReport(sb.String(), output, cwd)
}

// buildGraphSection analyses one root and returns the Markdown section content.
func buildGraphSection(label, root string, driver domain.LanguageDriver, ctx *domain.ProjectContext) (string, error) {
	pipeline := analysis.New(driver)
	analyses, err := pipeline.DiscoverAndAnalyzeAll(root, ctx)
	if err != nil {
		return "", fmt.Errorf("analyse %s: %w", root, err)
	}

	graph := analysis.BuildDependencyGraph(analyses)
	metrics := analysis.ComputeMetrics(graph)

	var sb strings.Builder
	sb.WriteString(analysis.RenderMetricsTable(metrics))
	sb.WriteString("\n")
	sb.WriteString(analysis.RenderMermaid(graph))

	if verbose && label != "" {
		fmt.Printf("  %s: %d node(s), %d edge(s)\n", label, len(graph.Nodes), len(graph.Edges))
	}
	return sb.String(), nil
}

func writeGraphReport(content, output, root string) error {
	if dryRun {
		fmt.Print(content)
		return nil
	}

	absOutput, _ := filepath.Abs(output)
	if err := os.WriteFile(absOutput, []byte(content), 0o644); err != nil {
		return fmt.Errorf("write report: %w", err)
	}

	rel, _ := filepath.Rel(root, absOutput)
	fmt.Printf("Dependency graph written to %s\n", rel)
	return nil
}
