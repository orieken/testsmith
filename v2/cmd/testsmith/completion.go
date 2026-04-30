package main

import (
	"os"

	"github.com/spf13/cobra"
)

func newCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion scripts",
		Long: `Generate shell completion scripts for testsmith.

Bash:
  # Load for this session:
  source <(testsmith completion bash)

  # Install permanently (Linux):
  testsmith completion bash > /etc/bash_completion.d/testsmith

  # Install permanently (macOS with Homebrew bash-completion@2):
  testsmith completion bash > $(brew --prefix)/etc/bash_completion.d/testsmith

Zsh:
  # Enable completions if not already done:
  echo "autoload -U compinit; compinit" >> ~/.zshrc

  # Install:
  testsmith completion zsh > "${fpath[1]}/_testsmith"

Fish:
  testsmith completion fish | source

  # Install permanently:
  testsmith completion fish > ~/.config/fish/completions/testsmith.fish

PowerShell:
  testsmith completion powershell | Out-String | Invoke-Expression`,
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
