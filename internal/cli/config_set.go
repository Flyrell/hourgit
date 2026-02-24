package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
	"github.com/teambition/rrule-go"
)

var configSetCmd = LeafCommand{
	Use:   "set",
	Short: "Interactively edit a project's schedule",
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
		kit := NewPromptKit()

		return runConfigSet(cmd, homeDir, repoDir, projectFlag, kit)
	},
}.Build()

func runConfigSet(cmd *cobra.Command, homeDir, repoDir, projectFlag string, kit PromptKit) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetSchedules(cfg, entry.ID)

	return runScheduleEditor(cmd, kit, schedules, entry.Name, func(s []schedule.ScheduleEntry) error {
		return project.SetSchedules(homeDir, entry.ID, s)
	})
}

// runScheduleEditor runs the interactive schedule editor loop.
// label is used in output messages (e.g. project name or "defaults").
// save is called when the user chooses "Save & quit".
func runScheduleEditor(cmd *cobra.Command, kit PromptKit, schedules []schedule.ScheduleEntry, label string, save func([]schedule.ScheduleEntry) error) error {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "%s\n\n", Text(fmt.Sprintf("Editing schedule for '%s'", Primary(label))))

	for {
		printScheduleList(cmd, schedules)

		actionOptions := []string{"Add schedule", "Edit schedule", "Delete schedule", "Save & quit"}
		actionIdx, err := kit.Select("Action", actionOptions)
		if err != nil {
			return err
		}

		switch actionIdx {
		case 3: // Save & quit
			if err := save(schedules); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "%s\n", Text(fmt.Sprintf("schedule for '%s' saved", Primary(label))))
			return nil

		case 0: // Add
			newEntry, err := buildScheduleEntry(kit, w)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			if entriesOverlap(schedules, newEntry) {
				override, err := kit.Confirm(Text("This schedule overlaps with existing entries. Override them for matching days?"))
				if err != nil {
					return err
				}
				if override {
					newEntry.Override = true
				}
			}
			schedules = append(schedules, newEntry)

		case 1: // Edit
			if len(schedules) == 0 {
				_, _ = fmt.Fprintf(w, "%s\n", Error("no schedules to edit"))
				continue
			}
			idx, err := selectScheduleIndex(kit, schedules, "Edit which schedule?")
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			newEntry, err := buildScheduleEntry(kit, w)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			// Check overlap against all entries except the one being replaced
			others := make([]schedule.ScheduleEntry, 0, len(schedules)-1)
			others = append(others, schedules[:idx]...)
			others = append(others, schedules[idx+1:]...)
			if entriesOverlap(others, newEntry) {
				override, err := kit.Confirm(Text("This schedule overlaps with existing entries. Override them for matching days?"))
				if err != nil {
					return err
				}
				if override {
					newEntry.Override = true
				}
			}
			schedules[idx] = newEntry

		case 2: // Delete
			if len(schedules) == 0 {
				_, _ = fmt.Fprintf(w, "%s\n", Error("no schedules to delete"))
				continue
			}
			idx, err := selectScheduleIndex(kit, schedules, "Delete which schedule?")
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			schedules = append(schedules[:idx], schedules[idx+1:]...)
		}
	}
}

// selectScheduleIndex prompts the user to pick a schedule from the list.
func selectScheduleIndex(kit PromptKit, schedules []schedule.ScheduleEntry, title string) (int, error) {
	options := make([]string, len(schedules))
	for i, s := range schedules {
		options[i] = fmt.Sprintf("%d. %s", i+1, schedule.FormatScheduleEntry(s))
	}
	return kit.Select(title, options)
}

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

// buildScheduleEntry guides the user through a step-by-step schedule builder.
func buildScheduleEntry(kit PromptKit, w io.Writer) (schedule.ScheduleEntry, error) {
	schedType, err := promptScheduleType(kit)
	if err != nil {
		return schedule.ScheduleEntry{}, err
	}

	var rruleStr string
	switch schedType {
	case "recurring":
		rruleStr, err = promptRecurrence(kit, w)
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
	case "oneoff":
		d, err := promptDate(kit.Prompt, w, "Date")
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:    rrule.DAILY,
			Count:   1,
			Dtstart: d,
		})
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		rruleStr = r.String()
	case "range":
		startDate, err := promptDate(kit.Prompt, w, "Start date")
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		endDate, err := promptDate(kit.Prompt, w, "End date")
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		if !startDate.Before(endDate) {
			return schedule.ScheduleEntry{}, fmt.Errorf("start date must be before end date")
		}
		until := time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 0, endDate.Location())
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:    rrule.DAILY,
			Dtstart: startDate,
			Until:   until,
		})
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		rruleStr = r.String()
	}

	ranges, err := promptTimeRanges(kit, w)
	if err != nil {
		return schedule.ScheduleEntry{}, err
	}

	entry := schedule.ScheduleEntry{
		Ranges: ranges,
		RRule:  rruleStr,
	}

	_, _ = fmt.Fprintf(w, "\n  %s\n", Text("→ "+schedule.FormatScheduleEntry(entry)))

	return entry, nil
}

// promptScheduleType asks the user to pick recurring, one-off, or date range.
func promptScheduleType(kit PromptKit) (string, error) {
	options := []string{"Recurring", "One-off date", "Date range"}
	idx, err := kit.Select("Schedule type", options)
	if err != nil {
		return "", err
	}
	switch idx {
	case 0:
		return "recurring", nil
	case 1:
		return "oneoff", nil
	case 2:
		return "range", nil
	}
	return "", fmt.Errorf("invalid selection")
}

// promptRecurrence builds an RRULE string from user choices.
func promptRecurrence(kit PromptKit, w io.Writer) (string, error) {
	options := []string{
		"Every weekday (Mon-Fri)",
		"Every weekend (Sat-Sun)",
		"Every day",
		"Specific days",
		"Every N days",
		"Every N weeks",
	}
	idx, err := kit.Select("Recurrence", options)
	if err != nil {
		return "", err
	}

	switch idx {
	case 0:
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:      rrule.WEEKLY,
			Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
		})
		if err != nil {
			return "", err
		}
		return r.String(), nil

	case 1:
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:      rrule.WEEKLY,
			Byweekday: []rrule.Weekday{rrule.SA, rrule.SU},
		})
		if err != nil {
			return "", err
		}
		return r.String(), nil

	case 2:
		r, err := rrule.NewRRule(rrule.ROption{Freq: rrule.DAILY})
		if err != nil {
			return "", err
		}
		return r.String(), nil

	case 3:
		days, err := promptDays(kit)
		if err != nil {
			return "", err
		}
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:      rrule.WEEKLY,
			Byweekday: days,
		})
		if err != nil {
			return "", err
		}
		return r.String(), nil

	case 4:
		n, err := promptInterval(kit.Prompt, w, "days")
		if err != nil {
			return "", err
		}
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:     rrule.DAILY,
			Interval: n,
		})
		if err != nil {
			return "", err
		}
		return r.String(), nil

	case 5:
		n, err := promptInterval(kit.Prompt, w, "weeks")
		if err != nil {
			return "", err
		}
		r, err := rrule.NewRRule(rrule.ROption{
			Freq:     rrule.WEEKLY,
			Interval: n,
		})
		if err != nil {
			return "", err
		}
		return r.String(), nil
	}

	return "", fmt.Errorf("invalid selection")
}

var weekdayList = []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA, rrule.SU}

// promptDays asks the user to select specific days of the week using multi-select.
func promptDays(kit PromptKit) ([]rrule.Weekday, error) {
	dayNames := []string{"Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday"}
	indices, err := kit.MultiSelect("Select days", dayNames)
	if err != nil {
		return nil, err
	}
	if len(indices) == 0 {
		return nil, fmt.Errorf("at least one day must be selected")
	}
	days := make([]rrule.Weekday, len(indices))
	for i, idx := range indices {
		days[i] = weekdayList[idx]
	}
	return days, nil
}

// promptInterval asks the user for an interval number.
func promptInterval(prompt PromptFunc, w io.Writer, unit string) (int, error) {
	for {
		input, err := prompt(fmt.Sprintf("Every how many %s?", unit))
		if err != nil {
			return 0, err
		}
		n, err := strconv.Atoi(strings.TrimSpace(input))
		if err != nil || n < 1 {
			_, _ = fmt.Fprintf(w, "%s\n", Error("please enter a positive number"))
			continue
		}
		return n, nil
	}
}

// promptDate asks the user for a date.
func promptDate(prompt PromptFunc, w io.Writer, label string) (time.Time, error) {
	for {
		input, err := prompt(fmt.Sprintf("%s (e.g. 2026-03-02, tomorrow)", label))
		if err != nil {
			return time.Time{}, err
		}
		input = strings.TrimSpace(input)
		if input == "" {
			_, _ = fmt.Fprintf(w, "%s\n", Error("please enter a date"))
			continue
		}
		d, err := schedule.ParseDate(input)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", Error("invalid date: "+err.Error()))
			continue
		}
		return *d, nil
	}
}

// promptTimeRanges asks the user for one or more start/end time pairs.
// After each range, the user is asked whether to add another.
func promptTimeRanges(kit PromptKit, w io.Writer) ([]schedule.TimeRange, error) {
	var ranges []schedule.TimeRange

	for {
		from, to, err := promptSingleTimeRange(kit.Prompt, w)
		if err != nil {
			return nil, err
		}
		ranges = append(ranges, schedule.TimeRange{From: from, To: to})

		// Validate collected ranges so far
		if err := schedule.ValidateRanges(ranges); err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", Error(err.Error()))
			// Remove the last range that caused the error
			ranges = ranges[:len(ranges)-1]
			continue
		}

		addMore, err := kit.Confirm("Add another time range?")
		if err != nil {
			return nil, err
		}
		if !addMore {
			break
		}
	}

	return ranges, nil
}

// promptSingleTimeRange asks the user for a start and end time, validating order.
func promptSingleTimeRange(prompt PromptFunc, w io.Writer) (string, string, error) {
	for {
		fromInput, err := prompt("Start time (e.g. 9am, 9:00, 14:30)")
		if err != nil {
			return "", "", err
		}
		fromTod, err := schedule.ParseTimeOfDay(fromInput)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", Error("invalid time: "+err.Error()))
			continue
		}

		toInput, err := prompt("End time (e.g. 5pm, 17:00, 14:30)")
		if err != nil {
			return "", "", err
		}
		toTod, err := schedule.ParseTimeOfDay(toInput)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", Error("invalid time: "+err.Error()))
			continue
		}

		if !fromTod.Before(toTod) {
			_, _ = fmt.Fprintf(w, "%s\n", Error("end time must be after start time"))
			continue
		}

		return fromTod.String(), toTod.String(), nil
	}
}

// entriesOverlap checks whether candidate shares any days with existing entries
// by expanding both over a 90-day window from today.
func entriesOverlap(existing []schedule.ScheduleEntry, candidate schedule.ScheduleEntry) bool {
	now := time.Now()
	from := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 90)

	existingDays, err := schedule.ExpandSchedules(existing, from, to)
	if err != nil {
		return false
	}

	candidateDays, err := schedule.ExpandSchedules([]schedule.ScheduleEntry{candidate}, from, to)
	if err != nil {
		return false
	}

	existingSet := make(map[string]bool, len(existingDays))
	for _, ds := range existingDays {
		existingSet[ds.Date.Format("2006-01-02")] = true
	}

	for _, ds := range candidateDays {
		if existingSet[ds.Date.Format("2006-01-02")] {
			return true
		}
	}
	return false
}
