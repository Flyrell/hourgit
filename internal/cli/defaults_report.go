package cli

import (
	"fmt"
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var defaultsReportCmd = LeafCommand{
	Use:   "report",
	Short: "Show expanded default working hours for a given month",
	StrFlags: []StringFlag{
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		monthFlag, _ := cmd.Flags().GetString("month")
		yearFlag, _ := cmd.Flags().GetString("year")

		return runDefaultsReport(cmd, homeDir, monthFlag, yearFlag, time.Now())
	},
}.Build()

func runDefaultsReport(cmd *cobra.Command, homeDir, monthFlag, yearFlag string, now time.Time) error {
	year, month, err := parseMonthYearFlags(monthFlag, yearFlag, now)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	defaults := project.GetDefaults(cfg)

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	days, err := schedule.ExpandSchedules(defaults, monthStart, monthEnd)
	if err != nil {
		return err
	}

	monthLabel := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("January 2006")
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
