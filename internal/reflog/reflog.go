package reflog

import (
	"regexp"
	"strings"
	"time"
)

// CheckoutRecord represents a single checkout event parsed from git reflog output.
type CheckoutRecord struct {
	CommitRef string
	Timestamp time.Time
	Previous  string
	Next      string
}

// reflogLinePattern matches git reflog lines with --date=iso format.
// Example: "abc1234 HEAD@{2025-06-15 14:30:00 +0200}: checkout: moving from main to feature-x"
var reflogLinePattern = regexp.MustCompile(
	`^([0-9a-f]+)\s+HEAD@\{(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+[+-]\d{4})\}:\s+checkout:\s+moving from (\S+) to (\S+)$`,
)

// ParseReflog parses git reflog output and returns checkout records.
// Only "checkout: moving from X to Y" lines are matched; all other lines are skipped.
// Records are returned in reflog order (newest first).
func ParseReflog(output string) []CheckoutRecord {
	var records []CheckoutRecord

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := reflogLinePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		commitRef := matches[1]
		timestampStr := matches[2]
		prev := matches[3]
		next := matches[4]

		ts, err := time.Parse("2006-01-02 15:04:05 -0700", timestampStr)
		if err != nil {
			continue
		}

		records = append(records, CheckoutRecord{
			CommitRef: commitRef,
			Timestamp: ts.UTC(),
			Previous:  prev,
			Next:      next,
		})
	}

	return records
}
