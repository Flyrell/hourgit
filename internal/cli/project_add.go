package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var projectAddCmd = LeafCommand{
	Use:   "add PROJECT",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runProjectAdd(cmd, homeDir, args[0])
	},
}.Build()

func runProjectAdd(cmd *cobra.Command, homeDir, name string) error {
	entry, err := project.CreateProject(homeDir, name)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
	return nil
}
