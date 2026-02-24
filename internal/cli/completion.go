package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validShells = []string{"bash", "zsh", "fish", "powershell"}

var completionCmd = GroupCommand{
	Use:   "completion",
	Short: "Manage shell completions",
	Subcommands: []*cobra.Command{
		completionGenerateCmd,
		completionInstallCmd,
	},
}.Build()

var completionGenerateCmd = newCompletionGenerateCmd()

func newCompletionGenerateCmd() *cobra.Command {
	cmd := LeafCommand{
		Use:   "generate [SHELL]",
		Short: "Generate shell completion script",
		Args:  cobra.RangeArgs(0, 1),
		RunE: func(cmd *cobra.Command, args []string) error {
			shell := ""
			if len(args) > 0 {
				shell = args[0]
			} else {
				shell = detectShell()
				if shell == "" {
					return fmt.Errorf("could not detect shell from $SHELL environment variable; please specify one explicitly (bash, zsh, fish, powershell)")
				}
			}
			return runCompletion(cmd, shell)
		},
	}.Build()
	cmd.ValidArgs = validShells
	return cmd
}

func runCompletion(cmd *cobra.Command, shell string) error {
	root := cmd.Root()
	out := cmd.OutOrStdout()

	switch shell {
	case "bash":
		return root.GenBashCompletionV2(out, true)
	case "zsh":
		return root.GenZshCompletion(out)
	case "fish":
		return root.GenFishCompletion(out, true)
	case "powershell":
		return root.GenPowerShellCompletion(out)
	default:
		return fmt.Errorf("unsupported shell: %s (valid: bash, zsh, fish, powershell)", shell)
	}
}
