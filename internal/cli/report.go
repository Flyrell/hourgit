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

// reportInputs holds the raw data loaded from storage, shared between
// the interactive report and the PDF export paths.
type reportInputs struct {
	proj       *project.ProjectEntry
	checkouts  []entry.CheckoutEntry
	logs       []entry.Entry
	schedules  []schedule.DaySchedule
	genDays    []string
	year       int
	month      time.Month
}

var reportCmd = LeafCommand{
	Use:   "report",
	Short: "Generate a monthly time report",
	StrFlags: []StringFlag{
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "output", Usage: "export report as PDF to the given path (auto-named if empty)"},
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
		outputFlag, _ := cmd.Flags().GetString("output")

		return runReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, outputFlag, time.Now)
	},
}.Build()

func runReport(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, monthFlag, yearFlag, outputFlag string,
	nowFn func() time.Time,
) error {
	now := nowFn()

	inputs, err := loadReportInputs(homeDir, repoDir, projectFlag, monthFlag, yearFlag, now)
	if err != nil {
		return err
	}

	// PDF export path
	if cmd.Flags().Changed("output") {
		exportData := timetrack.BuildExportData(
			inputs.checkouts, inputs.logs, inputs.schedules,
			inputs.year, inputs.month, now, inputs.genDays,
			inputs.proj.Name,
		)

		if len(exportData.Days) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No time entries for %s %d.\n", inputs.month, inputs.year)
			return nil
		}

		outputPath := outputFlag
		if outputPath == "" {
			outputPath = fmt.Sprintf("%s-%d-%02d.pdf", inputs.proj.Slug, inputs.year, inputs.month)
		}

		if err := renderExportPDF(exportData, outputPath); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported report to %s\n", outputPath)
		return nil
	}

	// Interactive table path
	data := timetrack.BuildReport(
		inputs.checkouts, inputs.logs, inputs.schedules,
		inputs.year, inputs.month, now, inputs.genDays,
	)

	if len(data.Rows) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No time entries for %s %d.\n", inputs.month, inputs.year)
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

// loadReportInputs resolves the project and loads all entries, schedules, and
// generated-day markers needed by both the interactive report and PDF export.
func loadReportInputs(homeDir, repoDir, projectFlag, monthFlag, yearFlag string, now time.Time) (*reportInputs, error) {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return nil, err
	}

	year, month, err := parseMonthYearFlags(monthFlag, yearFlag, now)
	if err != nil {
		return nil, err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	schedules := project.GetSchedules(cfg, proj.ID)
	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	daysInMonth := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	monthEnd := time.Date(year, month, daysInMonth, 23, 59, 59, 0, time.UTC)

	daySchedules, err := schedule.ExpandSchedules(schedules, monthStart, monthEnd)
	if err != nil {
		return nil, err
	}

	logs, err := entry.ReadAllEntries(homeDir, proj.Slug)
	if err != nil {
		return nil, err
	}

	checkouts, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	if err != nil {
		return nil, err
	}

	generatedDayEntries, err := entry.ReadAllGeneratedDayEntries(homeDir, proj.Slug)
	if err != nil {
		return nil, err
	}

	var genDays []string
	for _, g := range generatedDayEntries {
		genDays = append(genDays, g.Date)
	}

	return &reportInputs{
		proj:      proj,
		checkouts: checkouts,
		logs:      logs,
		schedules: daySchedules,
		genDays:   genDays,
		year:      year,
		month:     month,
	}, nil
}
