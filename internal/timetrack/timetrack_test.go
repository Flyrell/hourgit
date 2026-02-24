package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
)

// workday returns a DaySchedule for the given day with a 9am-5pm window.
func workday(year int, month time.Month, day int) schedule.DaySchedule {
	return schedule.DaySchedule{
		Date: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		Windows: []schedule.TimeWindow{
			{
				From: schedule.TimeOfDay{Hour: 9, Minute: 0},
				To:   schedule.TimeOfDay{Hour: 17, Minute: 0},
			},
		},
	}
}

// afterMonth returns a time after the end of the given month, for use as the
// `now` parameter in BuildReport tests where capping is not being tested.
func afterMonth(year int, month time.Month) time.Time {
	return time.Date(year, month+1, 1, 12, 0, 0, 0, time.UTC)
}

func TestBuildReport_SingleCheckoutFullMonth(t *testing.T) {
	year, month := 2025, time.January

	// Checkout before month start: branch "feature-x" active from day 1
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	// Workdays: Jan 2 (Thu), Jan 3 (Fri), Jan 6-10, Jan 13-17, Jan 20-24, Jan 27-31
	var days []schedule.DaySchedule
	for d := 1; d <= 31; d++ {
		dt := time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
		wd := dt.Weekday()
		if wd >= time.Monday && wd <= time.Friday {
			days = append(days, workday(year, month, d))
		}
	}

	report := BuildReport(checkouts, nil, days, year, month, afterMonth(year, month))

	assert.Equal(t, 1, len(report.Rows))
	assert.Equal(t, "feature-x", report.Rows[0].Name)

	// Each workday = 480 min. Count workdays in Jan 2025.
	workdays := 0
	for d := 1; d <= 31; d++ {
		dt := time.Date(year, month, d, 0, 0, 0, 0, time.UTC)
		wd := dt.Weekday()
		if wd >= time.Monday && wd <= time.Friday {
			workdays++
		}
	}
	assert.Equal(t, workdays*480, report.Rows[0].TotalMinutes)
}

func TestBuildReport_TwoCheckoutsSplitDay(t *testing.T) {
	year, month := 2025, time.January

	days := []schedule.DaySchedule{workday(year, month, 2)} // Thu Jan 2: 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-a"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC), Previous: "feature-a", Next: "feature-b"},
	}

	report := BuildReport(checkouts, nil, days, year, month, afterMonth(year, month))

	assert.Equal(t, 2, len(report.Rows))

	rowA := findRow(report, "feature-a")
	rowB := findRow(report, "feature-b")
	assert.NotNil(t, rowA)
	assert.NotNil(t, rowB)
	assert.Equal(t, 240, rowA.Days[2]) // 9:00-13:00 = 4h
	assert.Equal(t, 240, rowB.Days[2]) // 13:00-17:00 = 4h
}

func TestBuildReport_ManualLogDeduction(t *testing.T) {
	year, month := 2025, time.January

	days := []schedule.DaySchedule{workday(year, month, 2)} // 480 min total

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "research", Task: "research"},
	}

	report := BuildReport(checkouts, logs, days, year, month, afterMonth(year, month))

	rowCheckout := findRow(report, "feature-x")
	rowLog := findRow(report, "research")

	assert.NotNil(t, rowCheckout)
	assert.NotNil(t, rowLog)

	// Log takes 120 min, checkout should get 480-120=360 min
	assert.Equal(t, 360, rowCheckout.Days[2])
	assert.Equal(t, 120, rowLog.Days[2])
}

func TestBuildReport_LogTaskKeyFallback(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "did research", Task: ""},
		{ID: "l2", Start: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Minutes: 60, Message: "did research", Task: ""},
	}

	report := BuildReport(nil, logs, days, year, month, afterMonth(year, month))

	assert.Equal(t, 1, len(report.Rows))
	assert.Equal(t, "did research", report.Rows[0].Name)
	assert.Equal(t, 120, report.Rows[0].TotalMinutes)
}

func TestBuildReport_NoCheckoutsLogsOnly(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 180, Message: "analysis", Task: "analysis"},
	}

	report := BuildReport(nil, logs, days, year, month, afterMonth(year, month))

	assert.Equal(t, 1, len(report.Rows))
	assert.Equal(t, "analysis", report.Rows[0].Name)
	assert.Equal(t, 180, report.Rows[0].TotalMinutes)
}

func TestBuildReport_CheckoutBeforeMonthStart(t *testing.T) {
	year, month := 2025, time.February
	days := []schedule.DaySchedule{workday(year, month, 3)} // Mon Feb 3

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-y"},
	}

	report := BuildReport(checkouts, nil, days, year, month, afterMonth(year, month))

	assert.Equal(t, 1, len(report.Rows))
	assert.Equal(t, "feature-y", report.Rows[0].Name)
	assert.Equal(t, 480, report.Rows[0].Days[3])
}

func TestBuildReport_EmptyMonth(t *testing.T) {
	year, month := 2025, time.January

	report := BuildReport(nil, nil, nil, year, month, afterMonth(year, month))

	assert.Equal(t, 0, len(report.Rows))
	assert.Equal(t, 31, report.DaysInMonth)
}

func TestBuildReport_SortedByTotalDescending(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{
		workday(year, month, 2),
		workday(year, month, 3),
	}

	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "small", Task: "small"},
		{ID: "l2", Start: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Minutes: 120, Message: "big", Task: "big"},
		{ID: "l3", Start: time.Date(2025, 1, 3, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "big", Task: "big"},
	}

	report := BuildReport(nil, logs, days, year, month, afterMonth(year, month))

	assert.Equal(t, 2, len(report.Rows))
	assert.Equal(t, "big", report.Rows[0].Name)
	assert.Equal(t, "small", report.Rows[1].Name)
}

func TestBuildReport_LastCheckoutCappedAtNow(t *testing.T) {
	year, month := 2025, time.January

	// Schedule: Jan 2 (Thu) and Jan 3 (Fri), both 9-17
	days := []schedule.DaySchedule{
		workday(year, month, 2),
		workday(year, month, 3),
	}

	// Checkout on Jan 2 at 9am
	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Previous: "main", Next: "feature-x"},
	}

	// "now" is Jan 2 at 13:00 — should only get 4h (9-13), not 8h (9-17)
	now := time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC)
	report := BuildReport(checkouts, nil, days, year, month, now)

	assert.Equal(t, 1, len(report.Rows))
	assert.Equal(t, "feature-x", report.Rows[0].Name)
	assert.Equal(t, 240, report.Rows[0].Days[2])  // 9:00-13:00 = 4h
	assert.Equal(t, 0, report.Rows[0].Days[3])     // Jan 3 should have no time (now is before it)
	assert.Equal(t, 240, report.Rows[0].TotalMinutes)
}

func findRow(report ReportData, name string) *TaskRow {
	for i := range report.Rows {
		if report.Rows[i].Name == name {
			return &report.Rows[i]
		}
	}
	return nil
}
