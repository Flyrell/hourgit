package cli

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Flyrell/hour-git/internal/schedule"
	"github.com/Flyrell/hour-git/internal/project"
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
		prompt := NewPromptFunc(cmd.InOrStdin(), cmd.OutOrStdout())
		confirm := NewConfirmFunc(cmd.InOrStdin(), cmd.OutOrStdout())

		return runConfigSet(cmd, homeDir, repoDir, projectFlag, prompt, confirm)
	},
}.Build()

func runConfigSet(cmd *cobra.Command, homeDir, repoDir, projectFlag string, prompt PromptFunc, confirm ConfirmFunc) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	reg, err := project.ReadRegistry(homeDir)
	if err != nil {
		return err
	}

	schedules := project.GetSchedules(reg, entry.ID)

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "%s\n\n", Text(fmt.Sprintf("Editing schedule for '%s'", Primary(entry.Name))))

	for {
		printScheduleList(cmd, schedules)
		_, _ = fmt.Fprintf(w, "\n%s ", Text("[a]dd  [e]dit N  [d]elete N  [q]uit"))

		action, err := prompt("")
		if err != nil {
			return err
		}

		action = strings.TrimSpace(strings.ToLower(action))

		switch {
		case action == "q" || action == "quit":
			if err := project.SetSchedules(homeDir, entry.ID, schedules); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "%s\n", Text(fmt.Sprintf("schedule for '%s' saved", Primary(entry.Name))))
			return nil

		case action == "a" || action == "add":
			newEntry, err := buildScheduleEntry(prompt, w)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			if entriesOverlap(schedules, newEntry) {
				override, err := confirm(Text("This schedule overlaps with existing entries. Override them for matching days?"))
				if err != nil {
					return err
				}
				if override {
					newEntry.Override = true
				}
			}
			schedules = append(schedules, newEntry)

		case strings.HasPrefix(action, "e ") || strings.HasPrefix(action, "edit "):
			idx, err := parseActionIndex(action, len(schedules))
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			newEntry, err := buildScheduleEntry(prompt, w)
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			// Check overlap against all entries except the one being replaced
			others := make([]schedule.ScheduleEntry, 0, len(schedules)-1)
			others = append(others, schedules[:idx]...)
			others = append(others, schedules[idx+1:]...)
			if entriesOverlap(others, newEntry) {
				override, err := confirm(Text("This schedule overlaps with existing entries. Override them for matching days?"))
				if err != nil {
					return err
				}
				if override {
					newEntry.Override = true
				}
			}
			schedules[idx] = newEntry

		case strings.HasPrefix(action, "d ") || strings.HasPrefix(action, "delete "):
			idx, err := parseActionIndex(action, len(schedules))
			if err != nil {
				_, _ = fmt.Fprintf(w, "%s\n", Error("error: "+err.Error()))
				continue
			}
			schedules = append(schedules[:idx], schedules[idx+1:]...)

		default:
			_, _ = fmt.Fprintf(w, "%s\n", Error("unknown action, use [a]dd, [e]dit N, [d]elete N, or [q]uit"))
		}
	}
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
func buildScheduleEntry(prompt PromptFunc, w io.Writer) (schedule.ScheduleEntry, error) {
	schedType, err := promptScheduleType(prompt, w)
	if err != nil {
		return schedule.ScheduleEntry{}, err
	}

	var rruleStr string
	switch schedType {
	case "recurring":
		rruleStr, err = promptRecurrence(prompt, w)
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
	case "oneoff":
		d, err := promptDate(prompt, w, "Date")
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
		startDate, err := promptDate(prompt, w, "Start date")
		if err != nil {
			return schedule.ScheduleEntry{}, err
		}
		endDate, err := promptDate(prompt, w, "End date")
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

	from, to, err := promptTimeRange(prompt, w)
	if err != nil {
		return schedule.ScheduleEntry{}, err
	}

	entry := schedule.ScheduleEntry{
		From:  from,
		To:    to,
		RRule: rruleStr,
	}

	_, _ = fmt.Fprintf(w, "\n  %s\n", Text("→ "+schedule.FormatScheduleEntry(entry)))

	return entry, nil
}

// promptScheduleType asks the user to pick recurring, one-off, or date range.
func promptScheduleType(prompt PromptFunc, w io.Writer) (string, error) {
	_, _ = fmt.Fprintf(w, "\n%s\n", Text("Schedule type:"))
	_, _ = fmt.Fprintf(w, "  %s\n", Text("[1] Recurring  [2] One-off date  [3] Date range"))

	for {
		input, err := prompt(Text("> "))
		if err != nil {
			return "", err
		}
		switch strings.TrimSpace(input) {
		case "1":
			return "recurring", nil
		case "2":
			return "oneoff", nil
		case "3":
			return "range", nil
		default:
			_, _ = fmt.Fprintf(w, "%s\n", Error("please enter 1, 2, or 3"))
		}
	}
}

// promptRecurrence builds an RRULE string from user choices.
func promptRecurrence(prompt PromptFunc, w io.Writer) (string, error) {
	_, _ = fmt.Fprintf(w, "\n%s\n", Text("Recurrence:"))
	_, _ = fmt.Fprintf(w, "  %s\n", Text("[1] Every weekday (Mon-Fri)  [2] Every weekend (Sat-Sun)"))
	_, _ = fmt.Fprintf(w, "  %s\n", Text("[3] Every day                [4] Specific days"))
	_, _ = fmt.Fprintf(w, "  %s\n", Text("[5] Every N days             [6] Every N weeks"))

	for {
		input, err := prompt(Text("> "))
		if err != nil {
			return "", err
		}

		switch strings.TrimSpace(input) {
		case "1":
			r, err := rrule.NewRRule(rrule.ROption{
				Freq:      rrule.WEEKLY,
				Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
			})
			if err != nil {
				return "", err
			}
			return r.String(), nil

		case "2":
			r, err := rrule.NewRRule(rrule.ROption{
				Freq:      rrule.WEEKLY,
				Byweekday: []rrule.Weekday{rrule.SA, rrule.SU},
			})
			if err != nil {
				return "", err
			}
			return r.String(), nil

		case "3":
			r, err := rrule.NewRRule(rrule.ROption{Freq: rrule.DAILY})
			if err != nil {
				return "", err
			}
			return r.String(), nil

		case "4":
			days, err := promptDays(prompt, w)
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

		case "5":
			n, err := promptInterval(prompt, w, "days")
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

		case "6":
			n, err := promptInterval(prompt, w, "weeks")
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

		default:
			_, _ = fmt.Fprintf(w, "%s\n", Error("please enter 1-6"))
		}
	}
}

var weekdayList = []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA, rrule.SU}

// promptDays asks the user to select specific days of the week.
func promptDays(prompt PromptFunc, w io.Writer) ([]rrule.Weekday, error) {
	_, _ = fmt.Fprintf(w, "\n%s\n", Text("Select days (comma-separated):"))
	_, _ = fmt.Fprintf(w, "  %s\n", Text("[1] Mon  [2] Tue  [3] Wed  [4] Thu  [5] Fri  [6] Sat  [7] Sun"))

	for {
		input, err := prompt(Text("> "))
		if err != nil {
			return nil, err
		}

		parts := strings.Split(strings.TrimSpace(input), ",")
		var days []rrule.Weekday
		valid := true
		for _, p := range parts {
			n, err := strconv.Atoi(strings.TrimSpace(p))
			if err != nil || n < 1 || n > 7 {
				valid = false
				break
			}
			days = append(days, weekdayList[n-1])
		}

		if !valid || len(days) == 0 {
			_, _ = fmt.Fprintf(w, "%s\n", Error("please enter day numbers 1-7, comma-separated"))
			continue
		}

		return days, nil
	}
}

// promptInterval asks the user for an interval number.
func promptInterval(prompt PromptFunc, w io.Writer, unit string) (int, error) {
	for {
		input, err := prompt(Text(fmt.Sprintf("Every how many %s? ", unit)))
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
		input, err := prompt(Text(fmt.Sprintf("%s (e.g. 2026-03-02, tomorrow): ", label)))
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

// promptTimeRange asks the user for a start and end time, validating order.
func promptTimeRange(prompt PromptFunc, w io.Writer) (string, string, error) {
	for {
		fromInput, err := prompt(Text("Start time (e.g. 9am, 9:00, 14:30): "))
		if err != nil {
			return "", "", err
		}
		fromTod, err := schedule.ParseTimeOfDay(fromInput)
		if err != nil {
			_, _ = fmt.Fprintf(w, "%s\n", Error("invalid time: "+err.Error()))
			continue
		}

		toInput, err := prompt(Text("End time (e.g. 5pm, 17:00, 14:30): "))
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

func parseActionIndex(action string, count int) (int, error) {
	parts := strings.Fields(action)
	if len(parts) < 2 {
		return 0, fmt.Errorf("expected a number after the action")
	}
	n, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid number: %s", parts[1])
	}
	if n < 1 || n > count {
		return 0, fmt.Errorf("number out of range (1-%d)", count)
	}
	return n - 1, nil
}
