package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildExportData_LogEntriesGroupedByTask(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "Login flow", Task: "feature-auth"},
		{ID: "l2", Start: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Minutes: 90, Message: "Token refresh", Task: "feature-auth"},
		{ID: "l3", Start: time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), Minutes: 75, Message: "API design research", Task: ""},
	}

	data := BuildExportData(nil, logs, days, year, month, afterMonth(year, month), nil, "Test Project")

	assert.Equal(t, "Test Project", data.ProjectName)
	assert.Equal(t, 2025, data.Year)
	assert.Equal(t, time.January, data.Month)
	require.Equal(t, 1, len(data.Days))

	day := data.Days[0]
	assert.Equal(t, 2, day.Date.Day())
	assert.Equal(t, 225, day.TotalMinutes) // 60+90+75
	assert.Equal(t, 225, data.TotalMinutes)

	// Should have 2 groups: "API design research" (no task, uses message) and "feature-auth"
	require.Equal(t, 2, len(day.Groups))

	// Groups sorted alphabetically
	assert.Equal(t, "API design research", day.Groups[0].Task)
	assert.Equal(t, 75, day.Groups[0].TotalMinutes)
	assert.Equal(t, 1, len(day.Groups[0].Entries))

	assert.Equal(t, "feature-auth", day.Groups[1].Task)
	assert.Equal(t, 150, day.Groups[1].TotalMinutes)
	assert.Equal(t, 2, len(day.Groups[1].Entries))

	// Entries sorted by start time
	assert.Equal(t, "Login flow", day.Groups[1].Entries[0].Message)
	assert.Equal(t, "Token refresh", day.Groups[1].Entries[1].Message)
}

func TestBuildExportData_CheckoutAttribution(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	data := BuildExportData(checkouts, nil, days, year, month, afterMonth(year, month), nil, "Test")

	require.Equal(t, 1, len(data.Days))
	day := data.Days[0]
	assert.Equal(t, 2, day.Date.Day())
	assert.Equal(t, 480, day.TotalMinutes) // 8h

	require.Equal(t, 1, len(day.Groups))
	assert.Equal(t, "feature-x", day.Groups[0].Task)
	assert.Equal(t, 480, day.Groups[0].TotalMinutes)
}

func TestBuildExportData_GeneratedDaysSkipCheckouts(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2), workday(year, month, 3)}

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "Generated work", Task: "feature-x"},
	}

	generatedDays := []string{"2025-01-02"}

	data := BuildExportData(checkouts, logs, days, year, month, afterMonth(year, month), generatedDays, "Test")

	// Day 2 should only have the log entry (checkout skipped due to generated)
	// Day 3 should have checkout attribution
	require.Equal(t, 2, len(data.Days))

	day2 := data.Days[0]
	assert.Equal(t, 2, day2.Date.Day())
	assert.Equal(t, 120, day2.TotalMinutes) // only the log entry

	day3 := data.Days[1]
	assert.Equal(t, 3, day3.Date.Day())
	assert.Equal(t, 480, day3.TotalMinutes) // full checkout day
}

func TestBuildExportData_EmptyMonth(t *testing.T) {
	year, month := 2025, time.January

	data := BuildExportData(nil, nil, nil, year, month, afterMonth(year, month), nil, "Empty")

	assert.Equal(t, 0, len(data.Days))
	assert.Equal(t, 0, data.TotalMinutes)
}

func TestBuildExportData_MultipleDaysSorted(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 6), workday(year, month, 2)}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 6, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "work", Task: "task"},
		{ID: "l2", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 30, Message: "work", Task: "task"},
	}

	data := BuildExportData(nil, logs, days, year, month, afterMonth(year, month), nil, "Test")

	require.Equal(t, 2, len(data.Days))
	// Days should be sorted ascending
	assert.Equal(t, 2, data.Days[0].Date.Day())
	assert.Equal(t, 6, data.Days[1].Date.Day())
	assert.Equal(t, 90, data.TotalMinutes)
}

func TestBuildExportData_ScheduleDeduction(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 480 min total

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "research", Task: "research"},
	}

	data := BuildExportData(checkouts, logs, days, year, month, afterMonth(year, month), nil, "Test")

	require.Equal(t, 1, len(data.Days))
	day := data.Days[0]

	// Should have 2 groups: feature-x (checkout) and research (log)
	require.Equal(t, 2, len(day.Groups))

	// Log takes 120 min, checkout should get 480-120=360 min
	var checkoutMins, logMins int
	for _, g := range day.Groups {
		switch g.Task {
		case "feature-x":
			checkoutMins = g.TotalMinutes
		case "research":
			logMins = g.TotalMinutes
		}
	}
	assert.Equal(t, 360, checkoutMins)
	assert.Equal(t, 120, logMins)
}
