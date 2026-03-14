package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
)

func weekdaySchedule(fromH, fromM, toH, toM int) []schedule.DaySchedule {
	// Return a week of weekday schedules for June 2025 (Mon=9, Tue=10, Wed=11, Thu=12, Fri=13)
	var ds []schedule.DaySchedule
	for day := 1; day <= 30; day++ {
		d := time.Date(2025, 6, day, 0, 0, 0, 0, time.UTC)
		wd := d.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			continue
		}
		ds = append(ds, schedule.DaySchedule{
			Date: d,
			Windows: []schedule.TimeWindow{
				{
					From: schedule.TimeOfDay{Hour: fromH, Minute: fromM},
					To:   schedule.TimeOfDay{Hour: toH, Minute: toM},
				},
			},
		})
	}
	return ds
}

func TestComputeDayBudgetWithCheckouts(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.UTC) // Wednesday 2pm
	daySchedules := weekdaySchedule(9, 0, 17, 0)

	checkouts := []entry.CheckoutEntry{
		{
			ID:        "abc1234",
			Timestamp: time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC),
			Previous:  "main",
			Next:      "feature/auth",
		},
	}

	budget := ComputeDayBudget(checkouts, nil, nil, daySchedules, now, now)

	// Checked out at 9am, now is 2pm = 5h = 300 minutes of checkout time
	assert.Equal(t, 300, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 180, budget.RemainingMinutes)
}

func TestComputeDayBudgetWithLogs(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.UTC)
	daySchedules := weekdaySchedule(9, 0, 17, 0)

	logs := []entry.Entry{
		{
			ID:      "abc1234",
			Start:   time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC),
			Minutes: 150,
			Message: "morning work",
		},
	}

	budget := ComputeDayBudget(nil, logs, nil, daySchedules, now, now)

	assert.Equal(t, 150, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 330, budget.RemainingMinutes)
}

func TestComputeDayBudgetNonWorkingDay(t *testing.T) {
	now := time.Date(2025, 6, 14, 10, 0, 0, 0, time.UTC) // Saturday
	daySchedules := weekdaySchedule(9, 0, 17, 0)

	budget := ComputeDayBudget(nil, nil, nil, daySchedules, now, now)

	assert.Equal(t, 0, budget.LoggedMinutes)
	assert.Equal(t, 0, budget.ScheduledMinutes)
	assert.Equal(t, 0, budget.RemainingMinutes)
}

func TestComputeDayBudgetWithIdleTrimming(t *testing.T) {
	now := time.Date(2025, 6, 11, 14, 0, 0, 0, time.UTC)
	daySchedules := weekdaySchedule(9, 0, 17, 0)

	checkouts := []entry.CheckoutEntry{
		{
			ID:        "abc1234",
			Timestamp: time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC),
			Previous:  "main",
			Next:      "feature/auth",
		},
	}

	commits := []entry.CommitEntry{
		{
			ID:        "com1234",
			Timestamp: time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC),
			Message:   "commit 1",
			Branch:    "feature/auth",
		},
	}

	// Idle from 10:00 to 12:00 (2 hours)
	stops := []entry.ActivityStopEntry{
		{ID: "stp1234", Timestamp: time.Date(2025, 6, 11, 10, 0, 0, 0, time.UTC)},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "sta1234", Timestamp: time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)},
	}

	budgetWithIdle := ComputeDayBudget(
		checkouts, nil, commits, daySchedules, now, now,
		ActivityEntries{Stops: stops, Starts: starts},
	)

	budgetWithoutIdle := ComputeDayBudget(
		checkouts, nil, commits, daySchedules, now, now,
	)

	// With idle trimming, 2h idle gap should reduce logged time
	assert.Less(t, budgetWithIdle.LoggedMinutes, budgetWithoutIdle.LoggedMinutes)
	assert.Greater(t, budgetWithIdle.RemainingMinutes, budgetWithoutIdle.RemainingMinutes)
}

func TestComputeManualLogBudgetBasic(t *testing.T) {
	daySchedules := []schedule.DaySchedule{
		{
			Date: time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
			Windows: []schedule.TimeWindow{
				{
					From: schedule.TimeOfDay{Hour: 9, Minute: 0},
					To:   schedule.TimeOfDay{Hour: 17, Minute: 0},
				},
			},
		},
	}

	logs := []entry.Entry{
		{ID: "aaa1111", Start: time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC), Minutes: 240, Message: "work"},
	}

	targetDate := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)

	budget := ComputeManualLogBudget(logs, daySchedules, targetDate, "")

	assert.Equal(t, 240, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 240, budget.RemainingMinutes)
}

func TestComputeManualLogBudgetExcludeID(t *testing.T) {
	daySchedules := []schedule.DaySchedule{
		{
			Date: time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
			Windows: []schedule.TimeWindow{
				{
					From: schedule.TimeOfDay{Hour: 9, Minute: 0},
					To:   schedule.TimeOfDay{Hour: 17, Minute: 0},
				},
			},
		},
	}

	logs := []entry.Entry{
		{ID: "aaa1111", Start: time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC), Minutes: 240, Message: "work1"},
		{ID: "bbb2222", Start: time.Date(2025, 6, 11, 13, 0, 0, 0, time.UTC), Minutes: 120, Message: "work2"},
	}

	targetDate := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)

	budget := ComputeManualLogBudget(logs, daySchedules, targetDate, "aaa1111")

	// Only bbb2222 counted (120 minutes)
	assert.Equal(t, 120, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 360, budget.RemainingMinutes)
}

func TestComputeManualLogBudgetNoSchedule(t *testing.T) {
	// Saturday — no schedule
	targetDate := time.Date(2025, 6, 14, 12, 0, 0, 0, time.UTC)

	budget := ComputeManualLogBudget(nil, nil, targetDate, "")

	assert.Equal(t, 0, budget.LoggedMinutes)
	assert.Equal(t, 0, budget.ScheduledMinutes)
	assert.Equal(t, 0, budget.RemainingMinutes)
}

func TestComputeManualLogBudgetOverSchedule(t *testing.T) {
	daySchedules := []schedule.DaySchedule{
		{
			Date: time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
			Windows: []schedule.TimeWindow{
				{
					From: schedule.TimeOfDay{Hour: 9, Minute: 0},
					To:   schedule.TimeOfDay{Hour: 17, Minute: 0},
				},
			},
		},
	}

	logs := []entry.Entry{
		{ID: "aaa1111", Start: time.Date(2025, 6, 11, 9, 0, 0, 0, time.UTC), Minutes: 600, Message: "long day"},
	}

	targetDate := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)

	budget := ComputeManualLogBudget(logs, daySchedules, targetDate, "")

	assert.Equal(t, 600, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 0, budget.RemainingMinutes) // clamped to 0
}

func TestComputeManualLogBudgetDifferentDay(t *testing.T) {
	daySchedules := []schedule.DaySchedule{
		{
			Date: time.Date(2025, 6, 11, 0, 0, 0, 0, time.UTC),
			Windows: []schedule.TimeWindow{
				{
					From: schedule.TimeOfDay{Hour: 9, Minute: 0},
					To:   schedule.TimeOfDay{Hour: 17, Minute: 0},
				},
			},
		},
	}

	// Log on a different day
	logs := []entry.Entry{
		{ID: "aaa1111", Start: time.Date(2025, 6, 10, 9, 0, 0, 0, time.UTC), Minutes: 240, Message: "yesterday"},
	}

	targetDate := time.Date(2025, 6, 11, 12, 0, 0, 0, time.UTC)

	budget := ComputeManualLogBudget(logs, daySchedules, targetDate, "")

	assert.Equal(t, 0, budget.LoggedMinutes)
	assert.Equal(t, 480, budget.ScheduledMinutes)
	assert.Equal(t, 480, budget.RemainingMinutes)
}
