package main

import (
	"fmt"
	"os"

	"github.com/orieken/testsmith/internal/analysis"
	"github.com/orieken/testsmith/internal/config"
	"github.com/spf13/cobra"
)

func newGraphCmd() *cobra.Command {
	var output string

	cmd := &cobra.Command{
		Use:   "graph",
		Short: "Generate a dependency graph visualisation",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGraph(output)
		},
	}

	cmd.Flags().StringVar(&output, "output", "testsmith_graph.md", "output Markdown file")
	return cmd
}

func runGraph(output string) error {
	cwd, _ := os.Getwd()
	cfg, err := config.Load(cwd)
	if err != nil {
		return err
	}
	_ = cfg

	driver, ctx, err := reg.Detect(cwd)
	if err != nil {
		return fmt.Errorf("detect project: %w", err)
	}

	pipeline := analysis.New(driver)
	analyses, err := pipeline.DiscoverAndAnalyzeAll(ctx.Root, ctx)
	if err != nil {
		return err
	}

	graph := analysis.BuildDependencyGraph(analyses)
	metrics := analysis.ComputeMetrics(graph)

	metricsTable := analysis.RenderMetricsTable(metrics)
	mermaid := analysis.RenderMermaid(graph)
	content := "# TestSmith Dependency Graph\n\n" + metricsTable + "\n\n" + mermaid

	if err := os.WriteFile(output, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Dependency graph written to: %s\n\n", output)
	fmt.Println(metricsTable)
	return nil
}
