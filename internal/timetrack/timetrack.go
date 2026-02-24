package timetrack

import (
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// TaskRow holds aggregated time data for a single task (branch or manual log).
type TaskRow struct {
	Name         string
	TotalMinutes int
	Days         map[int]int // day-of-month -> minutes
}

// ReportData holds the complete report for a given month.
type ReportData struct {
	Year        int
	Month       time.Month
	DaysInMonth int
	Rows        []TaskRow
}

// BuildReport computes a monthly time report from checkout entries, manual log
// entries, and expanded day schedules. Time is attributed to branches based on
// checkout ranges clipped to schedule windows.
func BuildReport(
	checkouts []entry.CheckoutEntry,
	logs []entry.Entry,
	daySchedules []schedule.DaySchedule,
	year int, month time.Month,
	now time.Time,
) ReportData {
	daysInMonth := daysIn(year, month)

	// 1. Build schedule lookup: day -> windows and day -> total scheduled minutes
	scheduleWindows := make(map[int][]schedule.TimeWindow)
	scheduledMins := make(map[int]int)
	for _, ds := range daySchedules {
		y, m, d := ds.Date.Date()
		if y == year && m == month {
			scheduleWindows[d] = ds.Windows
			total := 0
			for _, w := range ds.Windows {
				total += windowMinutes(w)
			}
			scheduledMins[d] = total
		}
	}

	// 2. Bucket manual logs by (taskKey, day) and track logMinutesByDay
	logBucket := make(map[string]map[int]int)    // taskKey -> day -> minutes
	logMinutesByDay := make(map[int]int)          // day -> total log minutes
	for _, l := range logs {
		if l.Start.Year() != year || l.Start.Month() != month {
			continue
		}
		key := logTaskKey(l)
		day := l.Start.Day()
		if logBucket[key] == nil {
			logBucket[key] = make(map[int]int)
		}
		logBucket[key][day] += l.Minutes
		logMinutesByDay[day] += l.Minutes
	}

	// 3. Sort checkouts by timestamp; find relevant ones for this month
	sorted := make([]entry.CheckoutEntry, len(checkouts))
	copy(sorted, checkouts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(year, month, daysInMonth, 23, 59, 59, 0, time.UTC)

	// 4. Build checkout time buckets: branch -> day -> minutes
	checkoutBucket := make(map[string]map[int]int)

	// Find the last checkout before or at month start to know active branch
	var pairs []checkoutRange
	lastBeforeIdx := -1
	for i, c := range sorted {
		if !c.Timestamp.After(monthStart) {
			lastBeforeIdx = i
		}
	}

	// Build ranges from consecutive checkouts
	if lastBeforeIdx >= 0 {
		// Branch active from month start
		pairs = append(pairs, checkoutRange{
			branch: sorted[lastBeforeIdx].Next,
			from:   monthStart,
		})
	}

	for _, c := range sorted {
		if c.Timestamp.After(monthStart) && !c.Timestamp.After(monthEnd) {
			pairs = append(pairs, checkoutRange{
				branch: c.Next,
				from:   c.Timestamp,
			})
		}
	}

	// Close each range with the next one's start, last one extends to month end or now
	lastEnd := monthEnd.Add(time.Second) // inclusive end
	if now.Before(lastEnd) {
		lastEnd = now
	}
	for i := range pairs {
		if i+1 < len(pairs) {
			pairs[i].to = pairs[i+1].from
		} else {
			pairs[i].to = lastEnd
		}
	}

	// 5. For each range, compute overlap with schedule windows per day
	for _, p := range pairs {
		if p.branch == "" {
			continue
		}
		if checkoutBucket[p.branch] == nil {
			checkoutBucket[p.branch] = make(map[int]int)
		}
		for day := 1; day <= daysInMonth; day++ {
			windows, ok := scheduleWindows[day]
			if !ok {
				continue
			}
			mins := overlapMinutes(p.from, p.to, year, month, day, windows)
			if mins > 0 {
				checkoutBucket[p.branch][day] += mins
			}
		}
	}

	// 6. Deduct: if checkoutMins + logMins > scheduledMins on a day,
	//    reduce checkout time proportionally
	for day := 1; day <= daysInMonth; day++ {
		maxMins := scheduledMins[day]
		if maxMins <= 0 {
			continue
		}
		logMins := logMinutesByDay[day]
		availableForCheckouts := maxMins - logMins
		if availableForCheckouts < 0 {
			availableForCheckouts = 0
		}

		// Sum all checkout minutes for this day
		totalCheckoutMins := 0
		for _, dayMap := range checkoutBucket {
			totalCheckoutMins += dayMap[day]
		}

		if totalCheckoutMins > availableForCheckouts && totalCheckoutMins > 0 {
			ratio := float64(availableForCheckouts) / float64(totalCheckoutMins)
			for branch, dayMap := range checkoutBucket {
				dayMap[day] = int(float64(dayMap[day]) * ratio)
				checkoutBucket[branch] = dayMap
			}
		}
	}

	// 7. Merge checkout + log buckets into TaskRows
	rowMap := make(map[string]*TaskRow)
	for branch, dayMap := range checkoutBucket {
		row := &TaskRow{Name: branch, Days: make(map[int]int)}
		for day, mins := range dayMap {
			if mins > 0 {
				row.Days[day] = mins
				row.TotalMinutes += mins
			}
		}
		if row.TotalMinutes > 0 {
			rowMap[branch] = row
		}
	}
	for key, dayMap := range logBucket {
		row, ok := rowMap[key]
		if !ok {
			row = &TaskRow{Name: key, Days: make(map[int]int)}
			rowMap[key] = row
		}
		for day, mins := range dayMap {
			row.Days[day] += mins
			row.TotalMinutes += mins
		}
	}

	// Sort by total descending
	rows := make([]TaskRow, 0, len(rowMap))
	for _, row := range rowMap {
		rows = append(rows, *row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].TotalMinutes != rows[j].TotalMinutes {
			return rows[i].TotalMinutes > rows[j].TotalMinutes
		}
		return rows[i].Name < rows[j].Name
	})

	return ReportData{
		Year:        year,
		Month:       month,
		DaysInMonth: daysInMonth,
		Rows:        rows,
	}
}

type checkoutRange struct {
	branch string
	from   time.Time
	to     time.Time
}

// logTaskKey returns the grouping key for a manual log entry.
func logTaskKey(e entry.Entry) string {
	if e.Task != "" {
		return e.Task
	}
	return e.Message
}

// daysIn returns the number of days in the given month.
func daysIn(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

// windowMinutes returns the duration of a schedule window in minutes.
func windowMinutes(w schedule.TimeWindow) int {
	from := w.From.Hour*60 + w.From.Minute
	to := w.To.Hour*60 + w.To.Minute
	return to - from
}

// overlapMinutes computes how many minutes of the checkout range [from, to)
// overlap with the given schedule windows on a specific day.
func overlapMinutes(from, to time.Time, year int, month time.Month, day int, windows []schedule.TimeWindow) int {
	total := 0
	for _, w := range windows {
		wStart := time.Date(year, month, day, w.From.Hour, w.From.Minute, 0, 0, time.UTC)
		wEnd := time.Date(year, month, day, w.To.Hour, w.To.Minute, 0, 0, time.UTC)

		// Overlap: max(from, wStart) to min(to, wEnd)
		overlapStart := from
		if wStart.After(overlapStart) {
			overlapStart = wStart
		}
		overlapEnd := to
		if wEnd.Before(overlapEnd) {
			overlapEnd = wEnd
		}

		if overlapEnd.After(overlapStart) {
			total += int(overlapEnd.Sub(overlapStart).Minutes())
		}
	}
	return total
}
