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

func TestRenderTableOutput(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		Rows: []timetrack.TaskRow{
			{
				Name:         "feature-x",
				TotalMinutes: 120,
				Days:         map[int]int{2: 60, 3: 60},
			},
			{
				Name:         "bugfix",
				TotalMinutes: 30,
				Days:         map[int]int{2: 30},
			},
		},
	}

	result := renderTable(data, 0, 5, false)

	// Header should contain day numbers
	assert.Contains(t, result, "Task")
	assert.Contains(t, result, "1")
	assert.Contains(t, result, "2")
	assert.Contains(t, result, "3")

	// Rows should contain task names and times
	assert.Contains(t, result, "feature-x")
	assert.Contains(t, result, "bugfix")
	assert.Contains(t, result, "2h")
	assert.Contains(t, result, "30m")

	// Total row
	assert.Contains(t, result, "Total")
}

func TestRenderTableWithFooter(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		Rows:        nil,
	}

	result := renderTable(data, 0, 5, true)
	assert.Contains(t, result, "February 2026")
	assert.Contains(t, result, "scroll")
	assert.Contains(t, result, "quit")
}

func TestRenderTableNoFooter(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		Rows:        nil,
	}

	result := renderTable(data, 0, 5, false)
	assert.NotContains(t, result, "scroll")
	assert.NotContains(t, result, "quit")
}

func TestPrintStaticTable(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		Rows: []timetrack.TaskRow{
			{
				Name:         "work",
				TotalMinutes: 480,
				Days:         map[int]int{1: 480},
			},
		},
	}

	var buf bytes.Buffer
	err := printStaticTable(&buf, data)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "work")
	assert.Contains(t, output, "8h")
	// Static table should show all 28 days
	assert.Contains(t, output, "28")
	// No footer in static mode
	assert.NotContains(t, output, "scroll")
}

func TestVisibleDays(t *testing.T) {
	t.Run("wide terminal shows all days", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.ReportData{DaysInMonth: 28},
			termWidth: 500,
		}
		assert.Equal(t, 28, m.visibleDays())
	})

	t.Run("narrow terminal limits days", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.ReportData{DaysInMonth: 31},
			termWidth: 80,
		}
		days := m.visibleDays()
		assert.Greater(t, days, 0)
		assert.LessOrEqual(t, days, 31)
	})

	t.Run("very narrow terminal shows at least 1 day", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.ReportData{DaysInMonth: 31},
			termWidth: 10,
		}
		assert.Equal(t, 1, m.visibleDays())
	})
}

func TestMaxScroll(t *testing.T) {
	t.Run("wide terminal no scroll needed", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.ReportData{DaysInMonth: 28},
			termWidth: 500,
		}
		assert.Equal(t, 0, m.maxScroll())
	})

	t.Run("narrow terminal allows scroll", func(t *testing.T) {
		m := reportModel{
			data:      timetrack.ReportData{DaysInMonth: 31},
			termWidth: 80,
		}
		maxScroll := m.maxScroll()
		assert.Greater(t, maxScroll, 0)
		assert.Equal(t, 31-m.visibleDays(), maxScroll)
	})
}

func TestRenderTableScroll(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2026,
		Month:       time.February,
		DaysInMonth: 28,
		Rows: []timetrack.TaskRow{
			{
				Name:         "task",
				TotalMinutes: 60,
				Days:         map[int]int{15: 60},
			},
		},
	}

	// Scroll to day 14 (0-indexed), show 3 days -> should show days 15, 16, 17
	result := renderTable(data, 14, 3, false)
	lines := strings.Split(result, "\n")

	// Header should show days 15, 16, 17
	header := lines[0]
	assert.Contains(t, header, "15")
	assert.Contains(t, header, "16")
	assert.Contains(t, header, "17")
}
