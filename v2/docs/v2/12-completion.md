# Feature: completion — Shell Completion Scripts

## What It Does

`testsmith completion` generates shell completion scripts for bash, zsh, fish, and PowerShell. Once installed, pressing `<Tab>` completes subcommand names, flag names, and flag values (e.g. `--lang` choices).

Completion is provided natively by Cobra's built-in generator — no extra dependencies.

---

## CLI

```
testsmith completion [bash|zsh|fish|powershell]
```

---

## Installation

### Bash

```bash
# Load for the current session only:
source <(testsmith completion bash)

# Install permanently (Linux):
testsmith completion bash > /etc/bash_completion.d/testsmith

# Install permanently (macOS with Homebrew bash-completion@2):
testsmith completion bash > $(brew --prefix)/etc/bash_completion.d/testsmith
```

### Zsh

```zsh
# Enable completions if not already done (add to ~/.zshrc):
autoload -U compinit; compinit

# Install:
testsmith completion zsh > "${fpath[1]}/_testsmith"
```

### Fish

```fish
# Load for current session:
testsmith completion fish | source

# Install permanently:
testsmith completion fish > ~/.config/fish/completions/testsmith.fish
```

### PowerShell

```powershell
testsmith completion powershell | Out-String | Invoke-Expression
```

---

## Go Implementation

The completion command delegates entirely to Cobra's built-in generators:

```go
// cmd/testsmith/completion.go
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
```

`ValidArgs` is set to `["bash", "zsh", "fish", "powershell"]` and `cobra.OnlyValidArgs` is used so passing an unknown shell name exits non-zero.

### Files Involved

| File | Role |
|------|------|
| `cmd/testsmith/completion.go` | Cobra subcommand, shell dispatch |
