package cli

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/stretchr/testify/assert"
)

func buildDetailPanelModel(entries []timetrack.CellEntry, selectedIdx int) reportModel {
	row := timetrack.DetailedTaskRow{
		Name: "dev",
		Days: map[int]*timetrack.CellData{
			1: {Entries: entries, TotalMinutes: 0},
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

	assert.Contains(t, out, "09:00")
	assert.Contains(t, out, "12:00")
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
	assert.Contains(t, out, "14:00")
	assert.Contains(t, out, "15:00")
}

func TestRenderDetailPanel_SelectedMarker(t *testing.T) {
	entries := []timetrack.CellEntry{
		{Start: time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC), Minutes: 60, Message: "first"},
		{Start: time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC), Minutes: 60, Message: "second"},
	}

	m := buildDetailPanelModel(entries, 1)
	out := m.renderDetailPanel()

	assert.Contains(t, out, "> ")
	assert.Contains(t, out, "first")
	assert.Contains(t, out, "second")
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

	assert.Contains(t, out, "23:30")
	assert.Contains(t, out, "00:30")
}
