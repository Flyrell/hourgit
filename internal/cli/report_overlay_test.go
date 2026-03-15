package cli

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/timetrack"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntrySelectorOverlay_Navigation(t *testing.T) {
	entries := []timetrack.CellEntry{
		{ID: "e100001", Minutes: 60, Message: "first"},
		{ID: "e200002", Minutes: 120, Message: "second"},
	}
	o := newEntrySelectorOverlay(entries, "Pick one", "edit")

	assert.Equal(t, 0, o.cursor)

	// Move down
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyDown})
	o = updated.(*entrySelectorOverlay)
	assert.Equal(t, 1, o.cursor)

	// Move up
	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyUp})
	o = updated.(*entrySelectorOverlay)
	assert.Equal(t, 0, o.cursor)

	// Can't go above 0
	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyUp})
	o = updated.(*entrySelectorOverlay)
	assert.Equal(t, 0, o.cursor)
}

func TestEntrySelectorOverlay_Select(t *testing.T) {
	entries := []timetrack.CellEntry{
		{ID: "e100001", Minutes: 60, Message: "first"},
		{ID: "e200002", Minutes: 120, Message: "second"},
	}
	o := newEntrySelectorOverlay(entries, "Pick one", "edit")

	// Move to second entry and select
	o.cursor = 1
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "select", result.action)
	assert.Equal(t, "e200002", o.selectedEntry().ID)
}

func TestEntrySelectorOverlay_Cancel(t *testing.T) {
	entries := []timetrack.CellEntry{{ID: "e100001"}}
	o := newEntrySelectorOverlay(entries, "Pick", "edit")

	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "cancel", result.action)
}

func TestEntrySelectorOverlay_View(t *testing.T) {
	entries := []timetrack.CellEntry{
		{ID: "e100001", Minutes: 60, Message: "work", Persisted: true},
		{ID: "", Minutes: 120, Message: "generated", Persisted: false},
	}
	o := newEntrySelectorOverlay(entries, "Pick one", "edit")

	view := o.View()
	assert.Contains(t, view, "Pick one")
	assert.Contains(t, view, "work")
	assert.Contains(t, view, "generated")
}

func TestEditOverlay_FieldNavigation(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)

	assert.Equal(t, editFieldFrom, o.field)

	// Tab through all fields: From → To → Duration → Task → Message → Confirm
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldTo, o.field)

	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldDuration, o.field)

	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldTask, o.field)

	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldMessage, o.field)

	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldConfirm, o.field)

	// Shift-tab back
	updated, _ = o.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	o = updated.(*editOverlay)
	assert.Equal(t, editFieldMessage, o.field)
}

func TestEditOverlay_TextInput(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "",
		Task:    "",
	}
	o := newEditOverlay(ce)

	// First field is From — clear it and type new value
	o.from = ""
	for _, ch := range "10am" {
		updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		o = updated.(*editOverlay)
	}
	assert.Equal(t, "10am", o.from)

	// Backspace
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	o = updated.(*editOverlay)
	assert.Equal(t, "10a", o.from)
}

func TestEditOverlay_InvalidFrom(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)
	o.from = "invalid"
	o.field = editFieldConfirm

	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	o = updated.(*editOverlay)
	assert.Nil(t, cmd) // No result — stays in overlay
	assert.Contains(t, o.err, "Invalid from time")
}

func TestEditOverlay_InvalidTo(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)
	o.to = "invalid"
	o.field = editFieldConfirm

	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	o = updated.(*editOverlay)
	assert.Nil(t, cmd)
	assert.Contains(t, o.err, "Invalid to time")
}

func TestEditOverlay_ToBeforeFrom(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)
	o.from = "5pm"
	o.to = "9am"
	o.field = editFieldConfirm

	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	o = updated.(*editOverlay)
	assert.Nil(t, cmd)
	assert.Contains(t, o.err, "To must be after From")
}

func TestEditOverlay_ValidSubmit(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)
	o.from = "9:00"
	o.to = "11:30"
	o.duration = "2h30m"
	o.task = "updated-task"
	o.message = "updated-msg"
	o.field = editFieldConfirm

	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "edit", result.action)
	assert.Equal(t, 150, o.entry.Minutes) // 2h30m
	assert.Equal(t, 9, o.entry.Start.Hour())
	assert.Equal(t, "updated-task", o.entry.Task)
	assert.Equal(t, "updated-msg", o.entry.Message)
}

func TestEditOverlay_View(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 60,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
		Task:    "task",
	}
	o := newEditOverlay(ce)

	view := o.View()
	assert.Contains(t, view, "Edit Entry")
	assert.Contains(t, view, "From")
	assert.Contains(t, view, "To")
	assert.Contains(t, view, "Duration")
	assert.Contains(t, view, "Task")
	assert.Contains(t, view, "Message")
	assert.Contains(t, view, "Save")
}

func TestAddOverlay_ValidSubmit(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "my-task")
	o.duration = "1h"
	o.message = "new work"
	o.field = addFieldConfirm

	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "add", result.action)
}

func TestAddOverlay_EmptyDuration(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "task")
	o.duration = ""
	o.field = addFieldConfirm

	updated, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	o = updated.(*addOverlay)
	assert.Nil(t, cmd)
	assert.Contains(t, o.err, "Duration is required")
}

func TestAddOverlay_BuildEntry(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "my-task")
	o.duration = "2h"
	o.message = "did stuff"

	now := time.Date(2025, 1, 5, 14, 0, 0, 0, time.UTC)
	e, err := o.buildEntry(now)
	require.NoError(t, err)
	assert.Equal(t, 120, e.Minutes)
	assert.Equal(t, "did stuff", e.Message)
	assert.Equal(t, "my-task", e.Task)
	assert.Equal(t, "manual", e.Source)
	assert.Equal(t, 5, e.Start.Day())
}

func TestAddOverlay_BuildEntryMessageFallback(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "my-task")
	o.duration = "1h"
	o.message = "" // empty message should fall back to task

	now := time.Date(2025, 1, 5, 14, 0, 0, 0, time.UTC)
	e, err := o.buildEntry(now)
	require.NoError(t, err)
	assert.Equal(t, "my-task", e.Message)
}

func TestRemoveOverlay_Confirm(t *testing.T) {
	ce := timetrack.CellEntry{ID: "e100001", Minutes: 60, Message: "test"}
	o := newRemoveOverlay(ce)

	// Default cursor is on "No"
	assert.Equal(t, 1, o.confirm.cursor)

	// Press y for quick confirm
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "remove", result.action)
}

func TestRemoveOverlay_Cancel(t *testing.T) {
	ce := timetrack.CellEntry{ID: "e100001", Minutes: 60, Message: "test"}
	o := newRemoveOverlay(ce)

	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "cancel", result.action)
}

func TestRemoveOverlay_View(t *testing.T) {
	ce := timetrack.CellEntry{ID: "e100001", Minutes: 60, Message: "test work", Persisted: false}
	o := newRemoveOverlay(ce)

	view := o.View()
	assert.Contains(t, view, "Remove Entry")
	assert.Contains(t, view, "test work")
	assert.Contains(t, view, "(generated)")
	assert.Contains(t, view, "Yes")
	assert.Contains(t, view, "No")
}

func TestSubmitOverlay_Confirm(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	o := newSubmitOverlay(5, from, to)

	// Toggle to Yes
	updated, _ := o.Update(tea.KeyMsg{Type: tea.KeyTab})
	o = updated.(*submitOverlay)
	assert.Equal(t, 0, o.confirm.cursor)

	// Confirm
	_, cmd := o.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)

	msg := cmd()
	result, ok := msg.(overlayResult)
	require.True(t, ok)
	assert.Equal(t, "submit", result.action)
}

func TestSubmitOverlay_View(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)
	o := newSubmitOverlay(3, from, to)

	view := o.View()
	assert.Contains(t, view, "Submit Period")
	assert.Contains(t, view, "3 generated entries")
	assert.Contains(t, view, "Jan 1, 2025")
	assert.Contains(t, view, "Jan 31, 2025")
}

func TestReportModel_CountInMemoryEntries(t *testing.T) {
	m := reportModel{
		data: timetrack.DetailedReportData{
			Rows: []timetrack.DetailedTaskRow{
				{
					Name: "task",
					Days: map[int]*timetrack.CellData{
						1: {Entries: []timetrack.CellEntry{
							{Persisted: true},
							{Persisted: false},
						}},
						2: {Entries: []timetrack.CellEntry{
							{Persisted: false},
						}},
					},
				},
			},
		},
	}

	assert.Equal(t, 2, m.countInMemoryEntries())
}

func TestReportModel_AddCellEntry(t *testing.T) {
	m := reportModel{
		data: timetrack.DetailedReportData{
			Rows: []timetrack.DetailedTaskRow{
				{
					Name:         "existing",
					TotalMinutes: 60,
					Days: map[int]*timetrack.CellData{
						1: {TotalMinutes: 60, Entries: []timetrack.CellEntry{{ID: "e100001", Minutes: 60}}},
					},
				},
			},
		},
	}

	// Add to existing row
	ce := timetrack.CellEntry{ID: "e200002", Minutes: 30}
	m.addCellEntry("existing", 2, ce)
	assert.Equal(t, 90, m.data.Rows[0].TotalMinutes)
	assert.NotNil(t, m.data.Rows[0].Days[2])

	// Add to new task
	ce2 := timetrack.CellEntry{ID: "e300003", Minutes: 45}
	m.addCellEntry("new-task", 1, ce2)
	assert.Equal(t, 2, len(m.data.Rows))
}

func TestReportModel_RemoveCellEntry(t *testing.T) {
	m := reportModel{
		data: timetrack.DetailedReportData{
			Rows: []timetrack.DetailedTaskRow{
				{
					Name:         "task",
					TotalMinutes: 120,
					Days: map[int]*timetrack.CellData{
						1: {
							TotalMinutes: 120,
							Entries: []timetrack.CellEntry{
								{ID: "e100001", Minutes: 60},
								{ID: "e200002", Minutes: 60},
							},
						},
					},
				},
			},
		},
	}

	m.removeCellEntry(0, 1, timetrack.CellEntry{ID: "e100001", Minutes: 60})
	assert.Equal(t, 60, m.data.Rows[0].TotalMinutes)
	assert.Equal(t, 1, len(m.data.Rows[0].Days[1].Entries))

	// Remove last entry — cell data should be cleaned up
	m.removeCellEntry(0, 1, timetrack.CellEntry{ID: "e200002", Minutes: 60})
	assert.Equal(t, 0, m.data.Rows[0].TotalMinutes)
	assert.Nil(t, m.data.Rows[0].Days[1])
}

func TestReportModel_HandleEdit_PersistsInMemoryEntry(t *testing.T) {
	homeDir := t.TempDir()
	slug := "test-project"

	// Create project log dir
	_ = entry.WriteEntry(homeDir, slug, entry.Entry{ID: "d000001", Start: time.Now(), Minutes: 1, Message: "x", CreatedAt: time.Now()})
	_ = entry.DeleteEntry(homeDir, slug, "d000001")

	ce := timetrack.CellEntry{
		ID:        "",
		Minutes:   60,
		Start:     time.Date(2025, 1, 2, 9, 0, 0, 0, time.UTC),
		Message:   "feature-x",
		Task:      "feature-x",
		Source:    "checkout",
		Persisted: false,
	}

	editOv := newEditOverlay(ce)
	editOv.from = "9:00"
	editOv.to = "11:00"
	editOv.duration = "2h"
	editOv.task = "feature-x"
	editOv.message = "feature-x"
	editOv.field = editFieldConfirm

	// Trigger valid submit on the edit overlay
	updated, _ := editOv.Update(tea.KeyMsg{Type: tea.KeyEnter})
	editOv = updated.(*editOverlay)

	m := reportModel{
		homeDir: homeDir,
		slug:    slug,
		overlay: editOv,
		mode:    modeEditing,
		data: timetrack.DetailedReportData{
			DaysInMonth: 31,
			Rows: []timetrack.DetailedTaskRow{
				{
					Name:         "feature-x",
					TotalMinutes: 60,
					Days: map[int]*timetrack.CellData{
						2: {TotalMinutes: 60, Entries: []timetrack.CellEntry{ce}},
					},
				},
			},
		},
		cursorRow: 0,
		cursorCol: 1, // day 2
	}

	result, _ := m.handleEdit()
	rm := result.(reportModel)
	assert.Equal(t, modeNormal, rm.mode)
	assert.Nil(t, rm.overlay)
	assert.Contains(t, rm.footerMsg, "saved")
}

func TestEditOverlay_Interdependency_FromRecomputesTo(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 120,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
	}
	o := newEditOverlay(ce)
	// Initial: from=9:00, to=11:00, duration=2h
	assert.Equal(t, "09:00", o.from)
	assert.Equal(t, "11:00", o.to)
	assert.Equal(t, "2h", o.duration)

	// Change from to 10:00 and tab away → to should become 12:00 (keeping 2h duration)
	o.from = "10:00"
	// Advance field triggers recompute
	o.advanceField()
	assert.Equal(t, "12:00", o.to)
	assert.Equal(t, "2h", o.duration)
}

func TestEditOverlay_Interdependency_ToRecomputesDuration(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 120,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
	}
	o := newEditOverlay(ce)
	// Navigate to To field
	o.field = editFieldTo

	// Change to to 14:00 and tab away → duration should become 5h (9:00 to 14:00)
	o.to = "14:00"
	o.advanceField()
	assert.Equal(t, "5h", o.duration)
	assert.Equal(t, "09:00", o.from) // from unchanged
}

func TestEditOverlay_Interdependency_DurationRecomputesTo(t *testing.T) {
	ce := timetrack.CellEntry{
		Minutes: 120,
		Start:   time.Date(2025, 1, 5, 9, 0, 0, 0, time.UTC),
		Message: "test",
	}
	o := newEditOverlay(ce)
	// Navigate to Duration field
	o.field = editFieldDuration

	// Change duration to 4h and tab away → to should become 13:00 (9:00 + 4h)
	o.duration = "4h"
	o.advanceField()
	assert.Equal(t, "13:00", o.to)
	assert.Equal(t, "09:00", o.from) // from unchanged
}

func TestAddOverlay_FieldNavigation(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "my-task")

	assert.Equal(t, addFieldFrom, o.field)

	// Tab through: From → To → Duration → Task → Message → Confirm
	o.advanceField()
	assert.Equal(t, addFieldTo, o.field)
	o.advanceField()
	assert.Equal(t, addFieldDuration, o.field)
	o.advanceField()
	assert.Equal(t, addFieldTask, o.field)
	o.advanceField()
	assert.Equal(t, addFieldMessage, o.field)
	o.advanceField()
	assert.Equal(t, addFieldConfirm, o.field)

	// Back
	o.retreatField()
	assert.Equal(t, addFieldMessage, o.field)
}

func TestAddOverlay_Interdependency(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "task")
	o.from = "9:00"
	o.duration = "3h"

	// Leaving from → recomputes to
	o.field = addFieldFrom
	o.advanceField()
	assert.Equal(t, "12:00", o.to)

	// Change to and leave → recomputes duration
	o.to = "14:00"
	o.field = addFieldTo
	o.advanceField()
	assert.Equal(t, "5h", o.duration)
}

func TestAddOverlay_BuildEntryUsesFrom(t *testing.T) {
	o := newAddOverlay(5, time.January, 2025, "my-task")
	o.from = "10:30"
	o.duration = "2h"
	o.message = "did stuff"

	now := time.Date(2025, 1, 5, 14, 0, 0, 0, time.UTC)
	e, err := o.buildEntry(now)
	require.NoError(t, err)
	assert.Equal(t, 120, e.Minutes)
	assert.Equal(t, 10, e.Start.Hour())
	assert.Equal(t, 30, e.Start.Minute())
}
