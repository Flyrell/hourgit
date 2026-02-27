package cli

import (
	"fmt"
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
	proj      *project.ProjectEntry
	checkouts []entry.CheckoutEntry
	logs      []entry.Entry
	schedules []schedule.DaySchedule
	submits   []entry.SubmitEntry
	from      time.Time
	to        time.Time
	year      int
	month     time.Month
	weekNum   int // >0 when using --week view
}

var reportCmd = LeafCommand{
	Use:   "report",
	Short: "Generate a monthly time report",
	StrFlags: []StringFlag{
		{Name: "month", Usage: "month number 1-12 (default: current month)"},
		{Name: "week", Usage: "ISO week number 1-53 (default: current week)"},
		{Name: "year", Usage: "year (complementary to --month or --week)"},
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "export", Usage: "export format (pdf)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		monthFlag, _ := cmd.Flags().GetString("month")
		weekFlag, _ := cmd.Flags().GetString("week")
		yearFlag, _ := cmd.Flags().GetString("year")
		exportFlag, _ := cmd.Flags().GetString("export")

		monthChanged := cmd.Flags().Changed("month")
		weekChanged := cmd.Flags().Changed("week")
		yearChanged := cmd.Flags().Changed("year")

		return runReport(cmd, homeDir, repoDir, projectFlag, monthFlag, weekFlag, yearFlag, exportFlag, monthChanged, weekChanged, yearChanged, time.Now)
	},
}.Build()

func runReport(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, monthFlag, weekFlag, yearFlag, exportFlag string,
	monthChanged, weekChanged, yearChanged bool,
	nowFn func() time.Time,
) error {
	now := nowFn()

	inputs, err := loadReportInputs(homeDir, repoDir, projectFlag, monthFlag, weekFlag, yearFlag, monthChanged, weekChanged, yearChanged, now)
	if err != nil {
		return err
	}

	// PDF export path
	if exportFlag != "" {
		if exportFlag != "pdf" {
			return fmt.Errorf("unsupported export format %q (supported: pdf)", exportFlag)
		}

		exportData := timetrack.BuildExportData(
			inputs.checkouts, inputs.logs, inputs.schedules,
			inputs.year, inputs.month, now, nil,
			inputs.proj.Name,
		)

		if len(exportData.Days) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No time entries for %s %d.\n", inputs.month, inputs.year)
			return nil
		}

		var outputPath string
		if inputs.weekNum > 0 {
			outputPath = fmt.Sprintf("%s-%d-week-%02d.pdf", inputs.proj.Slug, inputs.year, inputs.weekNum)
		} else {
			outputPath = fmt.Sprintf("%s-%d-month-%02d.pdf", inputs.proj.Slug, inputs.year, inputs.month)
		}

		if err := renderExportPDF(exportData, outputPath); err != nil {
			return err
		}

		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Exported report to %s\n", outputPath)
		return nil
	}

	// Interactive table path — use detailed report
	data := timetrack.BuildDetailedReport(
		inputs.checkouts, inputs.logs, inputs.schedules,
		inputs.from, inputs.to, now,
	)

	if len(data.Rows) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No time entries for the selected period.\n")
		return nil
	}

	// Check if period was previously submitted
	submitted := isSubmitted(inputs.submits, inputs.from, inputs.to)

	return runReportTable(cmd, data, homeDir, inputs.proj.Slug, submitted)
}

// isSubmitted checks if any submit entry covers the given date range.
func isSubmitted(submits []entry.SubmitEntry, from, to time.Time) bool {
	for _, s := range submits {
		// A submit covers the range if it overlaps
		if !s.To.Before(from) && !s.From.After(to) {
			return true
		}
	}
	return false
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

// parseReportDateRange resolves the from/to date range from --month, --week, --year flags.
// Rules:
//   - --month + --week = error
//   - --year alone = error
//   - neither = default to current month
//   - --month (with optional --year) = full month
//   - --week (with optional --year) = ISO week Mon-Sun
func parseReportDateRange(monthFlag, weekFlag, yearFlag string, monthChanged, weekChanged, yearChanged bool, now time.Time) (from, to time.Time, year int, month time.Month, err error) {
	if monthChanged && weekChanged {
		return time.Time{}, time.Time{}, 0, 0, fmt.Errorf("--month and --week cannot be used together")
	}

	if yearChanged && !monthChanged && !weekChanged {
		return time.Time{}, time.Time{}, 0, 0, fmt.Errorf("--year must be used with --month or --week")
	}

	if weekChanged {
		return parseWeekRange(weekFlag, yearFlag, now)
	}

	// Default to month view (handles both --month and no flags)
	y, m, parseErr := parseMonthYearFlags(monthFlag, yearFlag, now)
	if parseErr != nil {
		return time.Time{}, time.Time{}, 0, 0, parseErr
	}

	daysInMonth := time.Date(y, m+1, 0, 0, 0, 0, 0, time.UTC).Day()
	from = time.Date(y, m, 1, 0, 0, 0, 0, time.UTC)
	to = time.Date(y, m, daysInMonth, 0, 0, 0, 0, time.UTC)
	return from, to, y, m, nil
}

// parseWeekRange parses --week and --year into a Mon-Sun date range.
func parseWeekRange(weekFlag, yearFlag string, now time.Time) (from, to time.Time, year int, month time.Month, err error) {
	y := now.Year()
	if yearFlag != "" {
		yy, parseErr := strconv.Atoi(yearFlag)
		if parseErr != nil || yy <= 0 {
			return time.Time{}, time.Time{}, 0, 0, fmt.Errorf("invalid --year value %q (expected a positive number)", yearFlag)
		}
		y = yy
	}

	var week int
	if weekFlag == "" {
		// No value provided, use current ISO week
		_, week = now.ISOWeek()
		if yearFlag == "" {
			y, _ = now.ISOWeek()
		}
	} else {
		w, parseErr := strconv.Atoi(weekFlag)
		if parseErr != nil || w < 1 || w > 53 {
			return time.Time{}, time.Time{}, 0, 0, fmt.Errorf("invalid --week value %q (expected 1-53)", weekFlag)
		}
		week = w
	}

	monday := isoWeekStart(y, week)
	sunday := monday.AddDate(0, 0, 6)

	// Use the month of the Monday as the primary month for display
	return monday, sunday, monday.Year(), monday.Month(), nil
}

// isoWeekStart returns the Monday of the given ISO year and week.
func isoWeekStart(year, week int) time.Time {
	// Jan 4 is always in week 1 of its ISO year
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.UTC)

	// Find the Monday of week 1
	jan4Weekday := jan4.Weekday()
	if jan4Weekday == time.Sunday {
		jan4Weekday = 7
	}
	week1Monday := jan4.AddDate(0, 0, -int(jan4Weekday-time.Monday))

	// Add (week - 1) * 7 days to get to the target week's Monday
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

// loadReportInputs resolves the project and loads all entries, schedules, and
// generated-day markers needed by both the interactive report and PDF export.
func loadReportInputs(homeDir, repoDir, projectFlag, monthFlag, weekFlag, yearFlag string, monthChanged, weekChanged, yearChanged bool, now time.Time) (*reportInputs, error) {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return nil, err
	}

	from, to, year, month, err := parseReportDateRange(monthFlag, weekFlag, yearFlag, monthChanged, weekChanged, yearChanged, now)
	if err != nil {
		return nil, err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	schedules := project.GetSchedules(cfg, proj.ID)

	// Expand schedules to cover the full date range (may span multiple months for week view)
	rangeStart := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(to.Year(), to.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	rangeEnd := time.Date(to.Year(), to.Month(), lastDay, 23, 59, 59, 0, time.UTC)

	daySchedules, err := schedule.ExpandSchedules(schedules, rangeStart, rangeEnd)
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

	submits, err := entry.ReadAllSubmitEntries(homeDir, proj.Slug)
	if err != nil {
		return nil, err
	}

	var weekNum int
	if weekChanged {
		// Derive week number from the resolved Monday date
		_, weekNum = from.ISOWeek()
	}

	return &reportInputs{
		proj:      proj,
		checkouts: checkouts,
		logs:      logs,
		schedules: daySchedules,
		submits:   submits,
		from:      from,
		to:        to,
		year:      year,
		month:     month,
		weekNum:   weekNum,
	}, nil
}
