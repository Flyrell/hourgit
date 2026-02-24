package cli

import (
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/spf13/cobra"
)

var generateCmd = LeafCommand{
	Use:   "generate",
	Short: "Generate time entries from checkout history",
	BoolFlags: []BoolFlag{
		{Name: "today", Usage: "generate for today"},
		{Name: "week", Usage: "generate for current week (Mon-Sun)"},
		{Name: "month", Usage: "generate for the current month"},
		{Name: "yes", Usage: "skip confirmation prompts"},
	},
	StrFlags: []StringFlag{
		{Name: "date", Usage: "generate for a specific date (YYYY-MM-DD)"},
		{Name: "year", Usage: "year (with --month)"},
		{Name: "project", Shorthand: "p", Usage: "project name or slug"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()
		projectFlag, _ := cmd.Flags().GetString("project")
		dateFlag, _ := cmd.Flags().GetString("date")
		yearFlag, _ := cmd.Flags().GetString("year")
		todayFlag, _ := cmd.Flags().GetBool("today")
		weekFlag, _ := cmd.Flags().GetBool("week")
		monthFlag, _ := cmd.Flags().GetBool("month")
		yesFlag, _ := cmd.Flags().GetBool("yes")

		pk := NewPromptKit()
		if yesFlag {
			pk.Confirm = AlwaysYes()
		}
		return runGenerate(cmd, homeDir, repoDir, projectFlag, dateFlag, yearFlag, todayFlag, weekFlag, monthFlag, pk, time.Now)
	},
}.Build()

// generateEntry holds a preview of an entry to be created.
type generateEntry struct {
	Day     int
	Date    string // "2006-01-02"
	Branch  string
	Minutes int
}

func runGenerate(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, dateFlag, yearFlag string,
	todayFlag, weekFlag, monthFlag bool,
	pk PromptKit,
	nowFn func() time.Time,
) error {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	now := nowFn()

	// Determine date range
	from, to, err := resolveGenerateDateRange(dateFlag, yearFlag, todayFlag, weekFlag, monthFlag, pk, now)
	if err != nil {
		return err
	}

	// Load data (same as buildReportData)
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetSchedules(cfg, proj.ID)

	// Expand to cover the month(s) spanned by the date range
	rangeStart := time.Date(from.Year(), from.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(to.Year(), to.Month()+1, 0, 0, 0, 0, 0, time.UTC).Day()
	rangeEnd := time.Date(to.Year(), to.Month(), lastDay, 23, 59, 59, 0, time.UTC)

	daySchedules, err := schedule.ExpandSchedules(schedules, rangeStart, rangeEnd)
	if err != nil {
		return err
	}

	checkouts, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}

	// Check for existing generated-day markers in range
	existingGenerated, err := entry.ReadAllGeneratedDayEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}

	dateRange := buildDateRange(from, to)
	dateSet := make(map[string]bool, len(dateRange))
	for _, d := range dateRange {
		dateSet[d] = true
	}

	var overlapDates []string
	for _, s := range existingGenerated {
		if dateSet[s.Date] {
			overlapDates = append(overlapDates, s.Date)
		}
	}

	if len(overlapDates) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s %d day(s) in this range have already been generated.\n",
			Warning("Warning:"),
			len(overlapDates),
		)
		ok, err := pk.Confirm("Overwrite existing generated entries?")
		if err != nil {
			return err
		}
		if !ok {
			return nil
		}

		// Delete old generated entries and generated-day markers for overlap dates
		if err := deleteGeneratedEntries(homeDir, proj.Slug, overlapDates); err != nil {
			return err
		}
		if err := entry.DeleteGeneratedDayEntriesByDate(homeDir, proj.Slug, overlapDates); err != nil {
			return err
		}
	}

	// Compute checkout attribution for each month in range
	entries, err := buildGenerateEntries(checkouts, daySchedules, from, to, now)
	if err != nil {
		return err
	}

	if len(entries) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No checkout time to generate for the selected range.")
		return nil
	}

	// Preview
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w, "Entries to generate:")
	_, _ = fmt.Fprintln(w)
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "  %s  %s  %s\n",
			Text(e.Date),
			Primary(e.Branch),
			Info(entry.FormatMinutes(e.Minutes)),
		)
	}
	_, _ = fmt.Fprintln(w)

	// Confirm
	ok, err := pk.Confirm(fmt.Sprintf("Create %d entries?", len(entries)))
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// Create entries and generated-day markers
	created := 0
	generatedDays := make(map[string]bool)
	for _, ge := range entries {
		dt, _ := time.Parse("2006-01-02", ge.Date)
		e := entry.Entry{
			ID:        hashutil.GenerateID("generate"),
			Start:     time.Date(dt.Year(), dt.Month(), dt.Day(), 9, 0, 0, 0, time.UTC),
			Minutes:   ge.Minutes,
			Message:   ge.Branch,
			Task:      ge.Branch,
			Source:    "generate",
			CreatedAt: now.UTC(),
		}
		if err := entry.WriteEntry(homeDir, proj.Slug, e); err != nil {
			return err
		}
		created++

		if !generatedDays[ge.Date] {
			generatedDays[ge.Date] = true
			gde := entry.GeneratedDayEntry{
				ID:   hashutil.GenerateID("generated_day"),
				Date: ge.Date,
			}
			if err := entry.WriteGeneratedDayEntry(homeDir, proj.Slug, gde); err != nil {
				return err
			}
		}
	}

	_, _ = fmt.Fprintf(w, "Generated %s across %s for project '%s'.\n",
		Primary(fmt.Sprintf("%d entries", created)),
		Primary(fmt.Sprintf("%d days", len(generatedDays))),
		Primary(proj.Name),
	)

	return nil
}

// resolveGenerateDateRange determines the date range from flags or interactive prompt.
func resolveGenerateDateRange(
	dateFlag, yearFlag string,
	todayFlag, weekFlag, monthFlag bool,
	pk PromptKit,
	now time.Time,
) (time.Time, time.Time, error) {
	flagCount := 0
	if todayFlag {
		flagCount++
	}
	if weekFlag {
		flagCount++
	}
	if monthFlag {
		flagCount++
	}
	if dateFlag != "" {
		flagCount++
	}
	if flagCount > 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("only one of --today, --week, --month, or --date can be specified")
	}

	if todayFlag {
		f, t := dateRangeToday(now)
		return f, t, nil
	}
	if weekFlag {
		f, t := dateRangeWeek(now)
		return f, t, nil
	}
	if monthFlag {
		return dateRangeMonth(now, yearFlag)
	}
	if dateFlag != "" {
		return dateRangeSpecific(dateFlag)
	}

	// Interactive mode
	idx, err := pk.Select("Generate for which timeframe?", []string{
		"Today",
		"This week (Mon-Sun)",
		"Specific date",
		"This month",
	})
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	switch idx {
	case 0:
		f, t := dateRangeToday(now)
		return f, t, nil
	case 1:
		f, t := dateRangeWeek(now)
		return f, t, nil
	case 2:
		input, err := pk.Prompt("Date (YYYY-MM-DD)")
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		return dateRangeSpecific(input)
	case 3:
		return dateRangeMonth(now, "")
	}

	return time.Time{}, time.Time{}, fmt.Errorf("unexpected selection")
}

func dateRangeToday(now time.Time) (time.Time, time.Time) {
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	return d, d
}

func dateRangeWeek(now time.Time) (time.Time, time.Time) {
	d := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// Go back to Monday
	offset := int(d.Weekday()) - int(time.Monday)
	if offset < 0 {
		offset += 7
	}
	monday := d.AddDate(0, 0, -offset)
	sunday := monday.AddDate(0, 0, 6)
	return monday, sunday
}

func dateRangeMonth(now time.Time, yearFlag string) (time.Time, time.Time, error) {
	year := now.Year()
	if yearFlag != "" {
		y, m, err := parseMonthYearFlags("", yearFlag, now)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		year = y
		_ = m
	}
	month := now.Month()
	first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	last := time.Date(year, month, lastDay, 0, 0, 0, 0, time.UTC)
	return first, last, nil
}

func dateRangeSpecific(dateStr string) (time.Time, time.Time, error) {
	d, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid date format, expected YYYY-MM-DD: %w", err)
	}
	return d, d, nil
}

// buildDateRange returns all date strings between from and to (inclusive).
func buildDateRange(from, to time.Time) []string {
	var dates []string
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		dates = append(dates, d.Format("2006-01-02"))
	}
	return dates
}

// buildGenerateEntries computes checkout attribution for the date range and
// returns entries to be created.
func buildGenerateEntries(
	checkouts []entry.CheckoutEntry,
	daySchedules []schedule.DaySchedule,
	from, to time.Time,
	now time.Time,
) ([]generateEntry, error) {
	var result []generateEntry

	// Process each month that overlaps with the range
	type monthKey struct {
		year  int
		month time.Month
	}
	months := make(map[monthKey]bool)
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		months[monthKey{d.Year(), d.Month()}] = true
	}

	for mk := range months {
		bucket := timetrack.BuildCheckoutAttribution(checkouts, daySchedules, mk.year, mk.month, now)
		for branch, dayMap := range bucket {
			for day, mins := range dayMap {
				dt := time.Date(mk.year, mk.month, day, 0, 0, 0, 0, time.UTC)
				if dt.Before(from) || dt.After(to) {
					continue
				}
				if mins <= 0 {
					continue
				}
				result = append(result, generateEntry{
					Day:     day,
					Date:    dt.Format("2006-01-02"),
					Branch:  branch,
					Minutes: mins,
				})
			}
		}
	}

	// Sort by date then branch
	sort.Slice(result, func(i, j int) bool {
		if result[i].Date != result[j].Date {
			return result[i].Date < result[j].Date
		}
		return result[i].Branch < result[j].Branch
	})

	return result, nil
}

// deleteGeneratedEntries deletes log entries with source="generate" that fall
// on the given dates.
func deleteGeneratedEntries(homeDir, slug string, dates []string) error {
	dateSet := make(map[string]bool, len(dates))
	for _, d := range dates {
		dateSet[d] = true
	}

	entries, err := entry.ReadAllEntries(homeDir, slug)
	if err != nil {
		return err
	}

	for _, e := range entries {
		if e.Source != "generate" {
			continue
		}
		dateStr := e.Start.Format("2006-01-02")
		if dateSet[dateStr] {
			if err := entry.DeleteEntry(homeDir, slug, e.ID); err != nil {
				return err
			}
		}
	}
	return nil
}
