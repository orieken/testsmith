package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newPruneCmd() *cobra.Command {
	var confirm bool

	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Identify and remove unused fixture files",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrune(confirm)
		},
	}

	cmd.Flags().BoolVar(&confirm, "confirm", false, "actually delete (default: dry-run)")
	return cmd
}

func runPrune(confirm bool) error {
	// Phase 2 implementation target.
	fmt.Println("prune: not yet implemented — Phase 2")
	return nil
}
