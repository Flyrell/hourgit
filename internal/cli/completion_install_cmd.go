package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var completionInstallCmd = LeafCommand{
	Use:   "install [SHELL]",
	Short: "Install shell completions into your shell config",
	Args:  cobra.RangeArgs(0, 1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "Skip confirmation prompt"},
	},
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

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		confirm := ResolveConfirmFunc(yes)

		return runCompletionInstall(cmd, shell, homeDir, confirm)
	},
}.Build()

func runCompletionInstall(cmd *cobra.Command, shell, homeDir string, confirm ConfirmFunc) error {
	if isCompletionInstalled(shell, homeDir) {
		configFile := shellConfigs[shell]
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("shell completions already installed for %s in %s", Primary(shell), Primary(filepath.Join("~", configFile)))))
		return nil
	}

	configFile, ok := shellConfigs[shell]
	if !ok {
		return fmt.Errorf("unsupported shell for completion install: %s", shell)
	}

	ok, err := confirm(fmt.Sprintf("Install shell completions for %s into %s?", shell, filepath.Join("~", configFile)))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	if err := installCompletion(shell, homeDir); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("shell completions installed for %s in %s", Primary(shell), Primary(filepath.Join("~", configFile)))))
	return nil
}
