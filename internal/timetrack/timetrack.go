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
// checkout ranges clipped to schedule windows. Days listed in generatedDays
// (format "2006-01-02") are excluded from checkout attribution — they have
// already been materialized as editable log entries by the generate command.
func BuildReport(
	checkouts []entry.CheckoutEntry,
	logs []entry.Entry,
	daySchedules []schedule.DaySchedule,
	year int, month time.Month,
	now time.Time,
	generatedDays []string,
) ReportData {
	daysInMonth := daysIn(year, month)

	generatedSet := make(map[int]bool, len(generatedDays))
	for _, ds := range generatedDays {
		t, err := time.Parse("2006-01-02", ds)
		if err != nil {
			continue
		}
		if t.Year() == year && t.Month() == month {
			generatedSet[t.Day()] = true
		}
	}

	scheduleWindows, scheduledMins := buildScheduleLookup(daySchedules, year, month)
	logBucket, logMinsByDay := buildLogBucket(logs, year, month)
	checkoutBucket := buildCheckoutBucket(checkouts, year, month, daysInMonth, scheduleWindows, now)

	// Zero out checkout attribution for generated days
	for day := range generatedSet {
		for branch := range checkoutBucket {
			delete(checkoutBucket[branch], day)
		}
	}

	deductScheduleOverrun(checkoutBucket, logMinsByDay, scheduledMins, daysInMonth, generatedSet)
	rows := mergeAndSortRows(checkoutBucket, logBucket)

	return ReportData{
		Year:        year,
		Month:       month,
		DaysInMonth: daysInMonth,
		Rows:        rows,
	}
}

// BuildCheckoutAttribution computes raw checkout time per branch per day
// (before schedule deduction). Used by the generate command to materialize
// checkout time into editable log entries.
func BuildCheckoutAttribution(
	checkouts []entry.CheckoutEntry,
	daySchedules []schedule.DaySchedule,
	year int, month time.Month,
	now time.Time,
) map[string]map[int]int {
	daysInMonth := daysIn(year, month)
	scheduleWindows, _ := buildScheduleLookup(daySchedules, year, month)
	return buildCheckoutBucket(checkouts, year, month, daysInMonth, scheduleWindows, now)
}

// buildScheduleLookup builds day -> windows and day -> total scheduled minutes maps.
func buildScheduleLookup(daySchedules []schedule.DaySchedule, year int, month time.Month) (map[int][]schedule.TimeWindow, map[int]int) {
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
	return scheduleWindows, scheduledMins
}

// buildLogBucket buckets manual log entries by (taskKey, day) and totals log minutes per day.
func buildLogBucket(logs []entry.Entry, year int, month time.Month) (map[string]map[int]int, map[int]int) {
	logBucket := make(map[string]map[int]int)
	logMinsByDay := make(map[int]int)
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
		logMinsByDay[day] += l.Minutes
	}
	return logBucket, logMinsByDay
}

// buildCheckoutBucket computes per-branch, per-day minutes from checkout entries
// clipped to schedule windows. Schedule window times are interpreted in the
// timezone of `now` (the user's local timezone).
func buildCheckoutBucket(
	checkouts []entry.CheckoutEntry,
	year int, month time.Month, daysInMonth int,
	scheduleWindows map[int][]schedule.TimeWindow,
	now time.Time,
) map[string]map[int]int {
	loc := now.Location()
	sorted := make([]entry.CheckoutEntry, len(checkouts))
	copy(sorted, checkouts)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Timestamp.Before(sorted[j].Timestamp)
	})

	monthStart := time.Date(year, month, 1, 0, 0, 0, 0, loc)
	monthEnd := time.Date(year, month, daysInMonth, 23, 59, 59, 0, loc)

	var pairs []checkoutRange
	lastBeforeIdx := -1
	for i, c := range sorted {
		if !c.Timestamp.After(monthStart) {
			lastBeforeIdx = i
		}
	}

	if lastBeforeIdx >= 0 {
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

	lastEnd := monthEnd.Add(time.Second)
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

	checkoutBucket := make(map[string]map[int]int)
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
			mins := overlapMinutes(p.from, p.to, year, month, day, windows, loc)
			if mins > 0 {
				checkoutBucket[p.branch][day] += mins
			}
		}
	}

	return checkoutBucket
}

// deductScheduleOverrun reduces checkout minutes proportionally when
// checkoutMins + logMins exceed the scheduled minutes for a day.
func deductScheduleOverrun(checkoutBucket map[string]map[int]int, logMinsByDay, scheduledMins map[int]int, daysInMonth int, generatedDays map[int]bool) {
	for day := 1; day <= daysInMonth; day++ {
		if generatedDays[day] {
			continue
		}
		maxMins := scheduledMins[day]
		if maxMins <= 0 {
			continue
		}
		logMins := logMinsByDay[day]
		availableForCheckouts := maxMins - logMins
		if availableForCheckouts < 0 {
			availableForCheckouts = 0
		}

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
}

// mergeAndSortRows merges checkout and log buckets into sorted TaskRows.
func mergeAndSortRows(checkoutBucket map[string]map[int]int, logBucket map[string]map[int]int) []TaskRow {
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

	return rows
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
// overlap with the given schedule windows on a specific day. Schedule window
// times are interpreted in the given location (the user's local timezone).
func overlapMinutes(from, to time.Time, year int, month time.Month, day int, windows []schedule.TimeWindow, loc *time.Location) int {
	total := 0
	for _, w := range windows {
		wStart := time.Date(year, month, day, w.From.Hour, w.From.Minute, 0, 0, loc)
		wEnd := time.Date(year, month, day, w.To.Hour, w.To.Minute, 0, 0, loc)

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
