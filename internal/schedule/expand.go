package schedule

import (
	"sort"
	"time"

	"github.com/teambition/rrule-go"
)

// TimeWindow represents a working time range within a single day.
type TimeWindow struct {
	From TimeOfDay
	To   TimeOfDay
}

// DaySchedule represents all working time windows for a specific date.
type DaySchedule struct {
	Date    time.Time
	Windows []TimeWindow
}

// ExpandSchedules evaluates schedule entries into concrete day-by-day working
// hours between from and to (inclusive). RRULEs are expanded, one-off dates
// are checked for inclusion, and bare entries (no rrule, no date) are skipped.
// The result is sorted by date, then by window start time within each day.
func ExpandSchedules(entries []ScheduleEntry, from, to time.Time) ([]DaySchedule, error) {
	dayMap := make(map[string][]TimeWindow)

	for _, entry := range entries {
		s, err := FromEntry(entry)
		if err != nil {
			return nil, err
		}

		windows := make([]TimeWindow, len(s.Ranges))
		for i, r := range s.Ranges {
			windows[i] = TimeWindow(r)
		}

		if s.RRule != nil {
			// For unbounded recurring rules (no DTSTART), set DTSTART to
			// the range start so Between() covers the requested window.
			// For bounded rules (single dates, date ranges), preserve DTSTART.
			opts := s.RRule.OrigOptions
			if opts.Dtstart.IsZero() {
				opts.Dtstart = from
			}
			r, err := rrule.NewRRule(opts)
			if err != nil {
				return nil, err
			}
			dates := r.Between(from, to, true)
			for _, d := range dates {
				key := d.Format("2006-01-02")
				if entry.Override {
					dayMap[key] = append([]TimeWindow{}, windows...)
				} else {
					dayMap[key] = append(dayMap[key], windows...)
				}
			}
		}
		// bare entries (no rrule) are skipped
	}

	result := make([]DaySchedule, 0, len(dayMap))
	for key, windows := range dayMap {
		d, _ := time.Parse("2006-01-02", key)
		sort.Slice(windows, func(i, j int) bool {
			if windows[i].From.Hour != windows[j].From.Hour {
				return windows[i].From.Hour < windows[j].From.Hour
			}
			return windows[i].From.Minute < windows[j].From.Minute
		})
		result = append(result, DaySchedule{Date: d, Windows: windows})
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].Date.Before(result[j].Date)
	})

	return result, nil
}
