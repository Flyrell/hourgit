package timetrack

import (
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// sessionSegment represents a sub-block of a checkout session, split by commits.
type sessionSegment struct {
	branch  string
	repo    string
	from    time.Time
	to      time.Time
	message string // commit message, empty for uncommitted trailing segment
}

// buildCheckoutSegments splits checkout sessions by commits to produce
// finer-grained time segments. Each commit creates a segment from the previous
// boundary to the commit timestamp. Time is attributed backwards from the commit
// — work before a commit is attributed to that commit. Trailing time after the
// last commit becomes an unnamed segment (uncommitted work).
//
// When no commits exist within a session, the entire session becomes one segment.
func buildCheckoutSegments(
	checkouts []entry.CheckoutEntry,
	commits []entry.CommitEntry,
	year int, month time.Month, daysInMonth int,
	now time.Time,
) []sessionSegment {
	loc := now.Location()

	// Sort checkouts chronologically
	sorted := make([]entry.CheckoutEntry, len(checkouts))
	copy(sorted, checkouts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Deduplicate: skip consecutive checkouts to the same branch
	if len(sorted) > 0 {
		deduped := []entry.CheckoutEntry{sorted[0]}
		for i := 1; i < len(sorted); i++ {
			if cleanBranchName(sorted[i].Next) != cleanBranchName(sorted[i-1].Next) {
				deduped = append(deduped, sorted[i])
			}
		}
		sorted = deduped
	}

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEnd := time.Date(year, month, daysInMonth, 23, 59, 59, 0, loc)

	// Build checkout ranges (same logic as buildCheckoutBucket)
	var pairs []checkoutRange
	lastBeforeIdx := -1
	for i, c := range sorted {
		if !c.Timestamp.After(monthStart) {
			lastBeforeIdx = i
		}
	}

	if lastBeforeIdx >= 0 {
		pairs = append(pairs, checkoutRange{
			branch: cleanBranchName(sorted[lastBeforeIdx].Next),
			from:   monthStart,
		})
	}

	for _, c := range sorted {
		if c.Timestamp.After(monthStart) && !c.Timestamp.After(monthEnd) {
			pairs = append(pairs, checkoutRange{
				branch: cleanBranchName(c.Next),
				from:   c.Timestamp,
			})
		}
	}

	lastEnd := monthEnd.Add(time.Second)
	if now.Before(lastEnd) {
		lastEnd = now
	}
	lastEnd = lastEnd.Truncate(time.Minute)
	for i := range pairs {
		if i+1 < len(pairs) {
			pairs[i].to = pairs[i+1].from
		} else {
			pairs[i].to = lastEnd
		}
		pairs[i].from = pairs[i].from.Truncate(time.Minute)
		pairs[i].to = pairs[i].to.Truncate(time.Minute)
	}

	// Sort commits chronologically
	sortedCommits := make([]entry.CommitEntry, len(commits))
	copy(sortedCommits, commits)
	sort.Slice(sortedCommits, func(i, j int) bool {
		return sortedCommits[i].Timestamp.Before(sortedCommits[j].Timestamp)
	})

	// Split each checkout session by commits
	var segments []sessionSegment
	for _, p := range pairs {
		if p.branch == "" {
			continue
		}

		// Find commits within this session's time range on the same branch
		var sessionCommits []entry.CommitEntry
		for _, c := range sortedCommits {
			if c.Timestamp.Before(p.from) || !c.Timestamp.Before(p.to) {
				continue
			}
			if cleanBranchName(c.Branch) == p.branch {
				sessionCommits = append(sessionCommits, c)
			}
		}

		if len(sessionCommits) == 0 {
			// No commits — single segment for the whole session
			segments = append(segments, sessionSegment{
				branch: p.branch,
				from:   p.from,
				to:     p.to,
			})
			continue
		}

		// Split by commits: time before each commit attributed to that commit
		boundary := p.from
		for _, c := range sessionCommits {
			commitTime := c.Timestamp.Truncate(time.Minute)
			if commitTime.After(boundary) {
				segments = append(segments, sessionSegment{
					branch:  p.branch,
					repo:    c.Repo,
					from:    boundary,
					to:      commitTime,
					message: c.Message,
				})
			}
			boundary = commitTime
		}

		// Trailing time after last commit = uncommitted work
		if boundary.Before(p.to) {
			segments = append(segments, sessionSegment{
				branch: p.branch,
				from:   boundary,
				to:     p.to,
			})
		}
	}

	return segments
}

// buildSegmentBucket aggregates segments into per-branch, per-day minutes
// clipped to schedule windows. This replaces buildCheckoutBucket when commits
// are available.
func buildSegmentBucket(
	segments []sessionSegment,
	year int, month time.Month, daysInMonth int,
	scheduleWindows map[int][]schedule.TimeWindow,
	loc *time.Location,
) map[string]map[int]int {
	bucket := make(map[string]map[int]int)
	for _, seg := range segments {
		if seg.branch == "" {
			continue
		}
		if bucket[seg.branch] == nil {
			bucket[seg.branch] = make(map[int]int)
		}
		for day := 1; day <= daysInMonth; day++ {
			windows, ok := scheduleWindows[day]
			if !ok {
				continue
			}
			mins := overlapMinutes(seg.from, seg.to, year, month, day, windows, loc)
			if mins > 0 {
				bucket[seg.branch][day] += mins
			}
		}
	}
	return bucket
}

// segmentCellEntry represents a segment's contribution to a specific (branch, day) cell.
type segmentCellEntry struct {
	branch  string
	day     int
	minutes int
	message string
	start   time.Time
}

// buildSegmentCellEntries converts segments into per-day cell entries clipped
// to schedule windows, preserving commit messages for individual entries.
func buildSegmentCellEntries(
	segments []sessionSegment,
	year int, month time.Month, daysInMonth int,
	scheduleWindows map[int][]schedule.TimeWindow,
	loc *time.Location,
) []segmentCellEntry {
	var entries []segmentCellEntry
	for _, seg := range segments {
		if seg.branch == "" {
			continue
		}
		for day := 1; day <= daysInMonth; day++ {
			windows, ok := scheduleWindows[day]
			if !ok {
				continue
			}
			mins := overlapMinutes(seg.from, seg.to, year, month, day, windows, loc)
			if mins > 0 {
				entries = append(entries, segmentCellEntry{
					branch:  seg.branch,
					day:     day,
					minutes: mins,
					message: seg.message,
					start:   seg.from,
				})
			}
		}
	}
	return entries
}
