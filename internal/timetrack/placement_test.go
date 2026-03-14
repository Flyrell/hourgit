package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
)

func TestFindAvailableSlot_EmptyDay(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	start, err := FindAvailableSlot(nil, windows, target, 60, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_LogAtStart(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Minutes: 60},
	}

	start, err := FindAvailableSlot(logs, windows, target, 60, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_FitsInSecondWindow(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 11, Minute: 0}},
		{From: schedule.TimeOfDay{Hour: 12, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Minutes: 120}, // fills first window
	}

	start, err := FindAvailableSlot(logs, windows, target, 60, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_GapBetweenLogs(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Minutes: 60},
		{ID: "l2", Start: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Minutes: 60},
	}

	// 60min gap at 10:00-11:00
	start, err := FindAvailableSlot(logs, windows, target, 60, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_NoRoom(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 10, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Minutes: 60},
	}

	_, err := FindAvailableSlot(logs, windows, target, 30, time.UTC)
	assert.Error(t, err)
}

func TestFindAvailableSlot_SplitScheduleSkipsGap(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 11, Minute: 0}},
		{From: schedule.TimeOfDay{Hour: 12, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	// Need 3 hours — doesn't fit in first 2h window, but fits in second 5h window
	start, err := FindAvailableSlot(nil, windows, target, 180, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_DifferentDayLogsIgnored(t *testing.T) {
	windows := []schedule.TimeWindow{
		{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
	}
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 3, 9, 0, 0, 0, time.UTC), Minutes: 480}, // different day
	}

	start, err := FindAvailableSlot(logs, windows, target, 60, time.UTC)
	assert.NoError(t, err)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), start)
}

func TestFindAvailableSlot_EmptyWindows(t *testing.T) {
	target := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := FindAvailableSlot(nil, nil, target, 60, time.UTC)
	assert.Error(t, err)

	_, err = FindAvailableSlot(nil, []schedule.TimeWindow{}, target, 60, time.UTC)
	assert.Error(t, err)
}
