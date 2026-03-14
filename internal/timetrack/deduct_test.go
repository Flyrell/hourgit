package timetrack

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
)

// splitWorkday returns a DaySchedule with a split schedule: 09:00-11:00, 12:00-17:00.
func splitWorkday(year int, month time.Month, day int) schedule.DaySchedule {
	return schedule.DaySchedule{
		Date: time.Date(year, month, day, 0, 0, 0, 0, time.UTC),
		Windows: []schedule.TimeWindow{
			{From: schedule.TimeOfDay{Hour: 9, Minute: 0}, To: schedule.TimeOfDay{Hour: 11, Minute: 0}},
			{From: schedule.TimeOfDay{Hour: 12, Minute: 0}, To: schedule.TimeOfDay{Hour: 17, Minute: 0}},
		},
	}
}

// Test Case 1: Checkout "A" at 02:00, checkout "B" at 11:00, manual log 10:00-11:00
func TestDeductLogOverlaps_LogBetweenCheckouts_Default(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 2, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Previous: "A", Next: "B"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting", Task: "meeting"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	rowB := findRow(report, "B")
	rowLog := findRow(report, "meeting")

	assert.NotNil(t, rowA)
	assert.NotNil(t, rowB)
	assert.NotNil(t, rowLog)

	// A: 09:00-10:00 = 60min (log carves 10:00-11:00)
	assert.Equal(t, 60, rowA.Days[2])
	// B: 11:00-17:00 = 360min
	assert.Equal(t, 360, rowB.Days[2])
	// Log: 60min
	assert.Equal(t, 60, rowLog.Days[2])
}

func TestDeductLogOverlaps_LogBetweenCheckouts_Split(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{splitWorkday(year, month, 2)} // 9-11, 12-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 2, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Previous: "A", Next: "B"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting", Task: "meeting"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	rowB := findRow(report, "B")

	assert.NotNil(t, rowA)
	assert.NotNil(t, rowB)

	// A: 09:00-10:00 = 60min (log carves 10:00-11:00, schedule only goes to 11:00)
	assert.Equal(t, 60, rowA.Days[2])
	// B: 12:00-17:00 = 300min (split schedule, second window)
	assert.Equal(t, 300, rowB.Days[2])
}

// Test Case 2: Log spanning a lunch gap
func TestDeductLogOverlaps_LogSpanningGap_Default(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 2, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Previous: "A", Next: "B"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "meeting", Task: "meeting"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	rowB := findRow(report, "B")

	assert.NotNil(t, rowA)
	assert.NotNil(t, rowB)

	// A: 09:00-10:00 = 60min (log carves 10:00-12:00, which also takes from B's start)
	assert.Equal(t, 60, rowA.Days[2])
	// B: checkout at 11:00, but log covers 10:00-12:00, so B effective from 12:00-17:00 = 300min
	assert.Equal(t, 300, rowB.Days[2])
}

// Test Case 3: Log at start of checkout
func TestDeductLogOverlaps_LogAtCheckoutStart_Default(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 17, 0, 0, 0, time.UTC), Previous: "A", Next: "B"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 180, Message: "research", Task: "research"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	assert.NotNil(t, rowA)

	// A: checkout 10:00-17:00, log carves 10:00-13:00, so A gets 13:00-17:00 = 240min
	assert.Equal(t, 240, rowA.Days[2])
}

// Edge case: Log on different day only affects that day
func TestDeductLogOverlaps_LogOnDifferentDay(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2), workday(year, month, 3)}

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 3, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting", Task: "meeting"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	assert.NotNil(t, rowA)

	// Day 2: unaffected by log on day 3
	assert.Equal(t, 480, rowA.Days[2])
	// Day 3: log carves 10:00-11:00, A gets 09:00-10:00 + 11:00-17:00 = 420min
	assert.Equal(t, 420, rowA.Days[3])
}

// Edge case: Log fully contains a segment
func TestDeductLogOverlaps_LogContainsSegment(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)}

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
		{ID: "c2", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC), Previous: "A", Next: "B"},
	}
	// Log covers entire A segment (10:00-11:00)
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC), Minutes: 120, Message: "meeting", Task: "meeting"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	// A: checkout 10:00-11:00, log covers 09:00-11:00, so A is fully removed
	assert.Nil(t, rowA)
}

// Edge case: Multiple logs on same day
func TestDeductLogOverlaps_MultipleLogs(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting1", Task: "meeting1"},
		{ID: "l2", Start: time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting2", Task: "meeting2"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	assert.NotNil(t, rowA)

	// A: 480 - 60 - 60 = 360min (two logs carved out)
	assert.Equal(t, 360, rowA.Days[2])
}

// Edge case: checkout-generated logs should NOT deduct
func TestDeductLogOverlaps_CheckoutGeneratedSkipped(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)}

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 120, Message: "A", Task: "A", Source: "checkout-generated"},
	}

	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil)

	rowA := findRow(report, "A")
	assert.NotNil(t, rowA)

	// checkout-generated log should not reduce checkout time — total = 480 + 120
	assert.Equal(t, 480+120, rowA.TotalMinutes)
}

// Activity start/stop + log interaction
func TestDeductLogOverlaps_WithIdleGaps(t *testing.T) {
	year, month := 2025, time.January
	days := []schedule.DaySchedule{workday(year, month, 2)} // 9-17

	checkouts := []entry.CheckoutEntry{
		{ID: "c1", Timestamp: time.Date(2024, 12, 31, 10, 0, 0, 0, time.UTC), Previous: "main", Next: "A"},
	}
	logs := []entry.Entry{
		{ID: "l1", Start: time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), Minutes: 60, Message: "meeting", Task: "meeting"},
	}

	// Idle gap from 10:00-11:00
	stops := []entry.ActivityStopEntry{
		{ID: "s1", Timestamp: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)},
	}
	starts := []entry.ActivityStartEntry{
		{ID: "a1", Timestamp: time.Date(2025, 1, 2, 11, 0, 0, 0, time.UTC)},
	}

	activity := ActivityEntries{Stops: stops, Starts: starts}
	report := BuildReport(checkouts, logs, nil, days, year, month, afterMonth(year, month), nil, activity)

	rowA := findRow(report, "A")
	assert.NotNil(t, rowA)

	// Schedule: 9-17 = 480min
	// Idle gap carves 10:00-11:00 = -60min → 420min
	// Log carves 14:00-15:00 = -60min → 360min
	assert.Equal(t, 360, rowA.Days[2])
}
