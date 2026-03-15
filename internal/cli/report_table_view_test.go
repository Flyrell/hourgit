package cli

import (
	"strings"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/stretchr/testify/assert"
)

func buildDetailPanelModel(entries []timetrack.CellEntry, selectedIdx int) reportModel {
	totalMin := 0
	for _, e := range entries {
		totalMin += e.Minutes
	}
	row := timetrack.DetailedTaskRow{
		Name: "dev",
		Days: map[int]*timetrack.CellData{
			1: {Entries: entries, TotalMinutes: totalMin},
		},
	}
	return reportModel{
		data: timetrack.DetailedReportData{
			Year:        2026,
			Month:       time.March,
			DaysInMonth: 31,
			Rows:        []timetrack.DetailedTaskRow{row},
		},
		cursorRow:        0,
		cursorCol:        0,
		selectedEntryIdx: selectedIdx,
	}
}

func TestRenderDetailPanel_Empty(t *testing.T) {
	m := reportModel{
		data: timetrack.DetailedReportData{
			DaysInMonth: 31,
			Rows:        []timetrack.DetailedTaskRow{{Days: map[int]*timetrack.CellData{}}},
		},
		cursorRow: 0,
		cursorCol: 0,
	}
	assert.Empty(t, m.renderDetailPanel())
}

func TestRenderDetailPanel_TimeRange(t *testing.T) {
	entries := []timetrack.CellEntry{
		{
			Start:   time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC),
			Minutes: 180,
			Message: "morning work",
		},
	}
	m := buildDetailPanelModel(entries, 0)
	out := m.renderDetailPanel()

	assert.Contains(t, out, "09:00-12:00")
	assert.Contains(t, out, "3h")
	assert.Contains(t, out, "morning work")
}

func TestRenderDetailPanel_EmptyMessage(t *testing.T) {
	entries := []timetrack.CellEntry{
		{
			Start:   time.Date(2026, 3, 1, 14, 0, 0, 0, time.UTC),
			Minutes: 60,
			Message: "",
		},
	}
	m := buildDetailPanelModel(entries, 0)
	out := m.renderDetailPanel()

	assert.Contains(t, out, "(no message)")
	assert.Contains(t, out, "14:00-15:00")
}

func TestRenderDetailPanel_SelectedMarker(t *testing.T) {
	entries := []timetrack.CellEntry{
		{Start: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC), Minutes: 60, Message: "first"},
		{Start: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "second"},
	}

	m := buildDetailPanelModel(entries, 1)
	out := m.renderDetailPanel()

	lines := strings.Split(out, "\n")
	var firstLine, secondLine string
	for _, l := range lines {
		if strings.Contains(l, "first") {
			firstLine = l
		}
		if strings.Contains(l, "second") {
			secondLine = l
		}
	}
	assert.NotEmpty(t, firstLine, "should find line containing 'first'")
	assert.NotEmpty(t, secondLine, "should find line containing 'second'")
	assert.NotContains(t, firstLine, "> ", "first entry should not have selection marker")
	assert.Contains(t, secondLine, "> ", "second entry should have selection marker")
	assert.Contains(t, out, "09:00-10:00")
	assert.Contains(t, out, "10:00-11:00")
}

func TestRenderDetailPanel_MidnightCrossing(t *testing.T) {
	entries := []timetrack.CellEntry{
		{
			Start:   time.Date(2026, 3, 1, 23, 30, 0, 0, time.UTC),
			Minutes: 60,
			Message: "late night",
		},
	}
	m := buildDetailPanelModel(entries, 0)
	out := m.renderDetailPanel()

	assert.Contains(t, out, "23:30-00:30")
}

func TestRenderDetailPanel_ZeroStart(t *testing.T) {
	entries := []timetrack.CellEntry{
		{
			Minutes: 90,
			Message: "duration only",
		},
	}
	m := buildDetailPanelModel(entries, 0)
	out := m.renderDetailPanel()

	assert.Contains(t, out, "1h 30m")
	assert.Contains(t, out, "duration only")
	assert.NotContains(t, out, "00:00-")
}
