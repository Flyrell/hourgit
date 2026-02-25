package cli

import (
	"fmt"
	"io"
	"os"

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
