package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
)

func TestBuildCheckoutSegments_NoCommits(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Previous: "feature-a", Next: "feature-b"},
	}

	segments := buildCheckoutSegments(checkouts, nil, year, month, daysInMonth, afterMonth(year, month))

	assert.Equal(t, 2, len(segments))

	assert.Equal(t, "feature-a", segments[0].branch)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), segments[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), segments[0].to)
	assert.Equal(t, "", segments[0].message)

	assert.Equal(t, "feature-b", segments[1].branch)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), segments[1].from)
	assert.Equal(t, "", segments[1].message)
}

func TestBuildCheckoutSegments_WithCommits(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Previous: "feature-a", Next: "main"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "fix: first commit"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "feat: second commit"},
	}

	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, afterMonth(year, month))

	// feature-a session (9:00-15:00) should be split into 3 segments:
	// 9:00-11:00 (first commit), 11:00-13:00 (second commit), 13:00-15:00 (trailing)
	// plus the main session (15:00-end)
	featureSegments := filterSegments(segments, "feature-a")
	assert.Equal(t, 3, len(featureSegments))

	assert.Equal(t, "fix: first commit", featureSegments[0].message)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), featureSegments[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), featureSegments[0].to)

	assert.Equal(t, "feat: second commit", featureSegments[1].message)
	assert.Equal(t, time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), featureSegments[1].from)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), featureSegments[1].to)

	// Trailing segment has empty message (uncommitted work)
	assert.Equal(t, "", featureSegments[2].message)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), featureSegments[2].from)
	assert.Equal(t, time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), featureSegments[2].to)
}

func TestBuildCheckoutSegments_TrailingUncommittedWork(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	// Single checkout, no subsequent checkout to end the session
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "feat: the commit"},
	}

	now := time.Date(2025, 1, 2, 16, 0, 0, 0, time.UTC)
	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, now)

	assert.Equal(t, 2, len(segments))

	// First segment: 9:00-12:00 with commit message
	assert.Equal(t, "feature-a", segments[0].branch)
	assert.Equal(t, "feat: the commit", segments[0].message)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), segments[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), segments[0].to)

	// Trailing segment: 12:00-16:00 with empty message (uncommitted work)
	assert.Equal(t, "feature-a", segments[1].branch)
	assert.Equal(t, "", segments[1].message)
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), segments[1].from)
	assert.Equal(t, time.Date(2025, 1, 2, 16, 0, 0, 0, time.UTC), segments[1].to)
}

func TestBuildCheckoutSegments_CommitsOnDifferentBranch(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Previous: "feature-a", Next: "main"},
	}

	// Commits on feature-b, not feature-a — should NOT split the feature-a session
	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Branch: "feature-b", Message: "fix: wrong branch"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Branch: "feature-b", Message: "feat: also wrong"},
	}

	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, afterMonth(year, month))

	// feature-a should remain as a single unsplit segment
	featureSegments := filterSegments(segments, "feature-a")
	assert.Equal(t, 1, len(featureSegments))
	assert.Equal(t, "", featureSegments[0].message)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), featureSegments[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), featureSegments[0].to)
}

func TestBuildDetailedReport_WithCommitsSplitsSession(t *testing.T) {
	year, month := 2025, time.January
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, month, 31, 0, 0, 0, 0, time.UTC)

	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17 = 480 min

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "fix: first"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "feat: second"},
	}

	report := BuildDetailedReport(checkouts, nil, commits, days, from, to, afterMonth(year, month))

	assert.Equal(t, 1, len(report.Rows))
	row := findDetailedRow(report, "feature-a")
	assert.NotNil(t, row)

	cd := row.Days[2]
	assert.NotNil(t, cd)

	// 3 entries: 2 commit segments + 1 trailing uncommitted segment
	// 9:00-12:00 (first commit), 12:00-15:00 (second commit), 15:00-17:00 (trailing)
	assert.Equal(t, 3, len(cd.Entries))
	assert.Equal(t, 480, cd.TotalMinutes)

	// Verify commit messages are preserved on first two entries
	assert.Equal(t, "fix: first", cd.Entries[0].Message)
	assert.Equal(t, 180, cd.Entries[0].Minutes) // 3h

	assert.Equal(t, "feat: second", cd.Entries[1].Message)
	assert.Equal(t, 180, cd.Entries[1].Minutes) // 3h

	// Trailing segment gets branch name as message
	assert.Equal(t, "feature-a", cd.Entries[2].Message)
	assert.Equal(t, 120, cd.Entries[2].Minutes) // 2h

	// All should be in-memory (not persisted)
	for _, e := range cd.Entries {
		assert.False(t, e.Persisted)
		assert.Equal(t, "checkout", e.Source)
	}
}

func TestBuildReport_WithCommitsSameTotal(t *testing.T) {
	year, month := 2025, time.January

	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17 = 480 min

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "fix: first"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Branch: "feature-a", Message: "feat: second"},
	}

	now := afterMonth(year, month)

	// With commits
	reportWithCommits := BuildReport(checkouts, nil, commits, days, year, month, now, nil)
	// Without commits
	reportNoCommits := BuildReport(checkouts, nil, nil, days, year, month, now, nil)

	assert.Equal(t, 1, len(reportWithCommits.Rows))
	assert.Equal(t, 1, len(reportNoCommits.Rows))

	// Total time should be the same regardless of commit splitting
	assert.Equal(t, reportNoCommits.Rows[0].TotalMinutes, reportWithCommits.Rows[0].TotalMinutes)
	assert.Equal(t, 480, reportWithCommits.Rows[0].TotalMinutes)
	assert.Equal(t, 480, reportWithCommits.Rows[0].Days[2])
}

// filterSegments returns only segments matching the given branch.
func filterSegments(segments []sessionSegment, branch string) []sessionSegment {
	var result []sessionSegment
	for _, s := range segments {
		if s.branch == branch {
			result = append(result, s)
		}
	}
	return result
}

// --- Idle gap trimming tests ---

func TestTrimSegmentsByIdleGaps_NoGaps(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t10am, message: "work"},
	}

	result := trimSegmentsByIdleGaps(segments, nil, nil)
	assert.Equal(t, segments, result)
}

var (
	t9am  = time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC)
	t930  = time.Date(2025, 1, 2, 9, 30, 0, 0, time.UTC)
	t10am = time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	t1030 = time.Date(2025, 1, 2, 10, 30, 0, 0, time.UTC)
	t11am = time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC)
	t12pm = time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC)
)

func TestTrimSegmentsByIdleGaps_GapInsideSegment(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t12pm, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 2)
	// Before gap: 9:00 - 10:00
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t10am, result[0].to)
	assert.Equal(t, "work", result[0].message)
	// After gap: 11:00 - 12:00
	assert.Equal(t, t11am, result[1].from)
	assert.Equal(t, t12pm, result[1].to)
	assert.Equal(t, "work", result[1].message)
}

func TestTrimSegmentsByIdleGaps_GapAtStart(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t12pm, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: time.Date(2025, 1, 2, 8, 30, 0, 0, time.UTC), Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t10am, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 1)
	// Trimmed start: 10:00 - 12:00
	assert.Equal(t, t10am, result[0].from)
	assert.Equal(t, t12pm, result[0].to)
}

func TestTrimSegmentsByIdleGaps_GapAtEnd(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t12pm, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t11am, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 1)
	// Trimmed end: 9:00 - 11:00
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t11am, result[0].to)
}

func TestTrimSegmentsByIdleGaps_GapFullyCoversSegment(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t10am, to: t11am, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t9am, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t12pm, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 0)
}

func TestTrimSegmentsByIdleGaps_MultipleGapsInOneSegment(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t12pm, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t930, Repo: "/repo"},
		{ID: "s2", Timestamp: t1030, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t10am, Repo: "/repo"},
		{ID: "a2", Timestamp: t11am, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	// Should be: [9:00-9:30], [10:00-10:30], [11:00-12:00]
	assert.Len(t, result, 3)
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t930, result[0].to)
	assert.Equal(t, t10am, result[1].from)
	assert.Equal(t, t1030, result[1].to)
	assert.Equal(t, t11am, result[2].from)
	assert.Equal(t, t12pm, result[2].to)
}

func TestTrimSegmentsByIdleGaps_GapSpansMultipleSegments(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t10am, message: "commit-1"},
		{branch: "main", from: t10am, to: t12pm, message: "commit-2"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t930, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	// First segment [9:00-10:00] trimmed to [9:00-9:30]
	// Second segment [10:00-12:00] trimmed to [11:00-12:00]
	assert.Len(t, result, 2)
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t930, result[0].to)
	assert.Equal(t, "commit-1", result[0].message)
	assert.Equal(t, t11am, result[1].from)
	assert.Equal(t, t12pm, result[1].to)
	assert.Equal(t, "commit-2", result[1].message)
}

func TestTrimSegmentsByIdleGaps_CommitMessagePreserved(t *testing.T) {
	segments := []sessionSegment{
		{branch: "feat", from: t9am, to: t12pm, message: "fix: important bug"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	for _, seg := range result {
		assert.Equal(t, "fix: important bug", seg.message)
		assert.Equal(t, "feat", seg.branch)
	}
}

func TestTrimSegmentsByIdleGaps_NoOverlap(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", from: t9am, to: t10am, message: "work"},
	}

	// Gap is entirely after the segment
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t11am, Repo: "/repo"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t12pm, Repo: "/repo"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 1)
	assert.Equal(t, segments[0], result[0])
}
