package cli

import (
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/timetrack"
	tea "github.com/charmbracelet/bubbletea"
)

func (m reportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	// If overlay is active, delegate to it
	if m.overlay != nil {
		return m.updateOverlay(msg)
	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m = m.clampScroll()
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "right", "l":
			if m.cursorCol < m.data.DaysInMonth-1 {
				m.cursorCol++
				m = m.ensureCursorVisible()
			}
		case "left", "h":
			if m.cursorCol > 0 {
				m.cursorCol--
				m = m.ensureCursorVisible()
			}
		case "down", "j":
			if m.cursorRow < len(m.data.Rows)-1 {
				m.cursorRow++
				m = m.ensureCursorVisible()
			}
		case "up", "k":
			if m.cursorRow > 0 {
				m.cursorRow--
				m = m.ensureCursorVisible()
			}
		case "e":
			return m.startEdit()
		case "a":
			return m.startAdd()
		case "r", "delete", "backspace":
			return m.startRemove()
		case "s":
			return m.startSubmit()
		}
	}
	return m, nil
}

// updateOverlay delegates input to the active overlay and handles overlay results.
func (m reportModel) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Handle overlay result messages directly
	if result, ok := msg.(overlayResult); ok {
		return m.handleOverlayResult(result)
	}

	updated, cmd := m.overlay.Update(msg)
	m.overlay = updated
	return m, cmd
}

// handleOverlayResult processes the result when an overlay completes.
func (m reportModel) handleOverlayResult(result overlayResult) (tea.Model, tea.Cmd) {
	switch result.action {
	case "cancel":
		m.overlay = nil
		m.mode = modeNormal
		m.footerMsg = ""
		return m, nil

	case "select":
		return m.handleSelection()

	case "edit":
		return m.handleEdit()

	case "add":
		return m.handleAdd()

	case "remove":
		return m.handleRemove()

	case "submit":
		return m.handleSubmit()
	}

	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) handleSelection() (tea.Model, tea.Cmd) {
	selector, ok := m.overlay.(*entrySelectorOverlay)
	if !ok {
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	selected := selector.selectedEntry()
	switch selector.action {
	case "edit":
		m.overlay = newEditOverlay(selected)
		return m, nil
	case "remove":
		m.overlay = newRemoveOverlay(selected)
		return m, nil
	}

	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) handleEdit() (tea.Model, tea.Cmd) {
	editor, ok := m.overlay.(*editOverlay)
	if !ok {
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	ce := editor.entry
	day := m.cursorCol + 1

	if ce.Persisted && ce.Entry != nil {
		// Update existing persisted entry
		ce.Entry.Minutes = ce.Minutes
		ce.Entry.Task = ce.Task
		ce.Entry.Message = ce.Message
		if err := entry.WriteEntry(m.homeDir, m.slug, *ce.Entry); err != nil {
			m.footerMsg = "Error saving: " + err.Error()
			m.overlay = nil
			m.mode = modeNormal
			return m, nil
		}
	} else {
		// Persist in-memory entry (checkout-generated)
		e := entry.Entry{
			ID:        hashutil.GenerateID("edit"),
			Start:     ce.Start,
			Minutes:   ce.Minutes,
			Message:   ce.Message,
			Task:      ce.Task,
			Source:    "checkout-generated",
			CreatedAt: time.Now().UTC(),
		}
		if err := entry.WriteEntry(m.homeDir, m.slug, e); err != nil {
			m.footerMsg = "Error saving: " + err.Error()
			m.overlay = nil
			m.mode = modeNormal
			return m, nil
		}
		ce.ID = e.ID
		ce.Persisted = true
		ce.Entry = &e
	}

	// Update in-place in DetailedReportData
	m.updateCellEntry(m.cursorRow, day, ce)
	m.footerMsg = "Entry saved"
	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) handleAdd() (tea.Model, tea.Cmd) {
	adder, ok := m.overlay.(*addOverlay)
	if !ok {
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	e, err := adder.buildEntry(time.Now())
	if err != nil {
		m.footerMsg = "Error: " + err.Error()
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	if err := entry.WriteEntry(m.homeDir, m.slug, e); err != nil {
		m.footerMsg = "Error saving: " + err.Error()
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	// Add to report data
	day := m.cursorCol + 1
	task := e.Task
	if task == "" {
		task = e.Message
	}

	ce := timetrack.CellEntry{
		ID:        e.ID,
		Start:     e.Start,
		Minutes:   e.Minutes,
		Message:   e.Message,
		Task:      e.Task,
		Source:    e.Source,
		Persisted: true,
		Entry:    &e,
	}

	m.addCellEntry(task, day, ce)
	m.footerMsg = "Entry added"
	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) handleRemove() (tea.Model, tea.Cmd) {
	remover, ok := m.overlay.(*removeOverlay)
	if !ok {
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	ce := remover.entry
	day := m.cursorCol + 1

	if ce.Persisted {
		if err := entry.DeleteEntry(m.homeDir, m.slug, ce.ID); err != nil {
			m.footerMsg = "Error removing: " + err.Error()
			m.overlay = nil
			m.mode = modeNormal
			return m, nil
		}
	}

	// Remove from report data
	m.removeCellEntry(m.cursorRow, day, ce)
	m.footerMsg = "Entry removed"
	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) handleSubmit() (tea.Model, tea.Cmd) {
	// Persist all in-memory entries
	for rowIdx := range m.data.Rows {
		row := &m.data.Rows[rowIdx]
		for day, cd := range row.Days {
			for i, ce := range cd.Entries {
				if ce.Persisted {
					continue
				}
				e := entry.Entry{
					ID:        hashutil.GenerateID("submit"),
					Start:     ce.Start,
					Minutes:   ce.Minutes,
					Message:   ce.Message,
					Task:      ce.Task,
					Source:    "checkout-generated",
					CreatedAt: time.Now().UTC(),
				}
				if err := entry.WriteEntry(m.homeDir, m.slug, e); err != nil {
					m.footerMsg = "Error submitting: " + err.Error()
					m.overlay = nil
					m.mode = modeNormal
					return m, nil
				}
				cd.Entries[i].ID = e.ID
				cd.Entries[i].Persisted = true
				cd.Entries[i].Entry = &e
			}
			row.Days[day] = cd
		}
	}

	// Create submit marker
	submitEntry := entry.SubmitEntry{
		ID:        hashutil.GenerateID("submit-marker"),
		From:      m.data.From,
		To:        m.data.To,
		CreatedAt: time.Now().UTC(),
	}
	if err := entry.WriteSubmitEntry(m.homeDir, m.slug, submitEntry); err != nil {
		m.footerMsg = "Error creating submit marker: " + err.Error()
		m.overlay = nil
		m.mode = modeNormal
		return m, nil
	}

	m.submitted = true
	m.footerMsg = "Period submitted"
	m.overlay = nil
	m.mode = modeNormal
	return m, nil
}

func (m reportModel) startEdit() (tea.Model, tea.Cmd) {
	if m.cursorRow >= len(m.data.Rows) || m.cursorCol < 0 || m.cursorCol >= m.data.DaysInMonth {
		return m, nil
	}
	day := m.cursorCol + 1
	cd := m.data.Rows[m.cursorRow].Days[day]
	if cd == nil || len(cd.Entries) == 0 {
		m.footerMsg = "No entries to edit in this cell"
		return m, nil
	}
	m.mode = modeEditing
	if len(cd.Entries) == 1 {
		m.overlay = newEditOverlay(cd.Entries[0])
	} else {
		m.overlay = newEntrySelectorOverlay(cd.Entries, "Select entry to edit", "edit")
	}
	return m, nil
}

func (m reportModel) startAdd() (tea.Model, tea.Cmd) {
	if m.cursorCol < 0 || m.cursorCol >= m.data.DaysInMonth {
		return m, nil
	}
	day := m.cursorCol + 1
	task := ""
	if m.cursorRow >= 0 && m.cursorRow < len(m.data.Rows) {
		task = m.data.Rows[m.cursorRow].Name
	}
	m.mode = modeAdding
	m.overlay = newAddOverlay(day, m.data.Month, m.data.Year, task)
	return m, nil
}

func (m reportModel) startRemove() (tea.Model, tea.Cmd) {
	if m.cursorRow >= len(m.data.Rows) || m.cursorCol < 0 || m.cursorCol >= m.data.DaysInMonth {
		return m, nil
	}
	day := m.cursorCol + 1
	cd := m.data.Rows[m.cursorRow].Days[day]
	if cd == nil || len(cd.Entries) == 0 {
		m.footerMsg = "No entries to remove in this cell"
		return m, nil
	}
	m.mode = modeRemoving
	if len(cd.Entries) == 1 {
		m.overlay = newRemoveOverlay(cd.Entries[0])
	} else {
		m.overlay = newEntrySelectorOverlay(cd.Entries, "Select entry to remove", "remove")
	}
	return m, nil
}

func (m reportModel) startSubmit() (tea.Model, tea.Cmd) {
	inMemoryCount := m.countInMemoryEntries()
	m.mode = modeSubmitting
	m.overlay = newSubmitOverlay(inMemoryCount, m.data.From, m.data.To)
	return m, nil
}
