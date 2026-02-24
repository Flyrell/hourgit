package entry

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var durationRe = regexp.MustCompile(`^(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?$`)

// ParseDuration parses a human-friendly duration string into minutes.
// Supported formats: "30m", "3h", "3h30m", "2d", "1d3h", "1d30m", "1d3h30m".
// 1d = 24h. Returns an error for empty, zero, or negative durations.
func ParseDuration(s string) (int, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	m := durationRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid duration format %q (expected e.g. 30m, 3h, 1d3h30m)", s)
	}

	days, _ := strconv.Atoi(m[1])
	hours, _ := strconv.Atoi(m[2])
	mins, _ := strconv.Atoi(m[3])

	total := days*24*60 + hours*60 + mins
	if total <= 0 {
		return 0, fmt.Errorf("duration must be positive")
	}

	return total, nil
}

// FormatMinutes converts a minute count to a human-friendly string.
// Examples: 90 → "1h 30m", 1500 → "1d 1h 0m", 30 → "30m".
func FormatMinutes(m int) string {
	if m <= 0 {
		return "0m"
	}

	days := m / (24 * 60)
	m %= 24 * 60
	hours := m / 60
	mins := m % 60

	var parts []string
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 || days > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	parts = append(parts, fmt.Sprintf("%dm", mins))

	return strings.Join(parts, " ")
}
