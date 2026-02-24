package entry

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input   string
		want    int
		wantErr bool
	}{
		{"30m", 30, false},
		{"3h", 180, false},
		{"3h30m", 210, false},
		{"2d", 2880, false},
		{"1d3h", 1620, false},
		{"1d30m", 1470, false},
		{"1d3h30m", 1650, false},
		{"1H30M", 90, false},    // case insensitive
		{" 2h ", 120, false},    // whitespace trimmed
		{"", 0, true},           // empty
		{"abc", 0, true},        // invalid
		{"0m", 0, true},         // zero
		{"0h0m", 0, true},       // zero
		{"-1h", 0, true},       // negative
		{"3.5h", 0, true},      // fractional
		{"3h30", 0, true},      // missing unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := ParseDuration(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatMinutes(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0m"},
		{30, "30m"},
		{60, "1h 0m"},
		{90, "1h 30m"},
		{1440, "1d 0h 0m"},
		{1500, "1d 1h 0m"},
		{1530, "1d 1h 30m"},
		{-5, "0m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatMinutes(tt.input))
		})
	}
}
