package schedule

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDate(t *testing.T) {
	// Fixed reference time: Wednesday, January 15, 2025
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name    string
		input   string
		want    time.Time
		wantErr bool
	}{
		// Relative
		{
			name:  "today",
			input: "today",
			want:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "tomorrow",
			input: "tomorrow",
			want:  time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		},

		// Weekday (now is Wednesday)
		{
			name:  "monday",
			input: "monday",
			want:  time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "next tuesday",
			input: "next tuesday",
			want:  time.Date(2025, 1, 21, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "friday",
			input: "friday",
			want:  time.Date(2025, 1, 17, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "wednesday (same day goes to next week)",
			input: "wednesday",
			want:  time.Date(2025, 1, 22, 0, 0, 0, 0, time.UTC),
		},

		// With "on " prefix
		{
			name:  "on monday",
			input: "on monday",
			want:  time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		},

		// Absolute dates
		{
			name:  "ISO date",
			input: "2024-01-15",
			want:  time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "jan 2",
			input: "jan 2",
			want:  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "jan 2 2026",
			input: "jan 2 2026",
			want:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "january 15",
			input: "january 15",
			want:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "january 15 2026",
			input: "january 15 2026",
			want:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Day-first formats
		{
			name:  "2 jan",
			input: "2 jan",
			want:  time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "2 jan 2026",
			input: "2 jan 2026",
			want:  time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "15 january",
			input: "15 january",
			want:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "15 january 2026",
			input: "15 january 2026",
			want:  time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC),
		},

		// Errors
		{name: "empty", input: "", wantErr: true},
		{name: "garbage", input: "not a date", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input, now)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}

func TestParseDate_NonUTCTimezone(t *testing.T) {
	// Use a fixed-offset timezone to avoid dependency on tzdata in Docker.
	cet := time.FixedZone("CET", 1*60*60) // UTC+1

	// Wednesday, January 15, 2025 in CET
	now := time.Date(2025, 1, 15, 10, 30, 0, 0, cet)

	tests := []struct {
		name  string
		input string
		want  time.Time
	}{
		{
			name:  "today returns UTC midnight",
			input: "today",
			want:  time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "tomorrow returns UTC midnight",
			input: "tomorrow",
			want:  time.Date(2025, 1, 16, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "weekday returns UTC midnight",
			input: "monday",
			want:  time.Date(2025, 1, 20, 0, 0, 0, 0, time.UTC),
		},
		{
			name:  "absolute date returns UTC midnight",
			input: "2025-03-15",
			want:  time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseDate(tt.input, now)
			require.NoError(t, err)
			require.NotNil(t, got)
			assert.Equal(t, tt.want, *got)
		})
	}
}
