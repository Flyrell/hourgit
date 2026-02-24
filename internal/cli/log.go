package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
)

var logCmd = LeafCommand{
	Use:   "log [message]",
	Short: "Log time manually for a project",
	Args:  cobra.MaximumNArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompts"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "duration", Usage: "duration to log (e.g. 30m, 3h, 3h30m)"},
		{Name: "from", Usage: "start time (e.g. 9am, 14:00)"},
		{Name: "to", Usage: "end time (e.g. 5pm, 17:00)"},
		{Name: "date", Usage: "date to log for (YYYY-MM-DD, default: today)"},
		{Name: "task", Usage: "task label for this entry"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()
		projectFlag, _ := cmd.Flags().GetString("project")
		durationFlag, _ := cmd.Flags().GetString("duration")
		fromFlag, _ := cmd.Flags().GetString("from")
		toFlag, _ := cmd.Flags().GetString("to")
		dateFlag, _ := cmd.Flags().GetString("date")
		taskFlag, _ := cmd.Flags().GetString("task")
		yesFlag, _ := cmd.Flags().GetBool("yes")

		var message string
		if len(args) > 0 {
			message = args[0]
		}

		pk := NewPromptKit()
		if yesFlag {
			pk.Confirm = AlwaysYes()
		}
		return runLog(cmd, homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message, pk, time.Now)
	},
}.Build()

func runLog(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, taskFlag, message string,
	pk PromptKit,
	nowFn func() time.Time,
) error {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	hasDuration := durationFlag != ""
	hasFrom := fromFlag != ""
	hasTo := toFlag != ""

	// 1. Validate mutual exclusivity
	if hasDuration && (hasFrom || hasTo) {
		return fmt.Errorf("--duration and --from/--to are mutually exclusive")
	}

	now := nowFn()

	// 2. Resolve date: use flag if provided, otherwise prompt
	if dateFlag == "" && !hasDuration && !hasFrom && !hasTo && message == "" {
		dateFlag, err = pk.Prompt("Date (YYYY-MM-DD, default: today)")
		if err != nil {
			return err
		}
	}

	baseDate, err := resolveBaseDate(dateFlag, now)
	if err != nil {
		return err
	}

	// 3. Resolve time mode
	if !hasDuration && !hasFrom && !hasTo {
		modeIdx, err := pk.Select("How do you want to log time?", []string{"Duration (e.g. 3h30m)", "Time range (e.g. 9am to 5pm)"})
		if err != nil {
			return err
		}
		if modeIdx == 0 {
			hasDuration = true
		}
	}

	var minutes int
	var start time.Time

	if hasDuration {
		if durationFlag == "" {
			durationFlag, err = pk.Prompt("Duration (e.g. 30m, 3h, 3h30m)")
			if err != nil {
				return err
			}
		}
		minutes, err = entry.ParseDuration(durationFlag)
		if err != nil {
			return err
		}
		y, m, d := baseDate.Date()
		start = time.Date(y, m, d, now.Hour(), now.Minute(), 0, 0, now.Location()).
			Add(-time.Duration(minutes) * time.Minute)
	} else {
		if fromFlag == "" {
			fromFlag, err = pk.Prompt("From (e.g. 9am, 14:00)")
			if err != nil {
				return err
			}
		}
		if toFlag == "" {
			toFlag, err = pk.Prompt("To (e.g. 5pm, 17:00)")
			if err != nil {
				return err
			}
		}
		start, minutes, err = parseFromTo(fromFlag, toFlag, baseDate)
		if err != nil {
			return err
		}
	}

	// 4. Validate 24h cap
	if minutes > 24*60 {
		return fmt.Errorf("cannot log more than 24h in a single entry")
	}

	// 5. Check schedule warnings
	proceed, err := checkScheduleWarnings(cmd, homeDir, proj, start, minutes, "", pk.Confirm)
	if err != nil {
		return err
	}
	if !proceed {
		return nil
	}

	// 6. Resolve message
	if message == "" {
		message, err = pk.Prompt("Message")
		if err != nil {
			return err
		}
	}
	if message == "" {
		return fmt.Errorf("message is required")
	}

	return writeAndPrintEntry(cmd, homeDir, proj, start, minutes, message, taskFlag, now)
}

// resolveBaseDate parses the --date flag value into a date.
// If dateFlag is empty, returns now (today).
func resolveBaseDate(dateFlag string, now time.Time) (time.Time, error) {
	if dateFlag == "" {
		return now, nil
	}
	d, err := time.Parse("2006-01-02", dateFlag)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date format, expected YYYY-MM-DD: %w", err)
	}
	return d, nil
}

func parseFromTo(fromStr, toStr string, baseDate time.Time) (time.Time, int, error) {
	fromTOD, err := schedule.ParseTimeOfDay(fromStr)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid --from time: %w", err)
	}
	toTOD, err := schedule.ParseTimeOfDay(toStr)
	if err != nil {
		return time.Time{}, 0, fmt.Errorf("invalid --to time: %w", err)
	}

	if !fromTOD.Before(toTOD) {
		return time.Time{}, 0, fmt.Errorf("--from (%s) must be before --to (%s)", fromTOD, toTOD)
	}

	y, m, d := baseDate.Date()
	loc := baseDate.Location()
	start := time.Date(y, m, d, fromTOD.Hour, fromTOD.Minute, 0, 0, loc)
	end := time.Date(y, m, d, toTOD.Hour, toTOD.Minute, 0, 0, loc)
	minutes := int(end.Sub(start).Minutes())

	return start, minutes, nil
}

// checkScheduleWarnings warns the user about schedule-related issues:
// 1. No scheduled hours for the day
// 2. Entry falls outside (or partially outside) schedule windows
// 3. Entry exceeds remaining scheduled budget
// Returns (true, nil) to proceed, (false, nil) to cancel.
func checkScheduleWarnings(
	cmd *cobra.Command,
	homeDir string,
	proj *project.ProjectEntry,
	entryStart time.Time,
	minutes int,
	excludeID string,
	confirm ConfirmFunc,
) (bool, error) {
	if confirm == nil {
		return true, nil
	}

	windows, scheduledMinutes, err := getDayScheduleWindows(homeDir, proj, entryStart)
	if err != nil {
		return false, err
	}

	// 1. No scheduled hours for this day
	if scheduledMinutes == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s this day has no scheduled working hours.\n",
			Warning("Warning:"),
		)
		ok, err := confirm("Continue anyway?")
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	// 2. Check if entry falls outside schedule windows
	proceed, err := checkBoundsWarning(cmd, confirm, windows, entryStart, minutes)
	if err != nil {
		return false, err
	}
	if !proceed {
		return false, nil
	}

	// 3. Check budget overrun
	return checkBudgetWarning(cmd, confirm, homeDir, proj, entryStart, minutes, scheduledMinutes, excludeID)
}

// getDayScheduleWindows returns the schedule windows and total scheduled minutes
// for the day containing entryStart.
func getDayScheduleWindows(homeDir string, proj *project.ProjectEntry, entryStart time.Time) ([]schedule.TimeWindow, int, error) {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, 0, err
	}

	schedules := project.GetSchedules(cfg, proj.ID)

	y, m, d := entryStart.Date()
	dayStart := time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
	dayEnd := time.Date(y, m, d, 23, 59, 59, 0, time.UTC)

	daySchedules, err := schedule.ExpandSchedules(schedules, dayStart, dayEnd)
	if err != nil {
		return nil, 0, err
	}

	var dayWindows []schedule.TimeWindow
	dateKey := dayStart.Format("2006-01-02")
	for _, ds := range daySchedules {
		if ds.Date.Format("2006-01-02") == dateKey {
			dayWindows = ds.Windows
			break
		}
	}

	scheduledMinutes := 0
	for _, w := range dayWindows {
		fromMins := w.From.Hour*60 + w.From.Minute
		toMins := w.To.Hour*60 + w.To.Minute
		scheduledMinutes += toMins - fromMins
	}

	return dayWindows, scheduledMinutes, nil
}

// checkBoundsWarning warns if the entry falls fully or partially outside
// schedule windows. Returns (true, nil) to proceed, (false, nil) to cancel.
func checkBoundsWarning(
	cmd *cobra.Command,
	confirm ConfirmFunc,
	windows []schedule.TimeWindow,
	entryStart time.Time,
	minutes int,
) (bool, error) {
	entryFromMins := entryStart.Hour()*60 + entryStart.Minute()
	entryToMins := entryFromMins + minutes

	overlapMins := 0
	for _, w := range windows {
		wFrom := w.From.Hour*60 + w.From.Minute
		wTo := w.To.Hour*60 + w.To.Minute
		overlap := min(entryToMins, wTo) - max(entryFromMins, wFrom)
		if overlap > 0 {
			overlapMins += overlap
		}
	}

	windowsSummary := formatWindowsSummary(windows)

	if overlapMins == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s this entry falls outside your scheduled hours (%s).\n",
			Warning("Warning:"),
			Primary(windowsSummary),
		)
	} else if overlapMins < minutes {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s this entry partially falls outside your scheduled hours (%s).\n",
			Warning("Warning:"),
			Primary(windowsSummary),
		)
	} else {
		return true, nil
	}

	ok, err := confirm("Continue anyway?")
	if err != nil {
		return false, err
	}
	return ok, nil
}

// checkBudgetWarning warns if the entry exceeds the remaining scheduled budget
// for the day. Returns (true, nil) to proceed, (false, nil) to cancel.
func checkBudgetWarning(
	cmd *cobra.Command,
	confirm ConfirmFunc,
	homeDir string,
	proj *project.ProjectEntry,
	entryStart time.Time,
	minutes, scheduledMinutes int,
	excludeID string,
) (bool, error) {
	entries, err := entry.ReadAllEntries(homeDir, proj.Slug)
	if err != nil {
		return false, err
	}

	y, m, d := entryStart.Date()
	loggedMinutes := 0
	for _, e := range entries {
		if excludeID != "" && e.ID == excludeID {
			continue
		}
		ey, em, ed := e.Start.Date()
		if ey == y && em == m && ed == d {
			loggedMinutes += e.Minutes
		}
	}

	remaining := scheduledMinutes - loggedMinutes
	if minutes > remaining {
		if remaining <= 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s you have already logged your full schedule for this day (%s scheduled, %s logged).\n",
				Warning("Warning:"),
				Primary(entry.FormatMinutes(scheduledMinutes)),
				Primary(entry.FormatMinutes(loggedMinutes)),
			)
		} else {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s you are about to log %s, but only %s remains in today's schedule (%s scheduled, %s already logged).\n",
				Warning("Warning:"),
				Primary(entry.FormatMinutes(minutes)),
				Primary(entry.FormatMinutes(remaining)),
				Primary(entry.FormatMinutes(scheduledMinutes)),
				Primary(entry.FormatMinutes(loggedMinutes)),
			)
		}

		ok, err := confirm("Continue anyway?")
		if err != nil {
			return false, err
		}
		return ok, nil
	}

	return true, nil
}

// formatWindowsSummary formats schedule windows as a comma-separated summary.
func formatWindowsSummary(windows []schedule.TimeWindow) string {
	parts := make([]string, len(windows))
	for i, w := range windows {
		parts[i] = schedule.FormatTimeRange(w.From.String(), w.To.String())
	}
	return strings.Join(parts, ", ")
}

func writeAndPrintEntry(
	cmd *cobra.Command,
	homeDir string,
	proj *project.ProjectEntry,
	start time.Time,
	minutes int,
	message, task string,
	now time.Time,
) error {
	e := entry.Entry{
		ID:        hashutil.GenerateID("log"),
		Start:     start.UTC(),
		Minutes:   minutes,
		Message:   message,
		Task:      task,
		CreatedAt: now.UTC(),
	}

	if err := entry.WriteEntry(homeDir, proj.Slug, e); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "logged %s for project '%s' (%s)\n",
		Primary(entry.FormatMinutes(e.Minutes)),
		Primary(proj.Name),
		Silent(e.ID),
	)

	return nil
}
