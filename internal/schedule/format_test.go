package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFormatTimeRange(t *testing.T) {
	tests := []struct {
		from, to string
		want     string
	}{
		{"09:00", "17:00", "9:00 AM - 5:00 PM"},
		{"00:00", "12:00", "12:00 AM - 12:00 PM"},
		{"13:30", "22:00", "1:30 PM - 10:00 PM"},
		{"06:00", "14:30", "6:00 AM - 2:30 PM"},
	}
	for _, tt := range tests {
		t.Run(tt.from+"-"+tt.to, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatTimeRange(tt.from, tt.to))
		})
	}
}

func TestFormatRRule(t *testing.T) {
	tests := []struct {
		name  string
		rrule string
		want  string
	}{
		{"weekdays", "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR", "every weekday"},
		{"weekends", "FREQ=WEEKLY;BYDAY=SA,SU", "every weekend"},
		{"single day", "FREQ=WEEKLY;BYDAY=MO", "every Monday"},
		{"multiple days", "FREQ=WEEKLY;BYDAY=MO,WE,FR", "every Monday, Wednesday, Friday"},
		{"daily", "FREQ=DAILY", "every day"},
		{"every 2 days", "FREQ=DAILY;INTERVAL=2", "every 2 days"},
		{"weekly", "FREQ=WEEKLY", "every week"},
		{"every 2 weeks", "FREQ=WEEKLY;INTERVAL=2", "every 2 weeks"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatRRule(tt.rrule))
		})
	}
}

func TestFormatDaySchedule(t *testing.T) {
	tests := []struct {
		name string
		ds   DaySchedule
		want string
	}{
		{
			name: "single window",
			ds: DaySchedule{
				Date:    time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC), // Monday
				Windows: []TimeWindow{{From: TimeOfDay{9, 0}, To: TimeOfDay{17, 0}}},
			},
			want: "Mon Feb  2:  9:00 AM - 5:00 PM",
		},
		{
			name: "multiple windows",
			ds: DaySchedule{
				Date: time.Date(2026, 2, 4, 0, 0, 0, 0, time.UTC), // Wednesday
				Windows: []TimeWindow{
					{From: TimeOfDay{9, 0}, To: TimeOfDay{12, 0}},
					{From: TimeOfDay{13, 0}, To: TimeOfDay{17, 0}},
				},
			},
			want: "Wed Feb  4:  9:00 AM - 12:00 PM, 1:00 PM - 5:00 PM",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatDaySchedule(tt.ds))
		})
	}
}

func TestFormatScheduleEntry(t *testing.T) {
	tests := []struct {
		name  string
		entry ScheduleEntry
		want  string
	}{
		{
			name:  "recurring weekday",
			entry: ScheduleEntry{Ranges: []TimeRange{{From: "09:00", To: "17:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR"},
			want:  "9:00 AM - 5:00 PM, every weekday",
		},
		{
			name:  "one-off date",
			entry: ScheduleEntry{Ranges: []TimeRange{{From: "10:00", To: "14:00"}}, RRule: "DTSTART:20250315T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
			want:  "10:00 AM - 2:00 PM, on Mar 15",
		},
		{
			name:  "date range",
			entry: ScheduleEntry{Ranges: []TimeRange{{From: "09:00", To: "17:00"}}, RRule: "DTSTART:20260302T000000Z\nRRULE:FREQ=DAILY;UNTIL=20260306T235959Z"},
			want:  "9:00 AM - 5:00 PM, Mar 2 – Mar 6",
		},
		{
			name:  "bare time range",
			entry: ScheduleEntry{Ranges: []TimeRange{{From: "08:00", To: "16:00"}}},
			want:  "8:00 AM - 4:00 PM",
		},
		{
			name:  "recurring with override",
			entry: ScheduleEntry{Ranges: []TimeRange{{From: "08:00", To: "16:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO", Override: true},
			want:  "8:00 AM - 4:00 PM, every Monday (override)",
		},
		{
			name: "multiple ranges",
			entry: ScheduleEntry{
				Ranges: []TimeRange{
					{From: "09:00", To: "12:00"},
					{From: "13:00", To: "17:00"},
				},
				RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
			},
			want: "9:00 AM - 12:00 PM + 1:00 PM - 5:00 PM, every weekday",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatScheduleEntry(tt.entry))
		})
	}
}
