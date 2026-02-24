package schedule

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teambition/rrule-go"
)

func TestDefaultSchedules(t *testing.T) {
	schedules := DefaultSchedules()

	require.Len(t, schedules, 1)
	require.Len(t, schedules[0].Ranges, 1)
	assert.Equal(t, "09:00", schedules[0].Ranges[0].From)
	assert.Equal(t, "17:00", schedules[0].Ranges[0].To)
	assert.Equal(t, "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", schedules[0].RRule)
}

func TestToEntryRecurring(t *testing.T) {
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:      rrule.WEEKLY,
		Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
	})
	require.NoError(t, err)

	s := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 9, Minute: 0}, To: TimeOfDay{Hour: 17, Minute: 0}},
		},
		RRule: r,
	}

	entry := ToEntry(s)

	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "17:00", entry.Ranges[0].To)
	assert.Contains(t, entry.RRule, "FREQ=WEEKLY")
	assert.Contains(t, entry.RRule, "BYDAY=MO,TU,WE,TH,FR")
}

func TestToEntryWithDtstart(t *testing.T) {
	dtstart := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:    rrule.DAILY,
		Count:   1,
		Dtstart: dtstart,
	})
	require.NoError(t, err)

	s := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 10, Minute: 30}, To: TimeOfDay{Hour: 14, Minute: 0}},
		},
		RRule: r,
	}

	entry := ToEntry(s)

	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "10:30", entry.Ranges[0].From)
	assert.Equal(t, "14:00", entry.Ranges[0].To)
	assert.Contains(t, entry.RRule, "DTSTART")
	assert.Contains(t, entry.RRule, "FREQ=DAILY")
	assert.Contains(t, entry.RRule, "COUNT=1")
}

func TestToEntryBareTimeRange(t *testing.T) {
	s := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 8, Minute: 0}, To: TimeOfDay{Hour: 16, Minute: 0}},
		},
	}

	entry := ToEntry(s)

	require.Len(t, entry.Ranges, 1)
	assert.Equal(t, "08:00", entry.Ranges[0].From)
	assert.Equal(t, "16:00", entry.Ranges[0].To)
	assert.Empty(t, entry.RRule)
}

func TestToEntryMultipleRanges(t *testing.T) {
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:      rrule.WEEKLY,
		Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
	})
	require.NoError(t, err)

	s := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 9, Minute: 0}, To: TimeOfDay{Hour: 12, Minute: 0}},
			{From: TimeOfDay{Hour: 13, Minute: 0}, To: TimeOfDay{Hour: 17, Minute: 0}},
		},
		RRule: r,
	}

	entry := ToEntry(s)

	require.Len(t, entry.Ranges, 2)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "12:00", entry.Ranges[0].To)
	assert.Equal(t, "13:00", entry.Ranges[1].From)
	assert.Equal(t, "17:00", entry.Ranges[1].To)
}

func TestFromEntryRecurring(t *testing.T) {
	entry := ScheduleEntry{
		Ranges: []TimeRange{{From: "09:00", To: "17:00"}},
		RRule:  "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
	}

	s, err := FromEntry(entry)

	require.NoError(t, err)
	require.Len(t, s.Ranges, 1)
	assert.Equal(t, TimeOfDay{Hour: 9, Minute: 0}, s.Ranges[0].From)
	assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, s.Ranges[0].To)
	require.NotNil(t, s.RRule)
	assert.Equal(t, rrule.WEEKLY, s.RRule.OrigOptions.Freq)
}

func TestFromEntryWithDtstart(t *testing.T) {
	entry := ScheduleEntry{
		Ranges: []TimeRange{{From: "10:00", To: "14:00"}},
		RRule:  "DTSTART:20260315T000000Z\nRRULE:FREQ=DAILY;COUNT=1",
	}

	s, err := FromEntry(entry)

	require.NoError(t, err)
	require.Len(t, s.Ranges, 1)
	assert.Equal(t, TimeOfDay{Hour: 10, Minute: 0}, s.Ranges[0].From)
	assert.Equal(t, TimeOfDay{Hour: 14, Minute: 0}, s.Ranges[0].To)
	require.NotNil(t, s.RRule)
	assert.Equal(t, rrule.DAILY, s.RRule.OrigOptions.Freq)
	assert.Equal(t, 1, s.RRule.OrigOptions.Count)
	assert.Equal(t, 2026, s.RRule.OrigOptions.Dtstart.Year())
	assert.Equal(t, time.March, s.RRule.OrigOptions.Dtstart.Month())
	assert.Equal(t, 15, s.RRule.OrigOptions.Dtstart.Day())
}

func TestFromEntryMultipleRanges(t *testing.T) {
	entry := ScheduleEntry{
		Ranges: []TimeRange{
			{From: "09:00", To: "12:00"},
			{From: "13:00", To: "17:00"},
		},
		RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
	}

	s, err := FromEntry(entry)

	require.NoError(t, err)
	require.Len(t, s.Ranges, 2)
	assert.Equal(t, TimeOfDay{Hour: 9, Minute: 0}, s.Ranges[0].From)
	assert.Equal(t, TimeOfDay{Hour: 12, Minute: 0}, s.Ranges[0].To)
	assert.Equal(t, TimeOfDay{Hour: 13, Minute: 0}, s.Ranges[1].From)
	assert.Equal(t, TimeOfDay{Hour: 17, Minute: 0}, s.Ranges[1].To)
}

func TestFromEntryNoRanges(t *testing.T) {
	entry := ScheduleEntry{Ranges: nil, RRule: "FREQ=DAILY"}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no time ranges")
}

func TestFromEntryInvalidFrom(t *testing.T) {
	entry := ScheduleEntry{Ranges: []TimeRange{{From: "bad", To: "17:00"}}, RRule: "FREQ=DAILY"}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid from time")
}

func TestFromEntryInvalidTo(t *testing.T) {
	entry := ScheduleEntry{Ranges: []TimeRange{{From: "09:00", To: "bad"}}, RRule: "FREQ=DAILY"}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid to time")
}

func TestFromEntryOverMidnight(t *testing.T) {
	entry := ScheduleEntry{Ranges: []TimeRange{{From: "22:00", To: "06:00"}}, RRule: "FREQ=DAILY"}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be before end time")
}

func TestFromEntryOverlappingRanges(t *testing.T) {
	entry := ScheduleEntry{
		Ranges: []TimeRange{
			{From: "09:00", To: "14:00"},
			{From: "13:00", To: "17:00"},
		},
		RRule: "FREQ=DAILY",
	}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "overlap")
}

func TestFromEntryInvalidRRule(t *testing.T) {
	entry := ScheduleEntry{Ranges: []TimeRange{{From: "09:00", To: "17:00"}}, RRule: "INVALID"}
	_, err := FromEntry(entry)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid rrule")
}

func TestRoundTrip(t *testing.T) {
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:      rrule.WEEKLY,
		Byweekday: []rrule.Weekday{rrule.MO, rrule.WE, rrule.FR},
	})
	require.NoError(t, err)

	original := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 8, Minute: 30}, To: TimeOfDay{Hour: 16, Minute: 45}},
		},
		RRule: r,
	}

	entry := ToEntry(original)
	restored, err := FromEntry(entry)

	require.NoError(t, err)
	assert.Equal(t, original.Ranges, restored.Ranges)
	require.NotNil(t, restored.RRule)
	assert.Equal(t, original.RRule.OrigOptions.Freq, restored.RRule.OrigOptions.Freq)
}

func TestRoundTripMultipleRanges(t *testing.T) {
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:      rrule.WEEKLY,
		Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
	})
	require.NoError(t, err)

	original := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 9, Minute: 0}, To: TimeOfDay{Hour: 12, Minute: 0}},
			{From: TimeOfDay{Hour: 13, Minute: 0}, To: TimeOfDay{Hour: 17, Minute: 0}},
		},
		RRule: r,
	}

	entry := ToEntry(original)
	restored, err := FromEntry(entry)

	require.NoError(t, err)
	assert.Equal(t, original.Ranges, restored.Ranges)
}

func TestRoundTripWithDtstart(t *testing.T) {
	dtstart := time.Date(2026, 3, 2, 0, 0, 0, 0, time.UTC)
	until := time.Date(2026, 3, 6, 23, 59, 59, 0, time.UTC)
	r, err := rrule.NewRRule(rrule.ROption{
		Freq:    rrule.DAILY,
		Dtstart: dtstart,
		Until:   until,
	})
	require.NoError(t, err)

	original := Schedule{
		Ranges: []TimeOfDayRange{
			{From: TimeOfDay{Hour: 9, Minute: 0}, To: TimeOfDay{Hour: 17, Minute: 0}},
		},
		RRule: r,
	}

	entry := ToEntry(original)
	restored, err := FromEntry(entry)

	require.NoError(t, err)
	assert.Equal(t, original.Ranges, restored.Ranges)
	require.NotNil(t, restored.RRule)
	assert.Equal(t, rrule.DAILY, restored.RRule.OrigOptions.Freq)
	assert.False(t, restored.RRule.OrigOptions.Dtstart.IsZero())
}

func TestValidateRanges(t *testing.T) {
	t.Run("single valid range", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{{From: "09:00", To: "17:00"}})
		assert.NoError(t, err)
	})

	t.Run("two non-overlapping ranges", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{
			{From: "09:00", To: "12:00"},
			{From: "13:00", To: "17:00"},
		})
		assert.NoError(t, err)
	})

	t.Run("overlapping ranges", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{
			{From: "09:00", To: "14:00"},
			{From: "13:00", To: "17:00"},
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overlap")
	})

	t.Run("over-midnight range", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{{From: "22:00", To: "06:00"}})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must be before end time")
	})

	t.Run("invalid from time", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{{From: "bad", To: "17:00"}})
		assert.Error(t, err)
	})

	t.Run("adjacent ranges (no overlap)", func(t *testing.T) {
		err := ValidateRanges([]TimeRange{
			{From: "09:00", To: "12:00"},
			{From: "12:00", To: "17:00"},
		})
		assert.NoError(t, err)
	})
}

func TestUnmarshalJSONNewFormat(t *testing.T) {
	data := []byte(`{"ranges":[{"from":"09:00","to":"12:00"},{"from":"13:00","to":"17:00"}],"rrule":"FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"}`)

	var entry ScheduleEntry
	err := json.Unmarshal(data, &entry)

	require.NoError(t, err)
	require.Len(t, entry.Ranges, 2)
	assert.Equal(t, "09:00", entry.Ranges[0].From)
	assert.Equal(t, "12:00", entry.Ranges[0].To)
	assert.Equal(t, "13:00", entry.Ranges[1].From)
	assert.Equal(t, "17:00", entry.Ranges[1].To)
}

