package schedule

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var (
	// 9:30am, 9:30pm
	timeColonAMPM = regexp.MustCompile(`^(\d{1,2}):(\d{2})\s*(am|pm)$`)
	// 9.30am, 9.30pm
	timeDotAMPM = regexp.MustCompile(`^(\d{1,2})\.(\d{2})\s*(am|pm)$`)
	// 9am, 2pm
	timeAMPM = regexp.MustCompile(`^(\d{1,2})\s*(am|pm)$`)
	// 14:00, 09:30
	time24h = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)
	// 14.00, 09.30
	timeDot24h = regexp.MustCompile(`^(\d{1,2})\.(\d{2})$`)
)

// ParseTimeOfDay parses a time string into a TimeOfDay.
// Supported formats: "9:30am", "9.30am", "9am", "14:00", "14.00".
func ParseTimeOfDay(s string) (TimeOfDay, error) {
	return parseTimeOfDay(s)
}

// parseTimeOfDay parses a time string into a TimeOfDay.
func parseTimeOfDay(s string) (TimeOfDay, error) {
	s = strings.TrimSpace(strings.ToLower(s))

	if m := timeColonAMPM.FindStringSubmatch(s); m != nil {
		return parseHourMinuteAMPM(m[1], m[2], m[3])
	}

	if m := timeDotAMPM.FindStringSubmatch(s); m != nil {
		return parseHourMinuteAMPM(m[1], m[2], m[3])
	}

	if m := timeAMPM.FindStringSubmatch(s); m != nil {
		return parseHourMinuteAMPM(m[1], "0", m[2])
	}

	if m := time24h.FindStringSubmatch(s); m != nil {
		return parseHourMinute24(m[1], m[2])
	}

	if m := timeDot24h.FindStringSubmatch(s); m != nil {
		return parseHourMinute24(m[1], m[2])
	}

	return TimeOfDay{}, fmt.Errorf("unrecognized time format %q", s)
}

func parseHourMinuteAMPM(hourStr, minStr, ampm string) (TimeOfDay, error) {
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return TimeOfDay{}, err
	}
	minute, err := strconv.Atoi(minStr)
	if err != nil {
		return TimeOfDay{}, err
	}

	if hour < 1 || hour > 12 {
		return TimeOfDay{}, fmt.Errorf("hour %d out of range for 12-hour format", hour)
	}
	if minute < 0 || minute > 59 {
		return TimeOfDay{}, fmt.Errorf("minute %d out of range", minute)
	}

	if ampm == "am" {
		if hour == 12 {
			hour = 0
		}
	} else {
		if hour != 12 {
			hour += 12
		}
	}

	return TimeOfDay{Hour: hour, Minute: minute}, nil
}

func parseHourMinute24(hourStr, minStr string) (TimeOfDay, error) {
	hour, err := strconv.Atoi(hourStr)
	if err != nil {
		return TimeOfDay{}, err
	}
	minute, err := strconv.Atoi(minStr)
	if err != nil {
		return TimeOfDay{}, err
	}

	if hour < 0 || hour > 23 {
		return TimeOfDay{}, fmt.Errorf("hour %d out of range", hour)
	}
	if minute < 0 || minute > 59 {
		return TimeOfDay{}, fmt.Errorf("minute %d out of range", minute)
	}

	return TimeOfDay{Hour: hour, Minute: minute}, nil
}
