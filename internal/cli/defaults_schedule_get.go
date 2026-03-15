package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var defaultsScheduleGetCmd = LeafCommand{
	Use:   "get",
	Short: "Show the default schedule for new projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runDefaultsScheduleGet(cmd, homeDir)
	},
}.Build()

func runDefaultsScheduleGet(cmd *cobra.Command, homeDir string) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	defaults := project.GetDefaults(cfg)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text("Default schedule for new projects:"))
	printScheduleList(cmd, defaults)

	return nil
}
