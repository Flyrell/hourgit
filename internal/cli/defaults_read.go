package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var defaultsReadCmd = LeafCommand{
	Use:   "read",
	Short: "Show expanded default working hours for the current month",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runDefaultsRead(cmd, homeDir, time.Now())
	},
}.Build()

func runDefaultsRead(cmd *cobra.Command, homeDir string, now time.Time) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	defaults := project.GetDefaults(cfg)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	days, err := schedule.ExpandSchedules(defaults, monthStart, monthEnd)
	if err != nil {
		return err
	}

	monthLabel := now.Format("January 2006")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("Default working hours (%s):", monthLabel)))

	if len(days) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text("No working hours scheduled this month."))
		return nil
	}

	for _, ds := range days {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text(schedule.FormatDaySchedule(ds)))
	}

	return nil
}
