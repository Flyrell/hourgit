package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/teambition/rrule-go"
)

var everyNWeeks = regexp.MustCompile(`^every (\d+) weeks?$`)

// parseRecurrence parses a natural language or raw RRULE recurrence string.
func parseRecurrence(s string) (*rrule.RRule, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	// Raw RRULE passthrough
	if isRawRRule(s) {
		raw := strings.ToUpper(s)
		raw = strings.TrimPrefix(raw, "RRULE:")
		r, err := rrule.StrToRRule(raw)
		if err != nil {
			return nil, fmt.Errorf("invalid RRULE %q: %w", raw, err)
		}
		return r, nil
	}

	// Natural language patterns
	switch s {
	case "every day", "daily":
		return rrule.NewRRule(rrule.ROption{
			Freq: rrule.DAILY,
		})

	case "every weekday", "weekdays":
		return rrule.NewRRule(rrule.ROption{
			Freq:      rrule.WEEKLY,
			Byweekday: []rrule.Weekday{rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR},
		})

	case "every weekend", "weekends":
		return rrule.NewRRule(rrule.ROption{
			Freq:      rrule.WEEKLY,
			Byweekday: []rrule.Weekday{rrule.SA, rrule.SU},
		})

	case "every other week", "every second week":
		return rrule.NewRRule(rrule.ROption{
			Freq:     rrule.WEEKLY,
			Interval: 2,
		})
	}

	// "every monday", "every tuesday", etc.
	if strings.HasPrefix(s, "every ") {
		dayName := strings.TrimPrefix(s, "every ")

		// "every N weeks"
		if m := everyNWeeks.FindStringSubmatch(s); m != nil {
			n, _ := strconv.Atoi(m[1])
			return rrule.NewRRule(rrule.ROption{
				Freq:     rrule.WEEKLY,
				Interval: n,
			})
		}

		if wd, ok := rruleWeekday(dayName); ok {
			return rrule.NewRRule(rrule.ROption{
				Freq:      rrule.WEEKLY,
				Byweekday: []rrule.Weekday{wd},
			})
		}
	}

	return nil, fmt.Errorf("unrecognized recurrence %q", s)
}

var rruleWeekdays = map[string]rrule.Weekday{
	"sunday":    rrule.SU,
	"monday":    rrule.MO,
	"tuesday":   rrule.TU,
	"wednesday": rrule.WE,
	"thursday":  rrule.TH,
	"friday":    rrule.FR,
	"saturday":  rrule.SA,
}

func rruleWeekday(s string) (rrule.Weekday, bool) {
	wd, ok := rruleWeekdays[s]
	return wd, ok
}
