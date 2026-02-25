package entry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var durationRe = regexp.MustCompile(`^(?:(\d+)h)?(?:(\d+)m)?$`)

// ParseDuration parses a human-friendly duration string into minutes.
// Supported formats: "30m", "3h", "3h30m".
// Returns an error for empty, zero, or negative durations.
func ParseDuration(s string) (int, error) {
	s = strings.ReplaceAll(strings.TrimSpace(strings.ToLower(s)), " ", "")
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid duration format %q (expected e.g. 30m, 3h, 3h30m)", s)
	}

	hours, _ := strconv.Atoi(m[1])
	mins, _ := strconv.Atoi(m[2])

	total := hours*60 + mins
	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return total, nil
}

// FormatMinutes converts a minute count to a human-friendly string.
// Examples: 90 → "1h 30m", 30 → "30m".
func FormatMinutes(m int) string {
	if m <= 0 {
		return "0m"
	}

	hours := m / 60
	mins := m % 60

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins > 0 || hours == 0 {
		parts = append(parts, fmt.Sprintf("%dm", mins))
	}

	return strings.Join(parts, " ")
}
