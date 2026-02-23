package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// TimeOfDay represents a clock time without a date component.
type TimeOfDay struct {
	Hour   int // 0-23
	Minute int // 0-59
}

// String returns TimeOfDay in "HH:MM" format.
func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", t.Hour, t.Minute)
}

// Schedule is the result of parsing a natural language schedule string.
// Date and RRule are mutually exclusive.
type Schedule struct {
	From  TimeOfDay    // always required
	To    TimeOfDay    // always required
	Date  *time.Time   // specific date (nil if recurring or bare time range)
	RRule *rrule.RRule // recurrence rule (nil if one-off)
}

// ParseSchedule parses a natural language schedule string into a Schedule.
// It expects the format: "from <time> to <time> [date|recurrence]"
func ParseSchedule(input string) (Schedule, error) {
	return ParseScheduleWithNow(input, time.Now())
}

// ParseScheduleWithNow parses a schedule string using the provided time for
// relative date resolution (today, tomorrow, next monday, etc.).
func ParseScheduleWithNow(input string, now time.Time) (Schedule, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))

	fromTime, toTime, remainder, err := extractTimes(normalized)
	if err != nil {
		return Schedule{}, err
	}

	schedule := Schedule{
		From: fromTime,
		To:   toTime,
	}

	remainder = strings.TrimSpace(remainder)
	if remainder == "" {
		return schedule, nil
	}

	// Classify the remainder
	if isRawRRule(remainder) {
		r, err := parseRecurrence(remainder)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid recurrence: %w", err)
		}
		schedule.RRule = r
	} else if isNaturalRecurrence(remainder) {
		r, err := parseRecurrence(remainder)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid recurrence: %w", err)
		}
		schedule.RRule = r
	} else {
		d, err := parseDate(remainder, now)
		if err != nil {
			return Schedule{}, fmt.Errorf("invalid date: %w", err)
		}
		schedule.Date = d
	}

	return schedule, nil
}

// extractTimes parses "from <time> to <time> ..." and returns the two times
// plus the remaining string after the "to <time>" segment.
func extractTimes(s string) (TimeOfDay, TimeOfDay, string, error) {
	if !strings.HasPrefix(s, "from ") {
		return TimeOfDay{}, TimeOfDay{}, "", fmt.Errorf("expected 'from <time> to <time>', got %q", s)
	}

	afterFrom := s[len("from "):]

	toIdx := findToKeyword(afterFrom)
	if toIdx == -1 {
		return TimeOfDay{}, TimeOfDay{}, "", fmt.Errorf("expected 'to <time>' in %q", s)
	}

	fromStr := strings.TrimSpace(afterFrom[:toIdx])
	afterTo := strings.TrimSpace(afterFrom[toIdx+len("to "):])

	fromTime, err := parseTimeOfDay(fromStr)
	if err != nil {
		return TimeOfDay{}, TimeOfDay{}, "", fmt.Errorf("invalid start time %q: %w", fromStr, err)
	}

	// Find where the to-time ends and remainder begins
	toStr, remainder := splitTimeAndRemainder(afterTo)

	toTime, err := parseTimeOfDay(toStr)
	if err != nil {
		return TimeOfDay{}, TimeOfDay{}, "", fmt.Errorf("invalid end time %q: %w", toStr, err)
	}

	return fromTime, toTime, remainder, nil
}

// findToKeyword finds the index of " to " as a word boundary in s.
// Returns -1 if not found. The returned index points at "to " (past the leading space).
func findToKeyword(s string) int {
	pos := strings.Index(s, " to ")
	if pos == -1 {
		return -1
	}
	return pos + 1
}

// splitTimeAndRemainder splits "5pm every weekday" into ("5pm", "every weekday").
// It assumes the first token (possibly with am/pm suffix) is the time.
func splitTimeAndRemainder(s string) (string, string) {
	parts := strings.SplitN(s, " ", 2)
	if len(parts) == 1 {
		return parts[0], ""
	}

	// Check if second token is part of time (e.g., this shouldn't happen with our format)
	return parts[0], parts[1]
}

// isRawRRule returns true if the string looks like a raw RRULE.
// Works on both lowercased and original-case input.
func isRawRRule(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasPrefix(lower, "freq=") || strings.HasPrefix(lower, "rrule:")
}

// isNaturalRecurrence returns true if the string starts with recurrence keywords.
func isNaturalRecurrence(s string) bool {
	return strings.HasPrefix(s, "every ") ||
		s == "every day" ||
		s == "daily" ||
		s == "weekdays" ||
		s == "weekends"
}
