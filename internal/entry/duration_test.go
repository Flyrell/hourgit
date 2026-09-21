package entry

import (
	"strconv"
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
		{"1H30M", 90, false},    // case insensitive
		{" 2h ", 120, false},    // whitespace trimmed
		{"1h 30m", 90, false},   // spaces between parts
		{"3h  30m", 210, false}, // multiple spaces between parts
		{"", 0, true},           // empty
		{"abc", 0, true},        // invalid
		{"0m", 0, true},         // zero
		{"0h0m", 0, true},       // zero
		{"-1h", 0, true},        // negative
		{"3.5h", 0, true},       // fractional
		{"3h30", 0, true},       // missing unit
		{"2d", 0, true},         // days not supported
		{"1d3h30m", 0, true},    // days not supported
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
		{60, "1h"},
		{90, "1h 30m"},
		{1440, "24h"},
		{1500, "25h"},
		{1530, "25h 30m"},
		{-5, "0m"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatMinutes(tt.input))
		})
	}
}

func TestRoundMinutes(t *testing.T) {
	tests := []struct {
		input    int
		interval int
		want     int
	}{
		{0, 15, 0},
		{7, 15, 0},
		{8, 15, 15},
		{14, 15, 15},
		{15, 15, 15},
		{22, 15, 15},
		{23, 15, 30},
		{187, 15, 180},
		{188, 15, 195},
		{-5, 15, -5},
		{10, 1, 10}, // interval<=1 passes through unchanged
		{10, 0, 10}, // degenerate interval passes through unchanged
	}

	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.input)+"@"+strconv.Itoa(tt.interval), func(t *testing.T) {
			assert.Equal(t, tt.want, RoundMinutes(tt.input, tt.interval))
		})
	}
}

func TestFormatMinutesRounded(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{7, "0m"},
		{8, "15m"},
		{191, "3h 15m"},
		{187, "3h"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatMinutesRounded(tt.input))
		})
	}
}
