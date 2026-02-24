package schedule

import (
	"fmt"
	"sort"

	"github.com/teambition/rrule-go"
)

// TimeRange is the storable form of a time range — "HH:MM" strings.
type TimeRange struct {
	From string `json:"from"` // "HH:MM"
	To   string `json:"to"`   // "HH:MM"
}

// TimeOfDayRange is the parsed form of a time range.
type TimeOfDayRange struct {
	From TimeOfDay
	To   TimeOfDay
}

// ScheduleEntry is the storable form of a schedule — one or more time ranges
// plus a recurrence rule. Single dates and date ranges are represented as RRULEs
// with DTSTART (and optionally UNTIL or COUNT).
type ScheduleEntry struct {
	Ranges   []TimeRange `json:"ranges"`
	RRule    string      `json:"rrule"`              // RFC 5545 RRULE string (always present)
	Override bool        `json:"override,omitempty"` // when true, replaces all previous windows for matching days
}

// DefaultSchedules returns the default working schedule: Mon-Fri 9am-5pm.
func DefaultSchedules() []ScheduleEntry {
	return []ScheduleEntry{
		{
			Ranges: []TimeRange{{From: "09:00", To: "17:00"}},
			RRule:  "FREQ=WEEKLY;BYDAY=MO,TU,WE,TH,FR",
		},
	}
}

// ToEntry converts a parsed Schedule into a storable ScheduleEntry.
func ToEntry(s Schedule) ScheduleEntry {
	ranges := make([]TimeRange, len(s.Ranges))
	for i, r := range s.Ranges {
		ranges[i] = TimeRange{From: r.From.String(), To: r.To.String()}
	}

	e := ScheduleEntry{Ranges: ranges}
	if s.RRule != nil {
		e.RRule = s.RRule.String()
	}
	return e
}

// FromEntry converts a storable ScheduleEntry back into a Schedule.
func FromEntry(e ScheduleEntry) (Schedule, error) {
	if len(e.Ranges) == 0 {
		return Schedule{}, fmt.Errorf("schedule entry has no time ranges")
	}

	ranges := make([]TimeOfDayRange, len(e.Ranges))
	for i, r := range e.Ranges {
		from, err := parseTimeOfDay(r.From)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid from time %q: %w", r.From, err)
		}
		to, err := parseTimeOfDay(r.To)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid to time %q: %w", r.To, err)
		}
		if !from.Before(to) {
			return Schedule{}, fmt.Errorf("start time %s must be before end time %s", r.From, r.To)
		}
		ranges[i] = TimeOfDayRange{From: from, To: to}
	}

	if err := validateNoOverlap(ranges); err != nil {
		return Schedule{}, err
	}

	s := Schedule{Ranges: ranges}

	if e.RRule != "" {
		r, err := rrule.StrToRRule(e.RRule)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid rrule %q: %w", e.RRule, err)
		}
		s.RRule = r
	}

	return s, nil
}

// ValidateRanges validates a slice of TimeRange values for use by the CLI
// during interactive input. It checks that each range has from < to and
// that ranges don't overlap.
func ValidateRanges(ranges []TimeRange) error {
	parsed := make([]TimeOfDayRange, len(ranges))
	for i, r := range ranges {
		from, err := parseTimeOfDay(r.From)
		if err != nil {
			return fmt.Errorf("invalid from time %q: %w", r.From, err)
		}
		to, err := parseTimeOfDay(r.To)
		if err != nil {
			return fmt.Errorf("invalid to time %q: %w", r.To, err)
		}
		if !from.Before(to) {
			return fmt.Errorf("start time %s must be before end time %s", r.From, r.To)
		}
		parsed[i] = TimeOfDayRange{From: from, To: to}
	}
	return validateNoOverlap(parsed)
}

// validateNoOverlap checks that no two ranges overlap. Ranges are assumed to
// already be individually valid (from < to). Sorts a copy by start time and
// checks each pair.
func validateNoOverlap(ranges []TimeOfDayRange) error {
	if len(ranges) < 2 {
		return nil
	}

	sorted := make([]TimeOfDayRange, len(ranges))
	copy(sorted, ranges)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].From.Before(sorted[j].From)
	})

	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		curr := sorted[i]
		// Overlap if curr.From < prev.To
		if curr.From.Before(prev.To) {
			return fmt.Errorf("time ranges overlap: %s-%s and %s-%s",
				prev.From.String(), prev.To.String(),
				curr.From.String(), curr.To.String())
		}
	}

	return nil
}
