package timetrack

import (
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// idleGap represents a paired [stop, start] idle period.
type idleGap struct {
	stop  time.Time // activity_stop timestamp (last file change before idle)
	start time.Time // activity_start timestamp (first file change after idle)
	repo  string    // repo this gap belongs to (empty = applies to all)
}

// buildIdleGaps pairs activity_stop and activity_start entries into idle gaps.
// Stops and starts are grouped by repo and paired within each group.
// Gaps are sorted chronologically by stop time.
func buildIdleGaps(stops []entry.ActivityStopEntry, starts []entry.ActivityStartEntry) []idleGap {
	// Group stops by repo
	stopsByRepo := make(map[string][]entry.ActivityStopEntry)
	for _, s := range stops {
		stopsByRepo[s.Repo] = append(stopsByRepo[s.Repo], s)
	}

	// Group starts by repo
	startsByRepo := make(map[string][]entry.ActivityStartEntry)
	for _, s := range starts {
		startsByRepo[s.Repo] = append(startsByRepo[s.Repo], s)
	}

	// Collect all repo keys
	repos := make(map[string]bool)
	for r := range stopsByRepo {
		repos[r] = true
	}

	var gaps []idleGap
	for repo := range repos {
		repoStops := stopsByRepo[repo]
		repoStarts := startsByRepo[repo]
		if len(repoStarts) == 0 {
			continue
		}

		// Sort stops and starts by timestamp
		sort.Slice(repoStops, func(i, j int) bool {
			return repoStops[i].Timestamp.Before(repoStops[j].Timestamp)
		})
		sort.Slice(repoStarts, func(i, j int) bool {
			return repoStarts[i].Timestamp.Before(repoStarts[j].Timestamp)
		})

		// Pair each stop with the next start that comes after it
		startIdx := 0
		for _, stop := range repoStops {
			for startIdx < len(repoStarts) && !repoStarts[startIdx].Timestamp.After(stop.Timestamp) {
				startIdx++
			}
			if startIdx < len(repoStarts) {
				gaps = append(gaps, idleGap{
					stop:  stop.Timestamp,
					start: repoStarts[startIdx].Timestamp,
					repo:  repo,
				})
				startIdx++
			}
		}
	}

	// Sort gaps chronologically by stop time
	sort.Slice(gaps, func(i, j int) bool {
		return gaps[i].stop.Before(gaps[j].stop)
	})
	return gaps
}

// trimSegmentsByIdleGaps removes idle periods from checkout segments.
// For each segment, only idle gaps matching the segment's repo are applied.
func trimSegmentsByIdleGaps(segments []sessionSegment, stops []entry.ActivityStopEntry, starts []entry.ActivityStartEntry) []sessionSegment {
	if len(stops) == 0 || len(starts) == 0 {
		return segments
	}

	gaps := buildIdleGaps(stops, starts)
	if len(gaps) == 0 {
		return segments
	}

	var result []sessionSegment
	for _, seg := range segments {
		filtered := filterGapsByRepo(gaps, seg.repo)
		trimmed := applyGapsToSegment(seg, filtered)
		result = append(result, trimmed...)
	}
	return result
}

// filterGapsByRepo returns gaps that apply to the given segment repo.
// Rules:
//   - Gap with empty repo → applies to all segments (backward compat / log deduction)
//   - Segment with empty repo → affected by all gaps (conservative fallback)
//   - Otherwise → gap applies only if repos match
func filterGapsByRepo(gaps []idleGap, segRepo string) []idleGap {
	if segRepo == "" {
		return gaps
	}
	var filtered []idleGap
	for _, g := range gaps {
		if g.repo == "" || g.repo == segRepo {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

// applyGapsToSegment applies all overlapping idle gaps to a single segment,
// potentially splitting it into multiple sub-segments.
func applyGapsToSegment(seg sessionSegment, gaps []idleGap) []sessionSegment {
	current := []sessionSegment{seg}

	for _, gap := range gaps {
		var next []sessionSegment
		for _, s := range current {
			split := splitSegmentByGap(s, gap)
			next = append(next, split...)
		}
		current = next
	}

	return current
}

// splitSegmentByGap handles the intersection of a single segment with a single gap.
// Gap interval is [gap.stop, gap.start) — the time between last activity and resume.
func splitSegmentByGap(seg sessionSegment, gap idleGap) []sessionSegment {
	gapFrom := gap.stop
	gapTo := gap.start

	// No overlap: gap is entirely before or after segment
	if !gapTo.After(seg.from) || !gapFrom.Before(seg.to) {
		return []sessionSegment{seg}
	}

	// Gap fully contains segment
	if !gapFrom.After(seg.from) && !gapTo.Before(seg.to) {
		return nil
	}

	// Gap overlaps start only
	if !gapFrom.After(seg.from) && gapTo.Before(seg.to) {
		return []sessionSegment{{
			branch: seg.branch, repo: seg.repo,
			from: gapTo, to: seg.to, message: seg.message,
		}}
	}

	// Gap overlaps end only
	if gapFrom.After(seg.from) && !gapTo.Before(seg.to) {
		return []sessionSegment{{
			branch: seg.branch, repo: seg.repo,
			from: seg.from, to: gapFrom, message: seg.message,
		}}
	}

	// Gap is strictly inside segment — split into two
	return []sessionSegment{
		{branch: seg.branch, repo: seg.repo, from: seg.from, to: gapFrom, message: seg.message},
		{branch: seg.branch, repo: seg.repo, from: gapTo, to: seg.to, message: seg.message},
	}
}

// sessionSegment represents a sub-block of a checkout session, split by commits.
type sessionSegment struct {
	branch  string
	repo    string
	from    time.Time
	to      time.Time
	message string // commit message, empty for uncommitted trailing segment
}

// buildSyntheticCheckouts creates synthetic checkout entries from commits to
// detect repo switches. When consecutive commits are on different repos, a
// synthetic checkout is placed at the midpoint to split the timeline.
func buildSyntheticCheckouts(commits []entry.CommitEntry) []entry.CheckoutEntry {
	if len(commits) == 0 {
		return nil
	}

	sorted := make([]entry.CommitEntry, len(commits))
	copy(sorted, commits)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	var synthetic []entry.CheckoutEntry
	for i := 1; i < len(sorted); i++ {
		prev := sorted[i-1]
		curr := sorted[i]
		if prev.Repo == "" || curr.Repo == "" {
			continue
		}
		if prev.Repo == curr.Repo {
			continue
		}
		// Different repos — create synthetic checkout at midpoint
		mid := prev.Timestamp.Add(curr.Timestamp.Sub(prev.Timestamp) / 2)
		synthetic = append(synthetic, entry.CheckoutEntry{
			ID:        "synthetic-" + curr.ID,
			Type:      "checkout",
			Timestamp: mid,
			Previous:  prev.Branch,
			Next:      curr.Branch,
			Repo:      curr.Repo,
		})
	}
	return synthetic
}

// buildCheckoutSegments splits checkout sessions by commits to produce
// finer-grained time segments. Each commit creates a segment from the previous
// boundary to the commit timestamp. Time is attributed backwards from the commit
// — work before a commit is attributed to that commit. Trailing time after the
// last commit becomes an unnamed segment (uncommitted work).
//
// Synthetic checkouts are injected from commit-based repo switches so that
// commits in different repos are never orphaned.
//
// When no commits exist within a session, the entire session becomes one segment.
func buildCheckoutSegments(
	checkouts []entry.CheckoutEntry,
	commits []entry.CommitEntry,
	year int, month time.Month, daysInMonth int,
	now time.Time,
) []sessionSegment {
	loc := now.Location()

	// Merge synthetic checkouts from commit-based repo switches.
	// Allocate a new slice to avoid mutating the caller's backing array.
	synthetic := buildSyntheticCheckouts(commits)
	sorted := make([]entry.CheckoutEntry, 0, len(checkouts)+len(synthetic))
	sorted = append(sorted, checkouts...)
	sorted = append(sorted, synthetic...)

	// Sort checkouts chronologically
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	// Deduplicate: skip consecutive checkouts to the same branch+repo
	if len(sorted) > 0 {
		deduped := []entry.CheckoutEntry{sorted[0]}
		for i := 1; i < len(sorted); i++ {
			if cleanBranchName(sorted[i].Next) != cleanBranchName(sorted[i-1].Next) ||
				sorted[i].Repo != sorted[i-1].Repo {
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
			repo:   sorted[lastBeforeIdx].Repo,
			from:   monthStart,
		})
	}

	for _, c := range sorted {
		if c.Timestamp.After(monthStart) && !c.Timestamp.After(monthEnd) {
			pairs = append(pairs, checkoutRange{
				branch: cleanBranchName(c.Next),
				repo:   c.Repo,
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

		// Find commits within this session's time range on the same branch+repo
		var sessionCommits []entry.CommitEntry
		for _, c := range sortedCommits {
			if c.Timestamp.Before(p.from) || !c.Timestamp.Before(p.to) {
				continue
			}
			if cleanBranchName(c.Branch) == p.branch &&
				(c.Repo == "" || p.repo == "" || c.Repo == p.repo) {
				sessionCommits = append(sessionCommits, c)
			}
		}

		if len(sessionCommits) == 0 {
			// No commits — single segment for the whole session
			segments = append(segments, sessionSegment{
				branch: p.branch,
				repo:   p.repo,
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
				repo := c.Repo
				if repo == "" {
					repo = p.repo
				}
				segments = append(segments, sessionSegment{
					branch:  p.branch,
					repo:    repo,
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
				repo:   p.repo,
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
