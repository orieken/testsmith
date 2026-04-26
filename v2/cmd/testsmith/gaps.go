package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newGapsCmd() *cobra.Command {
	var output string
	var top int

	cmd := &cobra.Command{
		Use:   "gaps",
		Short: "Analyse and report test coverage gaps",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGaps(output, top)
		},
	}

	cmd.Flags().StringVar(&output, "output", "testsmith_coverage_report.md", "output Markdown file")
	cmd.Flags().IntVar(&top, "top", 0, "show only top N gaps (0 = show all)")
	return cmd
}

func runGaps(output string, top int) error {
	// Phase 2 implementation target.
	fmt.Println("gaps: not yet implemented — Phase 2")
	return nil
}
