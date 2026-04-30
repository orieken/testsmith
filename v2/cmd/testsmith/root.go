package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
	dryRun  bool
)

var rootCmd = &cobra.Command{
	Use:   "testsmith",
	Short: "Language-agnostic test scaffold generator",
	Long: `TestSmith v2 — generate test scaffolds for any language from a single binary.

Supported languages: Python, TypeScript/JavaScript, Go, Java, C#
Run 'testsmith help <command>' for detailed usage of each subcommand.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "path to .testsmith.yaml (default: search up from cwd)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "emit detailed analysis and generation output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "print the generation plan without writing any files")

	rootCmd.AddCommand(
		newGenerateCmd(),
		newAdaptersCmd(),
		newConfigCmd(),
		newMigrateCmd(),
		newValidateCmd(),
		newGraphCmd(),
		newPruneCmd(),
		newGapsCmd(),
		newWatchCmd(),
		newInitCmd(),
		newVersionCmd(),
		newCompletionCmd(),
	)
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
