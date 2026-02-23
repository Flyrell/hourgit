package schedule

import (
	"fmt"

	"github.com/teambition/rrule-go"
)

// Schedule is the parsed in-memory representation of a schedule entry.
type Schedule struct {
	Ranges []TimeOfDayRange // one or more time ranges (always at least one)
	RRule  *rrule.RRule     // recurrence rule (always present for storable schedules)
}

// TimeOfDay represents a clock time without a date component.
type TimeOfDay struct {
	Hour   int // 0-23
	Minute int // 0-59
}

// String returns TimeOfDay in "HH:MM" format.
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// Before reports whether t is strictly before other.
func (t TimeOfDay) Before(other TimeOfDay) bool {
	if t.Hour != other.Hour {
		return t.Hour < other.Hour
	}
	return t.Minute < other.Minute
}
