package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var configReadCmd = LeafCommand{
	Use:   "read",
	Short: "Show expanded working hours for the current month",
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

		return runConfigRead(cmd, homeDir, repoDir, projectFlag, time.Now())
	},
}.Build()

func runConfigRead(cmd *cobra.Command, homeDir, repoDir, projectFlag string, now time.Time) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	entries := project.GetSchedules(cfg, entry.ID)

	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	days, err := schedule.ExpandSchedules(entries, monthStart, monthEnd)
	if err != nil {
		return err
	}

	monthLabel := now.Format("January 2006")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("Working hours for '%s' (%s):", Primary(entry.Name), monthLabel)))

	if len(days) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text("No working hours scheduled this month."))
		return nil
	}

	for _, ds := range days {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text(schedule.FormatDaySchedule(ds)))
	}

	return nil
}
