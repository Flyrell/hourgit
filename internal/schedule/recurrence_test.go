package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/teambition/rrule-go"
)

func TestParseRecurrence(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantFreq  rrule.Frequency
		wantIntvl int
		wantDays  []rrule.Weekday
		wantErr   bool
	}{
		{
			name:     "every day",
			input:    "every day",
			wantFreq: rrule.DAILY,
		},
		{
			name:     "daily",
			input:    "daily",
			wantFreq: rrule.DAILY,
		},
		{
			name:     "every weekday",
			input:    "every weekday",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
		},
		{
			name:     "weekdays",
			input:    "weekdays",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
		},
		{
			name:     "every weekend",
			input:    "every weekend",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.SA, rrule.SU},
		},
		{
			name:     "weekends",
			input:    "weekends",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.SA, rrule.SU},
		},
		{
			name:      "every second week",
			input:     "every second week",
			wantFreq:  rrule.WEEKLY,
			wantIntvl: 2,
		},
		{
			name:      "every other week",
			input:     "every other week",
			wantFreq:  rrule.WEEKLY,
			wantIntvl: 2,
		},
		{
			name:     "every monday",
			input:    "every monday",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.MO},
		},
		{
			name:     "every friday",
			input:    "every friday",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.FR},
		},
		{
			name:      "every 3 weeks",
			input:     "every 3 weeks",
			wantFreq:  rrule.WEEKLY,
			wantIntvl: 3,
		},
		{
			name:     "raw RRULE with FREQ prefix",
			input:    "FREQ=WEEKLY;BYDAY=MO,WE,FR",
			wantFreq: rrule.WEEKLY,
			wantDays: []rrule.Weekday{rrule.MO, rrule.WE, rrule.FR},
		},

		// Errors
		{name: "empty", input: "", wantErr: true},
		{name: "garbage", input: "not a recurrence", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseRecurrence(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)

			opts := got.OrigOptions
			assert.Equal(t, tt.wantFreq, opts.Freq, "frequency mismatch")

			if tt.wantIntvl > 0 {
				assert.Equal(t, tt.wantIntvl, opts.Interval, "interval mismatch")
			}

			if tt.wantDays != nil {
				assert.Equal(t, tt.wantDays, opts.Byweekday, "weekdays mismatch")
			}
		})
	}
}
