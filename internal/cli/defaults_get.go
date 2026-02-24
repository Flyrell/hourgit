package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var defaultsGetCmd = LeafCommand{
	Use:   "get",
	Short: "Show the default schedule for new projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runDefaultsGet(cmd, homeDir)
	},
}.Build()

func runDefaultsGet(cmd *cobra.Command, homeDir string) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	defaults := project.GetDefaults(cfg)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text("Default schedule for new projects:"))

	for i, s := range defaults {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text(fmt.Sprintf("%d. %s", i+1, schedule.FormatScheduleEntry(s))))
	}

	return nil
}
