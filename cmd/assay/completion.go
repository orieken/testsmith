package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for assay.

Bash:
  # Load for this session:
  source <(assay completion bash)

  # Install permanently (Linux):
  assay completion bash > /etc/bash_completion.d/assay

  # Install permanently (macOS with Homebrew bash-completion@2):
  assay completion bash > $(brew --prefix)/etc/bash_completion.d/assay

Zsh:
  # Enable completions if not already done:
  echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Install:
  assay completion zsh > "${fpath[1]}/_assay"

Fish:
  assay completion fish | source

  # Install permanently:
  assay completion fish > ~/.config/fish/completions/assay.fish

PowerShell:
  assay completion powershell | Out-String | Invoke-Expression`,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		DisableFlagsInUseLine: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}
