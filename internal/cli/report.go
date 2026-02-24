package cli

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/spf13/cobra"
)

var reportCmd = LeafCommand{
	Use:   "report",
	Short: "Generate a monthly time report",
	StrFlags: []StringFlag{
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
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

		return runReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, time.Now)
	},
}.Build()

func runReport(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, monthFlag, yearFlag string,
	nowFn func() time.Time,
) error {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	now := nowFn()
	year, month, err := parseMonthYearFlags(monthFlag, yearFlag, now)
	if err != nil {
		return err
	}

	data, err := buildReportData(homeDir, proj, year, month, now)
	if err != nil {
		return err
	}

	if len(data.Rows) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No time entries for %s %d.\n", month, year)
		return nil
	}

	return runReportTable(cmd, data)
}

// parseMonthYearFlags parses the --month and --year flags into year and month.
// Defaults to current month/year if empty.
func parseMonthYearFlags(monthFlag, yearFlag string, now time.Time) (int, time.Month, error) {
	year := now.Year()
	if yearFlag != "" {
		y, err := strconv.Atoi(yearFlag)
		if err != nil || y <= 0 {
			return 0, 0, fmt.Errorf("invalid --year value %q (expected a positive number)", yearFlag)
		}
		year = y
	}

	month := now.Month()
	if monthFlag != "" {
		m, err := strconv.Atoi(monthFlag)
		if err != nil || m < 1 || m > 12 {
			return 0, 0, fmt.Errorf("invalid --month value %q (expected 1-12)", monthFlag)
		}
		month = time.Month(m)
	}

	return year, month, nil
}

// buildReportData loads entries and schedules for the project and builds the report.
func buildReportData(homeDir string, proj *project.ProjectEntry, year int, month time.Month, now time.Time) (timetrack.ReportData, error) {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return timetrack.ReportData{}, err
	}

	schedules := project.GetSchedules(cfg, proj.ID)
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	monthEnd := time.Date(year, month, daysInMonth, 23, 59, 59, 0, time.UTC)

	daySchedules, err := schedule.ExpandSchedules(schedules, monthStart, monthEnd)
	if err != nil {
		return timetrack.ReportData{}, err
	}

	logs, err := entry.ReadAllEntries(homeDir, proj.Slug)
	if err != nil {
		return timetrack.ReportData{}, err
	}

	checkouts, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	if err != nil {
		return timetrack.ReportData{}, err
	}

	return timetrack.BuildReport(checkouts, logs, daySchedules, year, month, now), nil
}
