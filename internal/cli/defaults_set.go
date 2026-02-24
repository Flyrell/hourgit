package cli

import (
	"os"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var defaultsSetCmd = LeafCommand{
	Use:   "set",
	Short: "Interactively edit the default schedule for new projects",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		kit := NewPromptKit()
		return runDefaultsSet(cmd, homeDir, kit)
	},
}.Build()

func runDefaultsSet(cmd *cobra.Command, homeDir string, kit PromptKit) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetDefaults(cfg)

	return runScheduleEditor(cmd, kit, schedules, "defaults", func(s []schedule.ScheduleEntry) error {
		return project.SetDefaults(homeDir, s)
	})
}
