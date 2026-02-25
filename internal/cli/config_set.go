package cli

import (
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var configSetCmd = LeafCommand{
	Use:   "set",
	Short: "Interactively edit a project's schedule",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		kit := NewPromptKit()

		return runConfigSet(cmd, homeDir, repoDir, projectFlag, kit)
	},
}.Build()

func runConfigSet(cmd *cobra.Command, homeDir, repoDir, projectFlag string, kit PromptKit) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetSchedules(cfg, entry.ID)

	return runScheduleEditor(cmd, kit, schedules, entry.Name, func(s []schedule.ScheduleEntry) error {
		return project.SetSchedules(homeDir, entry.ID, s)
	})
}
