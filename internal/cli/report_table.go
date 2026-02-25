package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/timetrack"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const (
	taskColWidth = 30
	dayColWidth  = 7
)

var (
	headerStyle   = lipgloss.NewStyle().Bold(true)
	footerStyle   = lipgloss.NewStyle().Faint(true)
	dotStyle      = lipgloss.NewStyle().Faint(true)
	selectedStyle = lipgloss.NewStyle().Reverse(true)
)

// reportMode represents the current interaction mode of the report table.
type reportMode int

const (
	modeNormal reportMode = iota
	modeEditing
	modeAdding
	modeRemoving
	modeSubmitting
)

type reportModel struct {
	data       timetrack.DetailedReportData
	scrollX    int // first visible day column (0-indexed offset into days)
	scrollY    int // first visible row (0-indexed offset into rows)
	cursorRow  int // selected row index (into data.Rows)
	cursorCol  int // selected day column (0-indexed offset into days, -1 = task name column)
	termWidth  int
	termHeight int
	mode       reportMode
	overlay    tea.Model // active overlay (nil in normal mode)
	homeDir    string
	slug       string
	submitted  bool   // whether period was previously submitted
	footerMsg  string // temporary message shown in footer
}

func (m reportModel) visibleDays() int {
	available := m.termWidth - taskColWidth - 3 // separators + padding
	if available <= 0 {
		return 1
	}
	cols := available / (dayColWidth + 3) // " | " separator
	if cols < 1 {
		cols = 1
	}
	if cols > m.data.DaysInMonth {
		cols = m.data.DaysInMonth
	}
	return cols
}

func (m reportModel) visibleRows() int {
	// Reserve lines for: header(1) + separator(1) + totals separator(1) + totals(1) + footer(2) + warning(1)
	reserved := 7
	if m.submitted {
		reserved++
	}
	available := m.termHeight - reserved
	if available < 1 {
		return 1
	}
	if available > len(m.data.Rows) {
		return len(m.data.Rows)
	}
	return available
}

func (m reportModel) maxScrollX() int {
	max := m.data.DaysInMonth - m.visibleDays()
	if max < 0 {
		return 0
	}
	return max
}

func (m reportModel) maxScrollY() int {
	max := len(m.data.Rows) - m.visibleRows()
	if max < 0 {
		return 0
	}
	return max
}

func (m reportModel) Init() tea.Cmd {
	return nil
}

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

// countInMemoryEntries counts entries with Persisted=false across all cells.
func (m reportModel) countInMemoryEntries() int {
	count := 0
	for _, row := range m.data.Rows {
		for _, cd := range row.Days {
			for _, ce := range cd.Entries {
				if !ce.Persisted {
					count++
				}
			}
		}
	}
	return count
}

// updateCellEntry updates an existing entry in the report data.
func (m *reportModel) updateCellEntry(rowIdx, day int, updated timetrack.CellEntry) {
	if rowIdx >= len(m.data.Rows) {
		return
	}
	row := &m.data.Rows[rowIdx]
	cd := row.Days[day]
	if cd == nil {
		return
	}

	for i, ce := range cd.Entries {
		if (ce.ID != "" && ce.ID == updated.ID) || (ce.ID == "" && !ce.Persisted && ce.Message == updated.Message) {
			oldMins := ce.Minutes
			cd.Entries[i] = updated
			cd.TotalMinutes += updated.Minutes - oldMins
			row.TotalMinutes += updated.Minutes - oldMins
			break
		}
	}
}

// addCellEntry adds a new entry to the report data, finding or creating the appropriate row.
func (m *reportModel) addCellEntry(task string, day int, ce timetrack.CellEntry) {
	// Find existing row for this task
	rowIdx := -1
	for i, row := range m.data.Rows {
		if row.Name == task {
			rowIdx = i
			break
		}
	}

	if rowIdx == -1 {
		// Create new row
		m.data.Rows = append(m.data.Rows, timetrack.DetailedTaskRow{
			Name:         task,
			TotalMinutes: ce.Minutes,
			Days: map[int]*timetrack.CellData{
				day: {
					Entries:      []timetrack.CellEntry{ce},
					TotalMinutes: ce.Minutes,
				},
			},
		})
		return
	}

	row := &m.data.Rows[rowIdx]
	cd := row.Days[day]
	if cd == nil {
		cd = &timetrack.CellData{}
		row.Days[day] = cd
	}
	cd.Entries = append(cd.Entries, ce)
	cd.TotalMinutes += ce.Minutes
	row.TotalMinutes += ce.Minutes
}

// removeCellEntry removes an entry from the report data.
func (m *reportModel) removeCellEntry(rowIdx, day int, ce timetrack.CellEntry) {
	if rowIdx >= len(m.data.Rows) {
		return
	}
	row := &m.data.Rows[rowIdx]
	cd := row.Days[day]
	if cd == nil {
		return
	}

	for i, existing := range cd.Entries {
		match := false
		if existing.ID != "" && existing.ID == ce.ID {
			match = true
		} else if existing.ID == "" && !existing.Persisted && existing.Message == ce.Message && existing.Minutes == ce.Minutes {
			match = true
		}
		if match {
			cd.Entries = append(cd.Entries[:i], cd.Entries[i+1:]...)
			cd.TotalMinutes -= ce.Minutes
			row.TotalMinutes -= ce.Minutes
			break
		}
	}

	// Clean up empty cell data
	if len(cd.Entries) == 0 {
		delete(row.Days, day)
	}
}

// ensureCursorVisible adjusts scroll so the cursor is within the visible viewport.
func (m reportModel) ensureCursorVisible() reportModel {
	// Horizontal
	if m.cursorCol < m.scrollX {
		m.scrollX = m.cursorCol
	}
	if m.cursorCol >= m.scrollX+m.visibleDays() {
		m.scrollX = m.cursorCol - m.visibleDays() + 1
	}
	// Vertical
	if m.cursorRow < m.scrollY {
		m.scrollY = m.cursorRow
	}
	if m.cursorRow >= m.scrollY+m.visibleRows() {
		m.scrollY = m.cursorRow - m.visibleRows() + 1
	}
	return m.clampScroll()
}

// clampScroll ensures scroll values are within valid bounds.
func (m reportModel) clampScroll() reportModel {
	if m.scrollX > m.maxScrollX() {
		m.scrollX = m.maxScrollX()
	}
	if m.scrollX < 0 {
		m.scrollX = 0
	}
	if m.scrollY > m.maxScrollY() {
		m.scrollY = m.maxScrollY()
	}
	if m.scrollY < 0 {
		m.scrollY = 0
	}
	return m
}

func (m reportModel) View() string {
	tableStr := renderDetailedTable(m.data, m.scrollX, m.scrollY, m.visibleDays(), m.visibleRows(), m.cursorRow, m.cursorCol, m.submitted, m.footerMsg)

	// If overlay is active, render it on top
	if m.overlay != nil {
		overlayStr := m.overlay.View()
		return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, overlayStr,
			lipgloss.WithWhitespaceChars(" "),
		)
	}

	return tableStr
}

// renderDetailedTable produces the table string from DetailedReportData with cursor highlighting.
func renderDetailedTable(data timetrack.DetailedReportData, scrollX, scrollY, visibleDays, visibleRows, cursorRow, cursorCol int, submitted bool, footerMsg string) string {
	var b strings.Builder

	// Warning banner for submitted periods
	if submitted {
		b.WriteString(Warning("Previously submitted. Changes require re-submission."))
		b.WriteString("\n")
	}

	// Header row
	b.WriteString(headerStyle.Render(padRight("Task", taskColWidth)))
	for i := 0; i < visibleDays; i++ {
		day := scrollX + i + 1
		b.WriteString(" | ")
		b.WriteString(headerStyle.Render(padCenter(fmt.Sprintf("%d", day), dayColWidth)))
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("-", taskColWidth))
	for i := 0; i < visibleDays; i++ {
		b.WriteString("-+-")
		b.WriteString(strings.Repeat("-", dayColWidth))
	}
	b.WriteString("\n")

	// Data rows (respecting vertical scroll)
	endRow := scrollY + visibleRows
	if endRow > len(data.Rows) {
		endRow = len(data.Rows)
	}
	for rowIdx := scrollY; rowIdx < endRow; rowIdx++ {
		row := data.Rows[rowIdx]
		label := row.Name
		if len(label) > taskColWidth-12 {
			label = label[:taskColWidth-15] + "..."
		}
		label = fmt.Sprintf("%s [%s]", label, entry.FormatMinutes(row.TotalMinutes))
		b.WriteString(padRight(label, taskColWidth))

		for i := 0; i < visibleDays; i++ {
			day := scrollX + i + 1
			colIdx := scrollX + i // 0-indexed day column
			b.WriteString(" | ")

			cd := row.Days[day]
			cellText := ""
			if cd != nil && cd.TotalMinutes > 0 {
				cellText = padCenter(entry.FormatMinutes(cd.TotalMinutes), dayColWidth)
				// Mark cells containing in-memory entries with an asterisk
				hasInMemory := false
				for _, ce := range cd.Entries {
					if !ce.Persisted {
						hasInMemory = true
						break
					}
				}
				if hasInMemory {
					cellText = padCenter(entry.FormatMinutes(cd.TotalMinutes)+"*", dayColWidth)
				}
			} else {
				cellText = padCenter(".", dayColWidth)
			}

			if rowIdx == cursorRow && colIdx == cursorCol {
				b.WriteString(selectedStyle.Render(cellText))
			} else if cd != nil && cd.TotalMinutes > 0 {
				b.WriteString(cellText)
			} else {
				b.WriteString(dotStyle.Render(cellText))
			}
		}
		b.WriteString("\n")
	}

	// Totals separator
	b.WriteString(strings.Repeat("-", taskColWidth))
	for i := 0; i < visibleDays; i++ {
		b.WriteString("-+-")
		b.WriteString(strings.Repeat("-", dayColWidth))
	}
	b.WriteString("\n")

	// Totals row
	totalMinutes := 0
	for _, row := range data.Rows {
		totalMinutes += row.TotalMinutes
	}
	totalLabel := fmt.Sprintf("Total [%s]", entry.FormatMinutes(totalMinutes))
	b.WriteString(headerStyle.Render(padRight(totalLabel, taskColWidth)))

	for i := 0; i < visibleDays; i++ {
		day := scrollX + i + 1
		b.WriteString(" | ")
		dayTotal := 0
		for _, row := range data.Rows {
			cd := row.Days[day]
			if cd != nil {
				dayTotal += cd.TotalMinutes
			}
		}
		if dayTotal > 0 {
			b.WriteString(headerStyle.Render(padCenter(entry.FormatMinutes(dayTotal), dayColWidth)))
		} else {
			b.WriteString(dotStyle.Render(padCenter(".", dayColWidth)))
		}
	}
	b.WriteString("\n")

	// Footer
	b.WriteString("\n")
	footer := fmt.Sprintf(
		"%s %d  |  ←/→/↑/↓ navigate  |  e edit  |  a add  |  r remove  |  s submit  |  q quit",
		data.Month, data.Year,
	)
	if footerMsg != "" {
		footer = footerMsg + "  |  " + footer
	}
	b.WriteString(footerStyle.Render(footer))
	b.WriteString("\n")

	return b.String()
}

func runReportTable(cmd *cobra.Command, data timetrack.DetailedReportData, homeDir, slug string, submitted bool) error {
	out := cmd.OutOrStdout()

	// Non-TTY fallback: print static table
	if f, ok := out.(*os.File); !ok || !isatty.IsTerminal(f.Fd()) {
		return printStaticDetailedTable(out, data)
	}

	m := reportModel{
		data:       data,
		termWidth:  120,
		termHeight: 40,
		homeDir:    homeDir,
		slug:       slug,
		submitted:  submitted,
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(out))
	_, err := p.Run()
	return err
}

func printStaticDetailedTable(w io.Writer, data timetrack.DetailedReportData) error {
	_, err := fmt.Fprint(w, renderDetailedTable(data, 0, 0, data.DaysInMonth, len(data.Rows), -1, -1, false, ""))
	return err
}

func padRight(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func padCenter(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	total := width - len(s)
	left := total / 2
	right := total - left
	return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
}
