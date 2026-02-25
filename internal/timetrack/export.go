package timetrack

import (
	"sort"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
)

// ExportEntry represents a single time entry for PDF export.
type ExportEntry struct {
	Start   time.Time
	Minutes int
	Message string
}

// ExportTaskGroup groups entries under a task name with a subtotal.
type ExportTaskGroup struct {
	Task         string
	Entries      []ExportEntry
	TotalMinutes int
}

// ExportDay holds all task groups for a single day.
type ExportDay struct {
	Date         time.Time
	Groups       []ExportTaskGroup
	TotalMinutes int
}

// ExportData holds the complete export for a given month.
type ExportData struct {
	ProjectName  string
	Year         int
	Month        time.Month
	Days         []ExportDay
	TotalMinutes int
}

// BuildExportData builds detailed export data preserving individual entries,
// grouped by day and task. Checkout attribution on non-generated days produces
// one synthetic entry per branch-day.
func BuildExportData(
	checkouts []entry.CheckoutEntry,
	logs []entry.Entry,
	daySchedules []schedule.DaySchedule,
	year int, month time.Month,
	now time.Time,
	generatedDays []string,
	projectName string,
) ExportData {
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
	checkoutBucket := buildCheckoutBucket(checkouts, year, month, daysInMonth, scheduleWindows, now)

	// Zero out checkout attribution for generated days
	for day := range generatedSet {
		for branch := range checkoutBucket {
			delete(checkoutBucket[branch], day)
		}
	}

	// Build log minutes by day for deduction
	logMinsByDay := make(map[int]int)
	for _, l := range logs {
		if l.Start.Year() == year && l.Start.Month() == month {
			logMinsByDay[l.Start.Day()] += l.Minutes
		}
	}

	deductScheduleOverrun(checkoutBucket, logMinsByDay, scheduledMins, daysInMonth, generatedSet)

	// Build per-day data: day -> task -> []ExportEntry
	type dayTask struct {
		task    string
		entries []ExportEntry
	}
	dayGroups := make(map[int]map[string]*dayTask)

	// Add log entries
	for _, l := range logs {
		if l.Start.Year() != year || l.Start.Month() != month {
			continue
		}
		day := l.Start.Day()
		if dayGroups[day] == nil {
			dayGroups[day] = make(map[string]*dayTask)
		}
		key := logTaskKey(l)
		dt := dayGroups[day][key]
		if dt == nil {
			dt = &dayTask{task: key}
			dayGroups[day][key] = dt
		}
		dt.entries = append(dt.entries, ExportEntry{
			Start:   l.Start,
			Minutes: l.Minutes,
			Message: l.Message,
		})
	}

	// Add checkout attribution as synthetic entries
	for branch, dayMap := range checkoutBucket {
		cleanedBranch := cleanBranchName(branch)
		for day, mins := range dayMap {
			if mins <= 0 {
				continue
			}
			if dayGroups[day] == nil {
				dayGroups[day] = make(map[string]*dayTask)
			}
			dt := dayGroups[day][cleanedBranch]
			if dt == nil {
				dt = &dayTask{task: cleanedBranch}
				dayGroups[day][cleanedBranch] = dt
			}
			dt.entries = append(dt.entries, ExportEntry{
				Start:   time.Date(year, month, day, 9, 0, 0, 0, time.UTC),
				Minutes: mins,
				Message: cleanedBranch,
			})
		}
	}

	// Assemble ExportDays
	var days []ExportDay
	for day := 1; day <= daysInMonth; day++ {
		tasks, ok := dayGroups[day]
		if !ok {
			continue
		}

		var groups []ExportTaskGroup
		for _, dt := range tasks {
			totalMins := 0
			for _, e := range dt.entries {
				totalMins += e.Minutes
			}
			if totalMins <= 0 {
				continue
			}
			// Sort entries by start time
			sort.Slice(dt.entries, func(i, j int) bool {
				return dt.entries[i].Start.Before(dt.entries[j].Start)
			})
			groups = append(groups, ExportTaskGroup{
				Task:         dt.task,
				Entries:      dt.entries,
				TotalMinutes: totalMins,
			})
		}

		if len(groups) == 0 {
			continue
		}

		// Sort groups alphabetically
		sort.Slice(groups, func(i, j int) bool {
			return groups[i].Task < groups[j].Task
		})

		dayTotal := 0
		for _, g := range groups {
			dayTotal += g.TotalMinutes
		}

		days = append(days, ExportDay{
			Date:         time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
			Groups:       groups,
			TotalMinutes: dayTotal,
		})
	}

	grandTotal := 0
	for _, d := range days {
		grandTotal += d.TotalMinutes
	}

	return ExportData{
		ProjectName:  projectName,
		Year:         year,
		Month:        month,
		Days:         days,
		TotalMinutes: grandTotal,
	}
}
