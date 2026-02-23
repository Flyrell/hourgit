package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandSchedulesWeekdays(t *testing.T) {
	entries := DefaultSchedules() // Mon-Fri 9-5
	// Feb 2026: starts on Sunday, ends on Saturday
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	assert.Equal(t, 20, len(result)) // 20 weekdays in Feb 2026

	for _, ds := range result {
		wd := ds.Date.Weekday()
		assert.True(t, wd >= time.Monday && wd <= time.Friday, "expected weekday, got %s", wd)
		require.Len(t, ds.Windows, 1)
		assert.Equal(t, TimeOfDay{Hour: 9, Minute: 0}, ds.Windows[0].From)
		assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, ds.Windows[0].To)
	}
}

func TestExpandSchedulesMultipleEntries(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "09:00", To: "12:00", RRule: "FREQ=WEEKLY;BYDAY=MO"},
		{From: "13:00", To: "17:00", RRule: "FREQ=WEEKLY;BYDAY=MO"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)

	// Mondays in Feb 2026: 2, 9, 16, 23
	// Default is accumulate: both windows appear
	require.Equal(t, 4, len(result))
	for _, ds := range result {
		assert.Equal(t, time.Monday, ds.Date.Weekday())
		require.Len(t, ds.Windows, 2)
		assert.Equal(t, TimeOfDay{Hour: 9, Minute: 0}, ds.Windows[0].From)
		assert.Equal(t, TimeOfDay{Hour: 12, Minute: 0}, ds.Windows[0].To)
		assert.Equal(t, TimeOfDay{Hour: 13, Minute: 0}, ds.Windows[1].From)
		assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, ds.Windows[1].To)
	}
}

func TestExpandSchedulesOverride(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "09:00", To: "17:00", RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"},
		{From: "08:00", To: "16:00", RRule: "FREQ=WEEKLY;BYDAY=MO", Override: true},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	assert.Equal(t, 20, len(result)) // 20 weekdays in Feb 2026

	for _, ds := range result {
		require.Len(t, ds.Windows, 1)
		if ds.Date.Weekday() == time.Monday {
			// Monday overridden by second entry
			assert.Equal(t, TimeOfDay{Hour: 8, Minute: 0}, ds.Windows[0].From)
			assert.Equal(t, TimeOfDay{Hour: 16, Minute: 0}, ds.Windows[0].To)
		} else {
			// Other weekdays keep base schedule
			assert.Equal(t, TimeOfDay{Hour: 9, Minute: 0}, ds.Windows[0].From)
			assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, ds.Windows[0].To)
		}
	}
}

func TestExpandSchedulesOneOffDate(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "10:00", To: "14:00", RRule: "DTSTART:20260215T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	require.Len(t, result, 1)
	assert.Equal(t, 15, result[0].Date.Day())
	require.Len(t, result[0].Windows, 1)
	assert.Equal(t, TimeOfDay{Hour: 10, Minute: 0}, result[0].Windows[0].From)
}

func TestExpandSchedulesOneOffDateOutOfRange(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "10:00", To: "14:00", RRule: "DTSTART:20260315T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestExpandSchedulesDateRange(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "09:00", To: "17:00", RRule: "DTSTART:20260302T000000Z\nRRULE:FREQ=DAILY;UNTIL=20260306T235959Z"},
	}
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	require.Len(t, result, 5) // Mar 2-6
	assert.Equal(t, 2, result[0].Date.Day())
	assert.Equal(t, 6, result[4].Date.Day())
}

func TestExpandSchedulesBareEntry(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "09:00", To: "17:00"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestExpandSchedulesEmpty(t *testing.T) {
	result, err := ExpandSchedules(nil, time.Now(), time.Now())

	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestExpandSchedulesInvalidEntry(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "bad", To: "17:00", RRule: "FREQ=DAILY"},
	}
	_, err := ExpandSchedules(entries, time.Now(), time.Now())
	assert.Error(t, err)
}

func TestExpandSchedulesAccumulate(t *testing.T) {
	// Split shift: two entries for the same days without override → both windows appear
	entries := []ScheduleEntry{
		{From: "08:00", To: "12:00", RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"},
		{From: "13:00", To: "17:00", RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	assert.Equal(t, 20, len(result)) // 20 weekdays

	for _, ds := range result {
		require.Len(t, ds.Windows, 2)
		assert.Equal(t, TimeOfDay{Hour: 8, Minute: 0}, ds.Windows[0].From)
		assert.Equal(t, TimeOfDay{Hour: 12, Minute: 0}, ds.Windows[0].To)
		assert.Equal(t, TimeOfDay{Hour: 13, Minute: 0}, ds.Windows[1].From)
		assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, ds.Windows[1].To)
	}
}

func TestExpandSchedulesSortedByDate(t *testing.T) {
	entries := []ScheduleEntry{
		{From: "09:00", To: "17:00", RRule: "DTSTART:20260220T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
		{From: "09:00", To: "17:00", RRule: "DTSTART:20260210T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
		{From: "09:00", To: "17:00", RRule: "DTSTART:20260215T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 2, 28, 23, 59, 59, 0, time.UTC)

	result, err := ExpandSchedules(entries, from, to)

	require.NoError(t, err)
	require.Len(t, result, 3)
	assert.Equal(t, 10, result[0].Date.Day())
	assert.Equal(t, 15, result[1].Date.Day())
	assert.Equal(t, 20, result[2].Date.Day())
}
