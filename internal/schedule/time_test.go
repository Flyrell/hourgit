package schedule

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTimeOfDay(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    TimeOfDay
		wantErr bool
	}{
		// 12-hour with colon
		{name: "9:30am", input: "9:30am", want: TimeOfDay{Hour: 9, Minute: 30}},
		{name: "9:30pm", input: "9:30pm", want: TimeOfDay{Hour: 21, Minute: 30}},
		{name: "12:00am", input: "12:00am", want: TimeOfDay{Hour: 0, Minute: 0}},
		{name: "12:00pm", input: "12:00pm", want: TimeOfDay{Hour: 12, Minute: 0}},
		{name: "12:30pm", input: "12:30pm", want: TimeOfDay{Hour: 12, Minute: 30}},

		// 12-hour without colon
		{name: "9am", input: "9am", want: TimeOfDay{Hour: 9, Minute: 0}},
		{name: "5pm", input: "5pm", want: TimeOfDay{Hour: 17, Minute: 0}},
		{name: "12am", input: "12am", want: TimeOfDay{Hour: 0, Minute: 0}},
		{name: "12pm", input: "12pm", want: TimeOfDay{Hour: 12, Minute: 0}},

		// 24-hour
		{name: "14:00", input: "14:00", want: TimeOfDay{Hour: 14, Minute: 0}},
		{name: "09:30", input: "09:30", want: TimeOfDay{Hour: 9, Minute: 30}},
		{name: "00:00", input: "00:00", want: TimeOfDay{Hour: 0, Minute: 0}},
		{name: "23:59", input: "23:59", want: TimeOfDay{Hour: 23, Minute: 59}},

		// With spaces around am/pm
		{name: "9 am", input: "9 am", want: TimeOfDay{Hour: 9, Minute: 0}},
		{name: "9:30 pm", input: "9:30 pm", want: TimeOfDay{Hour: 21, Minute: 30}},

		// Dot-separator 12-hour
		{name: "9.30am", input: "9.30am", want: TimeOfDay{Hour: 9, Minute: 30}},
		{name: "9.30pm", input: "9.30pm", want: TimeOfDay{Hour: 21, Minute: 30}},
		{name: "12.00pm", input: "12.00pm", want: TimeOfDay{Hour: 12, Minute: 0}},

		// Dot-separator 24-hour
		{name: "14.00", input: "14.00", want: TimeOfDay{Hour: 14, Minute: 0}},
		{name: "09.30", input: "09.30", want: TimeOfDay{Hour: 9, Minute: 30}},

		// Errors
		{name: "empty", input: "", wantErr: true},
		{name: "garbage", input: "not a time", wantErr: true},
		{name: "hour 25", input: "25:00", wantErr: true},
		{name: "hour 13am", input: "13am", wantErr: true},
		{name: "minute 60", input: "9:60am", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseTimeOfDay(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestTimeOfDayBefore(t *testing.T) {
	tests := []struct {
		name string
		a, b TimeOfDay
		want bool
	}{
		{"earlier hour", TimeOfDay{8, 0}, TimeOfDay{9, 0}, true},
		{"later hour", TimeOfDay{10, 0}, TimeOfDay{9, 0}, false},
		{"same hour earlier minute", TimeOfDay{9, 0}, TimeOfDay{9, 30}, true},
		{"equal", TimeOfDay{9, 0}, TimeOfDay{9, 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.a.Before(tt.b))
		})
	}
}

func TestTimeOfDayString(t *testing.T) {
	assert.Equal(t, "09:00", TimeOfDay{Hour: 9, Minute: 0}.String())
	assert.Equal(t, "17:30", TimeOfDay{Hour: 17, Minute: 30}.String())
	assert.Equal(t, "00:00", TimeOfDay{Hour: 0, Minute: 0}.String())
}
