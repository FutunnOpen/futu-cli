package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// Shell names for completion generation.
const (
	shellBash       = "bash"
	shellZsh        = "zsh"
	shellFish       = "fish"
	shellPowershell = "powershell"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion scripts",
	Long: `Generate shell completion scripts for the Futu CLI.

To load completions:

Bash:
  $ source <(futu completion bash)
  # To load completions for each session, execute once:
  # Linux:
  $ futu completion bash > /etc/bash_completion.d/futu
  # macOS:
  $ futu completion bash > $(brew --prefix)/etc/bash_completion.d/futu

Zsh:
  $ source <(futu completion zsh)
  # To load completions for each session, execute once:
  $ futu completion zsh > "${fpath[1]}/_futu"

Fish:
  $ futu completion fish | source
  # To load completions for each session, execute once:
  $ futu completion fish > ~/.config/fish/completions/futu.fish

PowerShell:
  PS> futu completion powershell | Out-String | Invoke-Expression
  # To load completions for each session, add output to your profile.
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{shellBash, shellZsh, shellFish, shellPowershell},
	Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
	RunE: func(cmd *cobra.Command, args []string) error {
		return generateCompletion(args[0])
	},
}

func init() {
	rootCmd.AddCommand(completionCmd)
}

// generateCompletion dispatches to the appropriate cobra completion generator.
func generateCompletion(shell string) error {
	switch shell {
	case shellBash:
		return rootCmd.GenBashCompletion(os.Stdout)
	case shellZsh:
		return rootCmd.GenZshCompletion(os.Stdout)
	case shellFish:
		return rootCmd.GenFishCompletion(os.Stdout, true)
	case shellPowershell:
		return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
	default:
		return nil
	}
}
