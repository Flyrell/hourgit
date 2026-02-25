package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/stretchr/testify/assert"
)

func TestPadRight(t *testing.T) {
	t.Run("normal padding", func(t *testing.T) {
		assert.Equal(t, "abc   ", padRight("abc", 6))
	})

	t.Run("exact width", func(t *testing.T) {
		assert.Equal(t, "abcdef", padRight("abcdef", 6))
	})

	t.Run("truncation", func(t *testing.T) {
		assert.Equal(t, "abcdef", padRight("abcdefgh", 6))
	})

	t.Run("empty string", func(t *testing.T) {
		assert.Equal(t, "      ", padRight("", 6))
	})
}

func TestPadCenter(t *testing.T) {
	t.Run("normal centering", func(t *testing.T) {
		result := padCenter("ab", 6)
		assert.Equal(t, 6, len(result))
		assert.Equal(t, "  ab  ", result)
	})

	t.Run("odd width", func(t *testing.T) {
		result := padCenter("ab", 7)
		assert.Equal(t, 7, len(result))
		assert.Contains(t, result, "ab")
	})

	t.Run("exact width", func(t *testing.T) {
		assert.Equal(t, "abcdef", padCenter("abcdef", 6))
	})

	t.Run("truncation", func(t *testing.T) {
		assert.Equal(t, "abcdef", padCenter("abcdefgh", 6))
	})

	t.Run("empty string", func(t *testing.T) {
		result := padCenter("", 6)
		assert.Equal(t, 6, len(result))
	})
}

// makeScheduledDays returns a ScheduledDays map for Feb 2026 weekdays (Mon-Fri).
// Feb 2026: 1=Sun, so weekdays are 2-6, 9-13, 16-20, 23-27.
func makeScheduledDays() map[int]bool {
	days := map[int]bool{}
	for _, d := range []int{2, 3, 4, 5, 6, 9, 10, 11, 12, 13, 16, 17, 18, 19, 20, 23, 24, 25, 26, 27} {
		days[d] = true
	}
	return days
}

func makeDetailedData() timetrack.DetailedReportData {
	return timetrack.DetailedReportData{
		Year:          2026,
		Month:         time.February,
		DaysInMonth:   28,
		From:          time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:            time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		ScheduledDays: makeScheduledDays(),
		Rows: []timetrack.DetailedTaskRow{
			{
				Name:         "feature-x",
				TotalMinutes: 120,
				Days: map[int]*timetrack.CellData{
					2: {
						TotalMinutes: 60,
						Entries: []timetrack.CellEntry{
							{ID: "e100001", Minutes: 60, Message: "work", Task: "feature-x", Source: "manual", Persisted: true},
						},
					},
					3: {
						TotalMinutes: 60,
						Entries: []timetrack.CellEntry{
							{ID: "", Minutes: 60, Message: "feature-x", Task: "feature-x", Source: "checkout", Persisted: false},
						},
					},
				},
			},
			{
				Name:         "bugfix",
				TotalMinutes: 30,
				Days: map[int]*timetrack.CellData{
					2: {
						TotalMinutes: 30,
						Entries: []timetrack.CellEntry{
							{ID: "e200002", Minutes: 30, Message: "fix", Task: "bugfix", Source: "manual", Persisted: true},
						},
					},
				},
			},
		},
	}
}

func TestRenderDetailedTableOutput(t *testing.T) {
	data := makeDetailedData()

	result := renderDetailedTable(data, 0, 0, 5, len(data.Rows), -1, -1, false, "")

	// Title line
	assert.Contains(t, result, "--- February 2026 ---")

	// Header should contain day-of-week labels
	assert.Contains(t, result, "Sun 1")
	assert.Contains(t, result, "Mon 2")
	assert.Contains(t, result, "Tue 3")

	// Rows should contain task names and times
	assert.Contains(t, result, "feature-x")
	assert.Contains(t, result, "bugfix")
	assert.Contains(t, result, "2h")
	assert.Contains(t, result, "30m")

	// Total row
	assert.Contains(t, result, "Total")

	// In-memory entries should be marked with asterisk
	assert.Contains(t, result, "*")

	// Non-scheduled day (Sun 1) should show "x" not "."
	assert.Contains(t, result, "x")
}

func TestRenderDetailedTableWithFooter(t *testing.T) {
	data := timetrack.DetailedReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		From:        time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		Rows:        nil,
	}

	result := renderDetailedTable(data, 0, 0, 5, 0, -1, -1, false, "")
	assert.Contains(t, result, "February 2026")
	assert.Contains(t, result, "navigate")
	assert.Contains(t, result, "quit")
}

func TestRenderDetailedTableSubmittedWarning(t *testing.T) {
	data := timetrack.DetailedReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		From:        time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		Rows:        nil,
	}

	result := renderDetailedTable(data, 0, 0, 5, 0, -1, -1, true, "")
	assert.Contains(t, result, "Previously submitted")
}

func TestPrintStaticDetailedTable(t *testing.T) {
	data := timetrack.DetailedReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		From:        time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		Rows: []timetrack.DetailedTaskRow{
			{
				Name:         "work",
				TotalMinutes: 480,
				Days: map[int]*timetrack.CellData{
					1: {TotalMinutes: 480, Entries: []timetrack.CellEntry{
						{ID: "e100001", Minutes: 480, Message: "work", Persisted: true},
					}},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := printStaticDetailedTable(&buf, data)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "8h")
	// Static table should show all 28 days
	assert.Contains(t, output, "28")
	// No cursor highlight in static mode (-1, -1)
}

func TestVisibleDays(t *testing.T) {
	t.Run("wide terminal shows all days", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.DetailedReportData{DaysInMonth: 28},
			termWidth: 500,
		}
		assert.Equal(t, 28, m.visibleDays())
	})

	t.Run("narrow terminal limits days", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.DetailedReportData{DaysInMonth: 31},
			termWidth: 80,
		}
		days := m.visibleDays()
		assert.Greater(t, days, 0)
		assert.LessOrEqual(t, days, 31)
	})

	t.Run("very narrow terminal shows at least 1 day", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.DetailedReportData{DaysInMonth: 31},
			termWidth: 10,
		}
		assert.Equal(t, 1, m.visibleDays())
	})
}

func TestMaxScrollX(t *testing.T) {
	t.Run("wide terminal no scroll needed", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.DetailedReportData{DaysInMonth: 28},
			termWidth: 500,
		}
		assert.Equal(t, 0, m.maxScrollX())
	})

	t.Run("narrow terminal allows scroll", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.DetailedReportData{DaysInMonth: 31},
			termWidth: 80,
		}
		maxScroll := m.maxScrollX()
		assert.Greater(t, maxScroll, 0)
		assert.Equal(t, 31-m.visibleDays(), maxScroll)
	})
}

func TestRenderDetailedTableScroll(t *testing.T) {
	data := timetrack.DetailedReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		From:        time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		Rows: []timetrack.DetailedTaskRow{
			{
				Name:         "task",
				TotalMinutes: 60,
				Days: map[int]*timetrack.CellData{
					15: {TotalMinutes: 60, Entries: []timetrack.CellEntry{
						{ID: "e100001", Minutes: 60, Persisted: true},
					}},
				},
			},
		},
	}

	// Scroll to day 14 (0-indexed), show 3 days -> should show days 15, 16, 17
	result := renderDetailedTable(data, 14, 0, 3, 1, -1, -1, false, "")
	lines := strings.Split(result, "\n")

	// Header is on line 1 (line 0 is the title)
	header := lines[1]
	assert.Contains(t, header, "15")
	assert.Contains(t, header, "16")
	assert.Contains(t, header, "17")
}

func TestEnsureCursorVisible(t *testing.T) {
	m := reportModel{
		data: timetrack.DetailedReportData{
			DaysInMonth: 28,
			Rows: []timetrack.DetailedTaskRow{
				{Name: "a"}, {Name: "b"}, {Name: "c"}, {Name: "d"}, {Name: "e"},
			},
		},
		termWidth:  80,
		termHeight: 20,
		cursorCol:  20,
		cursorRow:  4,
		scrollX:    0,
		scrollY:    0,
	}

	m = m.ensureCursorVisible()

	// Cursor should be visible horizontally
	assert.LessOrEqual(t, m.scrollX, m.cursorCol)
	assert.Greater(t, m.scrollX+m.visibleDays(), m.cursorCol)
}

func TestRenderDetailedTable_CursorHighlight(t *testing.T) {
	data := makeDetailedData()

	// Cursor on row 0, col 1 (day 2 which has data)
	result := renderDetailedTable(data, 0, 0, 5, len(data.Rows), 0, 1, false, "")

	// The result should contain the table content
	assert.Contains(t, result, "feature-x")
	assert.Contains(t, result, "bugfix")
}

func TestRenderDetailedTable_FooterMsg(t *testing.T) {
	data := timetrack.DetailedReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		From:        time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:          time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
	}

	result := renderDetailedTable(data, 0, 0, 5, 0, -1, -1, false, "Entry saved!")
	assert.Contains(t, result, "Entry saved!")
}

func TestRenderDetailedTable_NonScheduledDaysShowX(t *testing.T) {
	// Only day 2 (Mon) is scheduled; day 1 (Sun) is not
	data := timetrack.DetailedReportData{
		Year:          2026,
		Month:         time.February,
		DaysInMonth:   28,
		From:          time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		To:            time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		ScheduledDays: map[int]bool{2: true},
		Rows: []timetrack.DetailedTaskRow{
			{
				Name:         "task",
				TotalMinutes: 60,
				Days: map[int]*timetrack.CellData{
					2: {TotalMinutes: 60, Entries: []timetrack.CellEntry{
						{ID: "e100001", Minutes: 60, Persisted: true},
					}},
				},
			},
		},
	}

	// Show days 1 and 2
	result := renderDetailedTable(data, 0, 0, 2, 1, -1, -1, false, "")
	lines := strings.Split(result, "\n")

	// Data row: day 1 (non-scheduled) should have "x", day 2 should have time
	dataLine := lines[3] // title(0) + header(1) + separator(2) + data(3)
	assert.Contains(t, dataLine, "x")
	assert.Contains(t, dataLine, "1h")
}

func TestIsWeekend(t *testing.T) {
	// Feb 1, 2026 = Sunday
	assert.True(t, isWeekend(2026, time.February, 1))
	// Feb 7, 2026 = Saturday
	assert.True(t, isWeekend(2026, time.February, 7))
	// Feb 2, 2026 = Monday
	assert.False(t, isWeekend(2026, time.February, 2))
}

func TestDayAbbrev(t *testing.T) {
	assert.Equal(t, "Sun", dayAbbrev(2026, time.February, 1))
	assert.Equal(t, "Mon", dayAbbrev(2026, time.February, 2))
	assert.Equal(t, "Sat", dayAbbrev(2026, time.February, 7))
}
