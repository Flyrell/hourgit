package cli

import (
	"fmt"
	"os"
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
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "duration", Usage: "duration to log (e.g. 30m, 3h, 1d3h30m)"},
		{Name: "from", Usage: "start time (e.g. 9am, 14:00)"},
		{Name: "to", Usage: "end time (e.g. 5pm, 17:00)"},
		{Name: "date", Usage: "date to log for (YYYY-MM-DD, default: today)"},
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

		var message string
		if len(args) > 0 {
			message = args[0]
		}

		pk := NewPromptKit()
		return runLog(cmd, homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, message, pk, time.Now)
	},
}.Build()

func runLog(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, durationFlag, fromFlag, toFlag, dateFlag, message string,
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
			durationFlag, err = pk.Prompt("Duration (e.g. 30m, 3h, 1d3h30m)")
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

	// 4. Resolve message
	if message == "" {
		message, err = pk.Prompt("Message")
		if err != nil {
			return err
		}
	}
	if message == "" {
		return fmt.Errorf("message is required")
	}

	return writeAndPrintEntry(cmd, homeDir, proj, start, minutes, message, now)
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

func writeAndPrintEntry(
	cmd *cobra.Command,
	homeDir string,
	proj *project.ProjectEntry,
	start time.Time,
	minutes int,
	message string,
	now time.Time,
) error {
	e := entry.Entry{
		ID:        hashutil.GenerateID("log"),
		Start:     start.UTC(),
		Minutes:   minutes,
		Message:   message,
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
