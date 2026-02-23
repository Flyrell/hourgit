package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/spf13/cobra"
)

var configResetCmd = LeafCommand{
	Use:   "reset",
	Short: "Reset a project's schedule to the default (Mon-Fri 9am-5pm)",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompt"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()
		projectFlag, _ := cmd.Flags().GetString("project")

		yes, _ := cmd.Flags().GetBool("yes")
		var confirm ConfirmFunc
		if yes {
			confirm = AlwaysYes()
		} else {
			confirm = NewConfirmFunc()
		}

		return runConfigReset(cmd, homeDir, repoDir, projectFlag, confirm)
	},
}.Build()

func runConfigReset(cmd *cobra.Command, homeDir, repoDir, projectFlag string, confirm ConfirmFunc) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	confirmed, err := confirm(fmt.Sprintf("Reset schedule for '%s' to default (Mon-Fri 9am-5pm)?", entry.Name))
	if err != nil {
		return err
	}
	if !confirmed {
		return fmt.Errorf("aborted")
	}

	if err := project.ResetSchedules(homeDir, entry.ID); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("schedule for '%s' reset to default", Primary(entry.Name))))
	return nil
}
