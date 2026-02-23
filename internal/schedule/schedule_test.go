package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teambition/rrule-go"
)

func TestParseSchedule(t *testing.T) {
	// Fixed reference time: Wednesday, January 15, 2025
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name      string
		input     string
		wantFrom  TimeOfDay
		wantTo    TimeOfDay
		wantDate  *time.Time
		wantRRule bool
		wantFreq  rrule.Frequency
		wantDays  []rrule.Weekday
		wantIntvl int
		wantErr   bool
	}{
		{
			name:     "bare time range",
			input:    "from 9am to 5pm",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
		},
		{
			name:     "24h time range",
			input:    "from 9:00 to 17:00",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
		},
		{
			name:     "today",
			input:    "from 9:00 to 17:00 today",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
			wantDate: timePtr(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:     "tomorrow",
			input:    "from 9am to 5pm tomorrow",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
			wantDate: timePtr(time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:     "on Monday",
			input:    "from 10am to 2pm on Monday",
			wantFrom: TimeOfDay{Hour: 10, Minute: 0},
			wantTo:   TimeOfDay{Hour: 14, Minute: 0},
			wantDate: timePtr(time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:     "ISO date",
			input:    "from 9am to 5pm on 2024-01-15",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
			wantDate: timePtr(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)),
		},
		{
			name:      "every day",
			input:     "from 9am to 5pm every day",
			wantFrom:  TimeOfDay{Hour: 9, Minute: 0},
			wantTo:    TimeOfDay{Hour: 17, Minute: 0},
			wantRRule: true,
			wantFreq:  rrule.DAILY,
		},
		{
			name:      "every weekday",
			input:     "from 9am to 5pm every weekday",
			wantFrom:  TimeOfDay{Hour: 9, Minute: 0},
			wantTo:    TimeOfDay{Hour: 17, Minute: 0},
			wantRRule: true,
			wantFreq:  rrule.WEEKLY,
			wantDays:  []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
		},
		{
			name:      "every second week",
			input:     "from 9am to 1pm every second week",
			wantFrom:  TimeOfDay{Hour: 9, Minute: 0},
			wantTo:    TimeOfDay{Hour: 13, Minute: 0},
			wantRRule: true,
			wantFreq:  rrule.WEEKLY,
			wantIntvl: 2,
		},
		{
			name:      "every monday",
			input:     "from 9am to 5pm every monday",
			wantFrom:  TimeOfDay{Hour: 9, Minute: 0},
			wantTo:    TimeOfDay{Hour: 17, Minute: 0},
			wantRRule: true,
			wantFreq:  rrule.WEEKLY,
			wantDays:  []rrule.Weekday{rrule.MO},
		},
		{
			name:      "raw RRULE",
			input:     "from 9am to 5pm FREQ=WEEKLY;BYDAY=MO,WE,FR",
			wantFrom:  TimeOfDay{Hour: 9, Minute: 0},
			wantTo:    TimeOfDay{Hour: 17, Minute: 0},
			wantRRule: true,
			wantFreq:  rrule.WEEKLY,
			wantDays:  []rrule.Weekday{rrule.MO, rrule.WE, rrule.FR},
		},

		// Case insensitivity
		{
			name:     "uppercase FROM TO",
			input:    "From 9AM To 5PM",
			wantFrom: TimeOfDay{Hour: 9, Minute: 0},
			wantTo:   TimeOfDay{Hour: 17, Minute: 0},
		},

		// Errors
		{name: "empty", input: "", wantErr: true},
		{name: "no from keyword", input: "9am to 5pm", wantErr: true},
		{name: "no to keyword", input: "from 9am 5pm", wantErr: true},
		{name: "bad start time", input: "from abc to 5pm", wantErr: true},
		{name: "bad end time", input: "from 9am to xyz", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseScheduleWithNow(tt.input, now)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantFrom, got.From, "From mismatch")
			assert.Equal(t, tt.wantTo, got.To, "To mismatch")

			if tt.wantDate != nil {
				require.NotNil(t, got.Date, "expected Date to be set")
				assert.Equal(t, *tt.wantDate, *got.Date, "Date mismatch")
			} else {
				assert.Nil(t, got.Date, "expected Date to be nil")
			}

			if tt.wantRRule {
				require.NotNil(t, got.RRule, "expected RRule to be set")
				opts := got.RRule.OrigOptions
				assert.Equal(t, tt.wantFreq, opts.Freq, "Freq mismatch")
				if tt.wantDays != nil {
					assert.Equal(t, tt.wantDays, opts.Byweekday, "Byweekday mismatch")
				}
				if tt.wantIntvl > 0 {
					assert.Equal(t, tt.wantIntvl, opts.Interval, "Interval mismatch")
				}
			} else {
				assert.Nil(t, got.RRule, "expected RRule to be nil")
			}
		})
	}
}

func timePtr(t time.Time) *time.Time {
	return &t
}
