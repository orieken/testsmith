package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var lang string

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold test directories and .testsmith.yaml",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(lang)
		},
	}

	cmd.Flags().StringVar(&lang, "lang", "", "hint the primary language if auto-detection fails")
	return cmd
}

func runInit(lang string) error {
	// Phase 2 implementation target.
	fmt.Println("init: not yet implemented — Phase 2")
	return nil
}
