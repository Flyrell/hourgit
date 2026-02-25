package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var defaultsResetCmd = LeafCommand{
	Use:   "reset",
	Short: "Reset the default schedule to factory settings (Mon-Fri 9am-5pm)",
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompt"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		confirm := ResolveConfirmFunc(yes)

		return runDefaultsReset(cmd, homeDir, confirm)
	},
}.Build()

func runDefaultsReset(cmd *cobra.Command, homeDir string, confirm ConfirmFunc) error {
	confirmed, err := confirm("Reset defaults to factory settings (Mon-Fri 9am-5pm)?")
	if err != nil {
		return err
	}
	if !confirmed {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
		return nil
	}

	if err := project.ResetDefaults(homeDir); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text("defaults reset to factory settings"))
	return nil
}
