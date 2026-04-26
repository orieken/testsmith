package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var llmFlag bool
	var debounce int

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch for file changes and auto-regenerate tests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(llmFlag, debounce)
		},
	}

	cmd.Flags().BoolVar(&llmFlag, "llm", false, "enable LLM on watched changes")
	cmd.Flags().IntVar(&debounce, "debounce", 500, "debounce interval in ms")
	return cmd
}

func runWatch(llmFlag bool, debounce int) error {
	// Phase 2 implementation target.
	fmt.Println("watch: not yet implemented — Phase 2")
	return nil
}
