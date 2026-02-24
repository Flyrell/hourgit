package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var configReportCmd = LeafCommand{
	Use:   "report",
	Short: "Show expanded working hours for a given month",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()

		projectFlag, _ := cmd.Flags().GetString("project")
		monthFlag, _ := cmd.Flags().GetString("month")
		yearFlag, _ := cmd.Flags().GetString("year")

		return runConfigReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, time.Now())
	},
}.Build()

func runConfigReport(cmd *cobra.Command, homeDir, repoDir, projectFlag, monthFlag, yearFlag string, now time.Time) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	year, month, err := parseMonthYearFlags(monthFlag, yearFlag, now)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	entries := project.GetSchedules(cfg, entry.ID)

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	days, err := schedule.ExpandSchedules(entries, monthStart, monthEnd)
	if err != nil {
		return err
	}

	monthLabel := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("January 2006")
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
