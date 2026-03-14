package timetrack

import (
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// DayBudget holds the time budget for a single day.
type DayBudget struct {
	LoggedMinutes    int // total attributed minutes
	ScheduledMinutes int // total scheduled working minutes
	RemainingMinutes int // max(0, scheduled - logged)
}

// ComputeDayBudget computes the full time attribution for a day including
// checkout-attributed time (with idle trimming) and manual logs.
// Used by: status command.
func ComputeDayBudget(
	checkouts []entry.CheckoutEntry,
	logs []entry.Entry,
	commits []entry.CommitEntry,
	daySchedules []schedule.DaySchedule,
	targetDate time.Time,
	now time.Time,
	activity ...ActivityEntries,
) DayBudget {
	year := targetDate.Year()
	month := targetDate.Month()

	report := BuildReport(checkouts, logs, commits, daySchedules, year, month, now, nil, activity...)

	day := targetDate.Day()
	loggedMinutes := 0
	for _, row := range report.Rows {
		if mins, ok := row.Days[day]; ok {
			loggedMinutes += mins
		}
	}

	// Get scheduled minutes for the target day
	scheduledMinutes := 0
	for _, ds := range daySchedules {
		if ds.Date.Day() == day && ds.Date.Month() == month && ds.Date.Year() == year {
			for _, w := range ds.Windows {
				scheduledMinutes += windowMinutes(w)
			}
			break
		}
	}

	remaining := scheduledMinutes - loggedMinutes
	if remaining < 0 {
		remaining = 0
	}

	return DayBudget{
		LoggedMinutes:    loggedMinutes,
		ScheduledMinutes: scheduledMinutes,
		RemainingMinutes: remaining,
	}
}

// ComputeManualLogBudget computes the manual log budget for a day,
// counting only manual log entries and submitted/generated entries.
// excludeID, if non-empty, skips the entry with that ID (for edit).
// Used by: log and edit budget warnings.
func ComputeManualLogBudget(
	logs []entry.Entry,
	daySchedules []schedule.DaySchedule,
	targetDate time.Time,
	excludeID string,
) DayBudget {
	y, m, d := targetDate.Date()

	loggedMinutes := 0
	for _, e := range logs {
		if excludeID != "" && e.ID == excludeID {
			continue
		}
		ey, em, ed := e.Start.Date()
		if ey == y && em == m && ed == d {
			loggedMinutes += e.Minutes
		}
	}

	// Get scheduled minutes for the target day
	scheduledMinutes := 0
	for _, ds := range daySchedules {
		if ds.Date.Day() == d && ds.Date.Month() == m && ds.Date.Year() == y {
			for _, w := range ds.Windows {
				scheduledMinutes += windowMinutes(w)
			}
			break
		}
	}

	remaining := scheduledMinutes - loggedMinutes
	if remaining < 0 {
		remaining = 0
	}

	return DayBudget{
		LoggedMinutes:    loggedMinutes,
		ScheduledMinutes: scheduledMinutes,
		RemainingMinutes: remaining,
	}
}
