package schedule

import (
	"fmt"

	"github.com/teambition/rrule-go"
)

// ScheduleEntry is the storable form of a schedule — a time range plus
// a recurrence rule. Single dates and date ranges are represented as RRULEs
// with DTSTART (and optionally UNTIL or COUNT).
type ScheduleEntry struct {
	From     string `json:"from"`               // "HH:MM"
	To       string `json:"to"`                 // "HH:MM"
	RRule    string `json:"rrule"`              // RFC 5545 RRULE string (always present)
	Override bool   `json:"override,omitempty"` // when true, replaces all previous windows for matching days
}

// DefaultSchedules returns the default working schedule: Mon-Fri 9am-5pm.
func DefaultSchedules() []ScheduleEntry {
	return []ScheduleEntry{
		{
			From:  "09:00",
			To:    "17:00",
			RRule: "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		},
	}
}

// ToEntry converts a parsed Schedule into a storable ScheduleEntry.
func ToEntry(s Schedule) ScheduleEntry {
	e := ScheduleEntry{
		From: s.From.String(),
		To:   s.To.String(),
	}
	if s.RRule != nil {
		e.RRule = s.RRule.String()
	}
	return e
}

// FromEntry converts a storable ScheduleEntry back into a Schedule.
func FromEntry(e ScheduleEntry) (Schedule, error) {
	from, err := parseTimeOfDay(e.From)
	if err != nil {
		return Schedule{}, fmt.Errorf("invalid from time %q: %w", e.From, err)
	}

	to, err := parseTimeOfDay(e.To)
	if err != nil {
		return Schedule{}, fmt.Errorf("invalid to time %q: %w", e.To, err)
	}

	s := Schedule{From: from, To: to}

	if e.RRule != "" {
		r, err := rrule.StrToRRule(e.RRule)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid rrule %q: %w", e.RRule, err)
		}
		s.RRule = r
	}

	return s, nil
}
