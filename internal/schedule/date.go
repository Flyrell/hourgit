package schedule

import (
	"fmt"
	"strings"
	"time"
)

// ParseDate parses a date expression relative to the current time.
func ParseDate(s string) (*time.Time, error) {
	return parseDate(s, time.Now())
}

// parseDate parses a date expression relative to now.
// Supports: "today", "tomorrow", "monday", "next tuesday", "on Monday",
// "2024-01-15", "Jan 2", "Jan 2 2006", "January 2", "January 2 2006",
// "2 Jan", "2 Jan 2006", "2 January", "2 January 2006".
func parseDate(s string, now time.Time) (*time.Time, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Strip "on " prefix
	s = strings.TrimPrefix(s, "on ")
	s = strings.TrimSpace(s)

	// Relative dates
	switch s {
	case "today":
		d := truncateToDay(now)
		return &d, nil
	case "tomorrow":
		d := truncateToDay(now).AddDate(0, 0, 1)
		return &d, nil
	}

	// Weekday names (with optional "next " prefix)
	cleaned := strings.TrimPrefix(s, "next ")
	if wd, ok := parseWeekday(cleaned); ok {
		d := nextWeekday(now, wd)
		return &d, nil
	}

	// Absolute date formats
	layouts := []string{
		"2006-01-02",
		"jan 2",
		"jan 2 2006",
		"january 2",
		"january 2 2006",
		"2 jan",
		"2 jan 2006",
		"2 january",
		"2 january 2006",
	}

	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, now.Location()); err == nil {
			// For layouts without a year, use the current year
			if !hasYear(layout) {
				t = time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, now.Location())
			}
			return &t, nil
		}
	}

	return nil, fmt.Errorf("unrecognized date %q", s)
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

var weekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"monday":    time.Monday,
	"tuesday":   time.Tuesday,
	"wednesday": time.Wednesday,
	"thursday":  time.Thursday,
	"friday":    time.Friday,
	"saturday":  time.Saturday,
}

func parseWeekday(s string) (time.Weekday, bool) {
	wd, ok := weekdays[s]
	return wd, ok
}

// nextWeekday returns the next occurrence of the given weekday after now.
// If now is that weekday, it returns the following week.
func nextWeekday(now time.Time, wd time.Weekday) time.Time {
	today := truncateToDay(now)
	daysAhead := int(wd) - int(today.Weekday())
	if daysAhead <= 0 {
		daysAhead += 7
	}
	return today.AddDate(0, 0, daysAhead)
}

func hasYear(layout string) bool {
	return strings.Contains(layout, "2006")
}
