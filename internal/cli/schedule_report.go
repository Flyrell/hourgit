package cli

import (
	"fmt"
	"time"

	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

// printScheduleReport expands the given schedule entries for a month and prints
// the day-by-day working hours. label is used in the header (e.g. project name
// or "Default working hours").
func printScheduleReport(cmd *cobra.Command, schedules []schedule.ScheduleEntry, label string, monthFlag string, yearFlag string, now time.Time) error {
	year, month, err := parseMonthYearFlags(monthFlag, yearFlag, now)
	if err != nil {
		return err
	}

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)

	days, err := schedule.ExpandSchedules(schedules, monthStart, monthEnd)
	if err != nil {
		return err
	}

	monthLabel := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC).Format("January 2006")
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("%s (%s):", label, monthLabel)))

	if len(days) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text("No working hours scheduled this month."))
		return nil
	}

	for _, ds := range days {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", Text(schedule.FormatDaySchedule(ds)))
	}

	return nil
}

// printScheduleList prints a numbered list of schedule entries.
func printScheduleList(cmd *cobra.Command, schedules []schedule.ScheduleEntry) {
	w := cmd.OutOrStdout()
	if len(schedules) == 0 {
		_, _ = fmt.Fprintf(w, "  %s\n", Silent("(no schedules)"))
		return
	}
	for i, s := range schedules {
		_, _ = fmt.Fprintf(w, "  %s\n", Text(fmt.Sprintf("%d. %s", i+1, schedule.FormatScheduleEntry(s))))
	}
}
