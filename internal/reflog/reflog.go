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

// CommitRecord represents a single commit event parsed from git reflog output.
type CommitRecord struct {
	CommitRef string
	Timestamp time.Time
	Message   string
}

// reflogLinePattern matches git reflog lines with --date=iso format.
// Example: "abc1234 HEAD@{2025-06-15 14:30:00 +0200}: checkout: moving from main to feature-x"
var reflogLinePattern = regexp.MustCompile(
	`^([0-9a-f]+)\s+HEAD@\{(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+[+-]\d{4})\}:\s+checkout:\s+moving from (\S+) to (\S+)$`,
)

// commitLinePattern matches commit and amend lines from git reflog output.
// Example: "abc1234 HEAD@{2025-06-15 14:30:00 +0200}: commit: add login form"
// Example: "abc1234 HEAD@{2025-06-15 14:30:00 +0200}: commit (amend): fix typo"
var commitLinePattern = regexp.MustCompile(
	`^([0-9a-f]+)\s+HEAD@\{(\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}:\d{2}\s+[+-]\d{4})\}:\s+commit(?:\s+\(amend\))?:\s+(.+)$`,
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

// ParseCommits parses git reflog output and returns commit records.
// Only "commit:" and "commit (amend):" lines are matched; merge, rebase, pull, and
// other lines are skipped. Records are returned in reflog order (newest first).
func ParseCommits(output string) []CommitRecord {
	var records []CommitRecord

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		matches := commitLinePattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		commitRef := matches[1]
		timestampStr := matches[2]
		message := matches[3]

		ts, err := time.Parse("2006-01-02 15:04:05 -0700", timestampStr)
		if err != nil {
			continue
		}

		records = append(records, CommitRecord{
			CommitRef: commitRef,
			Timestamp: ts.UTC(),
			Message:   message,
		})
	}

	return records
}
