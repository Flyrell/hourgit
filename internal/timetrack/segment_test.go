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

// --- Synthetic checkout tests ---

func TestBuildSyntheticCheckouts_NoCommits(t *testing.T) {
	result := buildSyntheticCheckouts(nil)
	assert.Nil(t, result)
}

func TestBuildSyntheticCheckouts_SingleRepo(t *testing.T) {
	commits := []entry.CommitEntry{
		{ID: "c1", Timestamp: t9am, Branch: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: t10am, Branch: "main", Repo: "/repoA"},
		{ID: "c3", Timestamp: t11am, Branch: "feat", Repo: "/repoA"},
	}
	result := buildSyntheticCheckouts(commits)
	assert.Nil(t, result)
}

func TestBuildSyntheticCheckouts_MultiRepo(t *testing.T) {
	commits := []entry.CommitEntry{
		{ID: "c1", Timestamp: t9am, Branch: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: t11am, Branch: "feat", Repo: "/repoB"},
	}
	result := buildSyntheticCheckouts(commits)
	assert.Len(t, result, 1)

	// Midpoint of 9:00 and 11:00 = 10:00
	assert.Equal(t, t10am, result[0].Timestamp)
	assert.Equal(t, "feat", result[0].Next)
	assert.Equal(t, "/repoB", result[0].Repo)
}

func TestBuildSyntheticCheckouts_ConsecutiveSameRepo(t *testing.T) {
	commits := []entry.CommitEntry{
		{ID: "c1", Timestamp: t9am, Branch: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: t10am, Branch: "feat", Repo: "/repoA"},
		{ID: "c3", Timestamp: t11am, Branch: "main", Repo: "/repoA"},
	}
	result := buildSyntheticCheckouts(commits)
	assert.Nil(t, result)
}

func TestBuildSyntheticCheckouts_EmptyRepoSkipped(t *testing.T) {
	commits := []entry.CommitEntry{
		{ID: "c1", Timestamp: t9am, Branch: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: t10am, Branch: "feat", Repo: ""},
		{ID: "c3", Timestamp: t11am, Branch: "main", Repo: "/repoB"},
	}
	// c1→c2: c2 has empty repo, skip
	// c2→c3: c2 has empty repo, skip
	result := buildSyntheticCheckouts(commits)
	assert.Nil(t, result)
}

// --- Repo-aware deduplication tests ---

func TestBuildCheckoutSegments_SameBranchDifferentRepos(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	// Two checkouts to "main" but in different repos — should NOT be deduplicated
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoB"},
		{ID: "c3", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Next: "feat", Repo: "/repoA"},
	}

	segments := buildCheckoutSegments(checkouts, nil, year, month, daysInMonth, afterMonth(year, month))

	// Should have 3 segments: main@repoA, main@repoB, feat@repoA
	assert.Equal(t, 3, len(segments))
	assert.Equal(t, "main", segments[0].branch)
	assert.Equal(t, "/repoA", segments[0].repo)
	assert.Equal(t, "main", segments[1].branch)
	assert.Equal(t, "/repoB", segments[1].repo)
	assert.Equal(t, "feat", segments[2].branch)
	assert.Equal(t, "/repoA", segments[2].repo)
}

// --- Repo-aware commit matching tests ---

func TestBuildCheckoutSegments_CommitMatchesByRepo(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	// Two overlapping sessions on "main" in different repos
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoB"},
	}

	// Commit on main in repoB — should only match the repoB session
	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoB", Message: "fix in repoB"},
	}

	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, afterMonth(year, month))

	// repoA session: 9:00-12:00, no commits → single segment
	repoASegs := filterSegmentsByRepo(segments, "/repoA")
	assert.Equal(t, 1, len(repoASegs))
	assert.Equal(t, "", repoASegs[0].message) // no commit message

	// repoB session: 12:00-end, split by commit at 13:00
	repoBSegs := filterSegmentsByRepo(segments, "/repoB")
	assert.Equal(t, 2, len(repoBSegs))
	assert.Equal(t, "fix in repoB", repoBSegs[0].message)
	assert.Equal(t, "", repoBSegs[1].message) // trailing
}

// --- Repo-aware idle gap tests ---

func TestTrimSegmentsByIdleGaps_DifferentRepoNotApplied(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", repo: "/repoA", from: t9am, to: t12pm, message: "work"},
	}

	// Idle gap from repoB — should NOT affect repoA segment
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repoB"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repoB"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 1)
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t12pm, result[0].to)
}

func TestTrimSegmentsByIdleGaps_SameRepoApplied(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", repo: "/repoA", from: t9am, to: t12pm, message: "work"},
	}

	// Idle gap from repoA — SHOULD affect repoA segment
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repoA"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repoA"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 2)
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t10am, result[0].to)
	assert.Equal(t, t11am, result[1].from)
	assert.Equal(t, t12pm, result[1].to)
}

func TestTrimSegmentsByIdleGaps_MultiRepoMixed(t *testing.T) {
	segments := []sessionSegment{
		{branch: "main", repo: "/repoA", from: t9am, to: t12pm, message: "workA"},
		{branch: "feat", repo: "/repoB", from: t9am, to: t12pm, message: "workB"},
	}

	// repoA idle gap 10:00-11:00, repoB idle gap 9:30-10:30
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repoA"},
		{ID: "s2", Timestamp: t930, Repo: "/repoB"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repoA"},
		{ID: "a2", Timestamp: t1030, Repo: "/repoB"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)

	// repoA: [9:00-10:00, 11:00-12:00]
	repoAResult := filterSegmentsByRepo(result, "/repoA")
	assert.Len(t, repoAResult, 2)
	assert.Equal(t, t9am, repoAResult[0].from)
	assert.Equal(t, t10am, repoAResult[0].to)
	assert.Equal(t, t11am, repoAResult[1].from)
	assert.Equal(t, t12pm, repoAResult[1].to)

	// repoB: [9:00-9:30, 10:30-12:00]
	repoBResult := filterSegmentsByRepo(result, "/repoB")
	assert.Len(t, repoBResult, 2)
	assert.Equal(t, t9am, repoBResult[0].from)
	assert.Equal(t, t930, repoBResult[0].to)
	assert.Equal(t, t1030, repoBResult[1].from)
	assert.Equal(t, t12pm, repoBResult[1].to)
}

func TestTrimSegmentsByIdleGaps_EmptyRepoBackwardCompat(t *testing.T) {
	// Segment with empty repo should be affected by ALL gaps
	segments := []sessionSegment{
		{branch: "main", repo: "", from: t9am, to: t12pm, message: "work"},
	}

	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t10am, Repo: "/repoA"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t11am, Repo: "/repoA"},
	}

	result := trimSegmentsByIdleGaps(segments, stops, starts)
	assert.Len(t, result, 2)
	assert.Equal(t, t9am, result[0].from)
	assert.Equal(t, t10am, result[0].to)
	assert.Equal(t, t11am, result[1].from)
	assert.Equal(t, t12pm, result[1].to)
}

func TestBuildIdleGaps_PairsPerRepo(t *testing.T) {
	// Stops and starts from different repos should be paired independently
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t9am, Repo: "/repoA"},
		{ID: "s2", Timestamp: t930, Repo: "/repoB"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t10am, Repo: "/repoA"},
		{ID: "a2", Timestamp: t1030, Repo: "/repoB"},
	}

	gaps := buildIdleGaps(stops, starts)
	assert.Len(t, gaps, 2)

	// Both repos should have their own gap
	repoAGaps := filterGapsByRepoExact(gaps, "/repoA")
	assert.Len(t, repoAGaps, 1)
	assert.Equal(t, t9am, repoAGaps[0].stop)
	assert.Equal(t, t10am, repoAGaps[0].start)

	repoBGaps := filterGapsByRepoExact(gaps, "/repoB")
	assert.Len(t, repoBGaps, 1)
	assert.Equal(t, t930, repoBGaps[0].stop)
	assert.Equal(t, t1030, repoBGaps[0].start)
}

func TestBuildIdleGaps_CrossRepoPairingPrevented(t *testing.T) {
	// Stop from repoA should NOT pair with start from repoB
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: t9am, Repo: "/repoA"},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: t10am, Repo: "/repoB"},
	}

	gaps := buildIdleGaps(stops, starts)
	// repoA has stop but no start in its repo → no gap
	assert.Len(t, gaps, 0)
}

func TestFilterGapsByRepo(t *testing.T) {
	gaps := []idleGap{
		{stop: t9am, start: t10am, repo: "/repoA"},
		{stop: t10am, start: t11am, repo: "/repoB"},
		{stop: t11am, start: t12pm, repo: ""}, // empty repo = universal
	}

	// Filter for repoA: should get repoA gap + empty-repo gap
	filtered := filterGapsByRepo(gaps, "/repoA")
	assert.Len(t, filtered, 2)
	assert.Equal(t, "/repoA", filtered[0].repo)
	assert.Equal(t, "", filtered[1].repo)

	// Filter for repoB: should get repoB gap + empty-repo gap
	filtered = filterGapsByRepo(gaps, "/repoB")
	assert.Len(t, filtered, 2)
	assert.Equal(t, "/repoB", filtered[0].repo)
	assert.Equal(t, "", filtered[1].repo)

	// Filter with empty repo: should get ALL gaps
	filtered = filterGapsByRepo(gaps, "")
	assert.Len(t, filtered, 3)
}

// --- Integration tests ---

func TestBuildCheckoutSegments_MultiRepoWithCommits(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	// Single checkout to main@repoA, then commits alternate repos
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "commit in A"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Branch: "feat", Repo: "/repoB", Message: "commit in B"},
		{ID: "cm3", Timestamp: time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "back to A"},
	}

	now := time.Date(2025, 1, 2, 16, 0, 0, 0, time.UTC)
	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, now)

	// Synthetic checkouts injected at midpoints:
	// cm1@repoA(10:00) → cm2@repoB(12:00): midpoint=11:00, checkout to feat@repoB
	// cm2@repoB(12:00) → cm3@repoA(14:00): midpoint=13:00, checkout to main@repoA
	//
	// Timeline:
	//   main@repoA 9:00-11:00 (commit at 10:00 splits: [9:00-10:00 "commit in A", 10:00-11:00 trailing])
	//   feat@repoB 11:00-13:00 (commit at 12:00 splits: [11:00-12:00 "commit in B", 12:00-13:00 trailing])
	//   main@repoA 13:00-16:00 (commit at 14:00 splits: [13:00-14:00 "back to A", 14:00-16:00 trailing])

	repoASegs := filterSegmentsByRepo(segments, "/repoA")
	repoBSegs := filterSegmentsByRepo(segments, "/repoB")

	// repoA: 4 segments [9:00-10:00, 10:00-11:00, 13:00-14:00, 14:00-16:00]
	assert.Equal(t, 4, len(repoASegs), "repoA segments")
	assert.Equal(t, "commit in A", repoASegs[0].message)
	assert.Equal(t, time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), repoASegs[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), repoASegs[0].to)
	assert.Equal(t, "", repoASegs[1].message) // trailing
	assert.Equal(t, time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), repoASegs[1].from)
	assert.Equal(t, time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), repoASegs[1].to)
	assert.Equal(t, "back to A", repoASegs[2].message)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), repoASegs[2].from)
	assert.Equal(t, time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), repoASegs[2].to)
	assert.Equal(t, "", repoASegs[3].message) // trailing
	assert.Equal(t, time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), repoASegs[3].from)
	assert.Equal(t, time.Date(2025, 1, 2, 16, 0, 0, 0, time.UTC), repoASegs[3].to)

	// repoB: 2 segments [11:00-12:00, 12:00-13:00]
	assert.Equal(t, 2, len(repoBSegs), "repoB segments")
	assert.Equal(t, "feat", repoBSegs[0].branch)
	assert.Equal(t, "commit in B", repoBSegs[0].message)
	assert.Equal(t, time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), repoBSegs[0].from)
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), repoBSegs[0].to)
	assert.Equal(t, "", repoBSegs[1].message) // trailing
	assert.Equal(t, time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), repoBSegs[1].from)
	assert.Equal(t, time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), repoBSegs[1].to)

	// Verify no time double-counting: total = 9:00-16:00 = 420 minutes
	totalMins := 0
	for _, s := range segments {
		totalMins += int(s.to.Sub(s.from).Minutes())
	}
	assert.Equal(t, 420, totalMins)
}

func TestBuildCheckoutSegments_CommitRepoFallbackToCheckoutRange(t *testing.T) {
	year, month := 2025, time.January
	daysInMonth := 31

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Next: "feat", Repo: "/repoA"},
	}

	// Commit with empty repo — should inherit /repoA from the checkout range
	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 12, 0, 0, 0, time.UTC), Branch: "main", Repo: "", Message: "legacy commit"},
	}

	segments := buildCheckoutSegments(checkouts, commits, year, month, daysInMonth, afterMonth(year, month))

	mainSegs := filterSegments(segments, "main")
	assert.Equal(t, 2, len(mainSegs))

	// Commit segment should inherit repo from checkout range
	assert.Equal(t, "/repoA", mainSegs[0].repo)
	assert.Equal(t, "legacy commit", mainSegs[0].message)

	// Trailing segment should also have checkout range's repo
	assert.Equal(t, "/repoA", mainSegs[1].repo)
}

func TestBuildReport_MultiRepoNoDoubleCounting(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17 = 480 min

	// Single checkout, commits alternate between two repos
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "work in A"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Branch: "feat", Repo: "/repoB", Message: "work in B"},
		{ID: "cm3", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "more A"},
	}

	now := afterMonth(year, month)
	report := BuildReport(checkouts, nil, commits, days, year, month, now, nil)

	// Total across all rows should equal schedule capacity (480 min), not exceed it
	totalMins := 0
	for _, row := range report.Rows {
		totalMins += row.TotalMinutes
	}
	assert.Equal(t, 480, totalMins, "total time should equal schedule capacity without double-counting")

	// Should have two rows: main (repoA time) and feat (repoB time)
	assert.Equal(t, 2, len(report.Rows))
}

func TestBuildDetailedReport_MultiRepoWithCommits(t *testing.T) {
	year, month := 2025, time.January
	from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year, month, 31, 0, 0, 0, 0, time.UTC)
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17 = 480 min

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Next: "main", Repo: "/repoA"},
	}

	commits := []entry.CommitEntry{
		{ID: "cm1", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "work in A"},
		{ID: "cm2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Branch: "feat", Repo: "/repoB", Message: "work in B"},
		{ID: "cm3", Timestamp: time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC), Branch: "main", Repo: "/repoA", Message: "more A"},
	}

	report := BuildDetailedReport(checkouts, nil, commits, days, from, to, afterMonth(year, month))

	// Total across all rows should equal 480 min
	totalMins := 0
	for _, row := range report.Rows {
		totalMins += row.TotalMinutes
	}
	assert.Equal(t, 480, totalMins, "total time should equal schedule capacity")

	// Should have entries for both main and feat
	assert.Equal(t, 2, len(report.Rows))
}

// --- Test helpers ---

func filterSegmentsByRepo(segments []sessionSegment, repo string) []sessionSegment {
	var result []sessionSegment
	for _, s := range segments {
		if s.repo == repo {
			result = append(result, s)
		}
	}
	return result
}

func filterGapsByRepoExact(gaps []idleGap, repo string) []idleGap {
	var result []idleGap
	for _, g := range gaps {
		if g.repo == repo {
			result = append(result, g)
		}
	}
	return result
}
