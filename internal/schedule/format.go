package schedule

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// FormatTimeRange formats "HH:MM" times into "H:MM AM - H:MM PM".
func FormatTimeRange(from, to string) string {
	return fmt.Sprintf("%s - %s", format12h(from), format12h(to))
}

// FormatRRule returns a human-readable description of an RRULE string.
func FormatRRule(rruleStr string) string {
	upper := strings.ToUpper(rruleStr)

	parts := make(map[string]string)
	for _, seg := range strings.Split(upper, ";") {
		kv := strings.SplitN(seg, "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}

	freq := parts["FREQ"]
	byday := parts["BYDAY"]
	interval := 1
	if v, ok := parts["INTERVAL"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			interval = n
		}
	}

	// Check for well-known patterns
	if freq == "WEEKLY" && byday != "" {
		days := strings.Split(byday, ",")
		if isWeekdays(days) {
			return "every weekday"
		}
		if isWeekends(days) {
			return "every weekend"
		}
		if len(days) == 1 {
			return "every " + dayAbbrevToName(days[0])
		}
		names := make([]string, len(days))
		for i, d := range days {
			names[i] = dayAbbrevToName(d)
		}
		return "every " + strings.Join(names, ", ")
	}

	if freq == "DAILY" {
		if interval > 1 {
			return fmt.Sprintf("every %d days", interval)
		}
		return "every day"
	}

	if freq == "WEEKLY" {
		if interval > 1 {
			return fmt.Sprintf("every %d weeks", interval)
		}
		return "every week"
	}

	return rruleStr
}

// FormatDaySchedule formats a DaySchedule as "Mon Feb  2:  9:00 AM - 5:00 PM".
// Multiple windows are comma-separated.
func FormatDaySchedule(ds DaySchedule) string {
	parts := make([]string, len(ds.Windows))
	for i, w := range ds.Windows {
		parts[i] = FormatTimeRange(w.From.String(), w.To.String())
	}
	return fmt.Sprintf("%s:  %s", ds.Date.Format("Mon Jan _2"), strings.Join(parts, ", "))
}

// FormatScheduleEntry returns a full human-readable line for a schedule entry.
// Multiple time ranges are joined with " + ".
func FormatScheduleEntry(e ScheduleEntry) string {
	rangeParts := make([]string, len(e.Ranges))
	for i, r := range e.Ranges {
		rangeParts[i] = FormatTimeRange(r.From, r.To)
	}
	timeRange := strings.Join(rangeParts, " + ")

	var result string
	if e.RRule != "" {
		dateInfo := FormatRRuleDateInfo(e.RRule)
		if dateInfo != "" {
			result = fmt.Sprintf("%s, %s", timeRange, dateInfo)
		} else {
			result = fmt.Sprintf("%s, %s", timeRange, FormatRRule(e.RRule))
		}
	} else {
		result = timeRange
	}
	if e.Override {
		result += " (override)"
	}
	return result
}

// FormatRRuleDateInfo extracts date context from an RRULE string.
// Returns a human-readable string for single dates (DTSTART+COUNT=1) and
// date ranges (DTSTART+UNTIL). Returns empty string for unbounded recurring rules.
func FormatRRuleDateInfo(rruleStr string) string {
	lines := strings.Split(rruleStr, "\n")

	var dtstart time.Time
	var rruleLine string
	for _, line := range lines {
		upper := strings.ToUpper(strings.TrimSpace(line))
		if strings.HasPrefix(upper, "DTSTART:") {
			val := strings.TrimPrefix(upper, "DTSTART:")
			if t, err := time.Parse("20060102T150405Z", val); err == nil {
				dtstart = t
			}
		}
		if strings.HasPrefix(upper, "RRULE:") {
			rruleLine = strings.TrimPrefix(upper, "RRULE:")
		} else if !strings.HasPrefix(upper, "DTSTART:") {
			rruleLine = upper
		}
	}

	if dtstart.IsZero() {
		return ""
	}

	parts := make(map[string]string)
	for _, seg := range strings.Split(rruleLine, ";") {
		kv := strings.SplitN(seg, "=", 2)
		if len(kv) == 2 {
			parts[kv[0]] = kv[1]
		}
	}

	// Single date: DTSTART + COUNT=1
	if parts["COUNT"] == "1" {
		return fmt.Sprintf("on %s", dtstart.Format("Jan 2"))
	}

	// Date range: DTSTART + UNTIL
	if untilStr, ok := parts["UNTIL"]; ok {
		if until, err := time.Parse("20060102T150405Z", untilStr); err == nil {
			return fmt.Sprintf("%s – %s", dtstart.Format("Jan 2"), until.Format("Jan 2"))
		}
	}

	return ""
}

// format12h converts "HH:MM" to "H:MM AM/PM".
func format12h(hhmm string) string {
	parts := strings.SplitN(hhmm, ":", 2)
	if len(parts) != 2 {
		return hhmm
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil {
		return hhmm
	}
	m := parts[1]

	suffix := "AM"
	display := h
	if h == 0 {
		display = 12
	} else if h == 12 {
		suffix = "PM"
	} else if h > 12 {
		display = h - 12
		suffix = "PM"
	}

	return fmt.Sprintf("%d:%s %s", display, m, suffix)
}

// matchExactSet returns true if actual contains exactly the expected strings (in any order).
func matchExactSet(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	set := make(map[string]bool, len(expected))
	for _, e := range expected {
		set[e] = false
	}
	for _, a := range actual {
		if _, ok := set[a]; !ok {
			return false
		}
		set[a] = true
	}
	for _, v := range set {
		if !v {
			return false
		}
	}
	return true
}

func isWeekdays(days []string) bool {
	return matchExactSet(days, "MO", "TU", "WE", "TH", "FR")
}

func isWeekends(days []string) bool {
	return matchExactSet(days, "SA", "SU")
}

var dayNames = map[string]string{
	"MO": "Monday",
	"TU": "Tuesday",
	"WE": "Wednesday",
	"TH": "Thursday",
	"FR": "Friday",
	"SA": "Saturday",
	"SU": "Sunday",
}

func dayAbbrevToName(abbrev string) string {
	if name, ok := dayNames[strings.ToUpper(abbrev)]; ok {
		return name
	}
	return abbrev
}
