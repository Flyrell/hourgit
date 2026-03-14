package timetrack

import (
	"fmt"
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// FindAvailableSlot finds the first available time slot within the schedule
// windows for the given date that can accommodate the requested minutes,
// without overlapping any existing log entries.
func FindAvailableSlot(
	existingLogs []entry.Entry,
	windows []schedule.TimeWindow,
	targetDate time.Time,
	minutes int,
	loc *time.Location,
) (time.Time, error) {
	y, m, d := targetDate.Date()

	// Build occupied ranges from existing logs on targetDate
	type timeRange struct {
		from, to int // minutes from midnight
	}
	var occupied []timeRange
	for _, l := range existingLogs {
		ls := l.Start.In(loc)
		if ls.Year() != y || ls.Month() != m || ls.Day() != d {
			continue
		}
		fromMins := ls.Hour()*60 + ls.Minute()
		occupied = append(occupied, timeRange{from: fromMins, to: fromMins + l.Minutes})
	}
	sort.Slice(occupied, func(i, j int) bool {
		return occupied[i].from < occupied[j].from
	})

	// Walk schedule windows chronologically, find first gap that fits
	for _, w := range windows {
		wFrom := w.From.Hour*60 + w.From.Minute
		wTo := w.To.Hour*60 + w.To.Minute

		cursor := wFrom
		for _, occ := range occupied {
			if occ.to <= cursor {
				continue // occupied range is before cursor
			}
			if occ.from >= wTo {
				break // occupied range is past this window
			}
			// Check gap before this occupied range
			if occ.from > cursor {
				gap := occ.from - cursor
				if gap >= minutes {
					return time.Date(y, m, d, cursor/60, cursor%60, 0, 0, loc), nil
				}
			}
			// Move cursor past this occupied range
			if occ.to > cursor {
				cursor = occ.to
			}
		}

		// Check remaining gap after all occupied ranges in this window
		if cursor < wTo {
			gap := wTo - cursor
			if gap >= minutes {
				return time.Date(y, m, d, cursor/60, cursor%60, 0, 0, loc), nil
			}
		}
	}

	return time.Time{}, fmt.Errorf("no available slot for %s in today's schedule", entry.FormatMinutes(minutes))
}
