package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var configGetCmd = LeafCommand{
	Use:   "get",
	Short: "Show the schedule configuration for a project",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()

		projectFlag, _ := cmd.Flags().GetString("project")

		return runConfigGet(cmd, homeDir, repoDir, projectFlag)
	},
}.Build()

func runConfigGet(cmd *cobra.Command, homeDir, repoDir, projectFlag string) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetSchedules(cfg, entry.ID)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("Schedule for '%s':", Primary(entry.Name))))

	for i, s := range schedules {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text(fmt.Sprintf("%d. %s", i+1, schedule.FormatScheduleEntry(s))))
	}

	return nil
}
