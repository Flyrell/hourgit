package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Flyrell/hourgit/internal/entry"
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
	headerStyle = lipgloss.NewStyle().Bold(true)
	footerStyle = lipgloss.NewStyle().Faint(true)
	dotStyle    = lipgloss.NewStyle().Faint(true)
)

type reportModel struct {
	data      timetrack.ReportData
	scroll    int // first visible day column (0-indexed offset into days)
	termWidth int
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

func (m reportModel) maxScroll() int {
	max := m.data.DaysInMonth - m.visibleDays()
	if max < 0 {
		return 0
	}
	return max
}

func (m reportModel) Init() tea.Cmd {
	return nil
}

func (m reportModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		if m.scroll > m.maxScroll() {
			m.scroll = m.maxScroll()
		}
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "right", "l":
			if m.scroll < m.maxScroll() {
				m.scroll++
			}
		case "left", "h":
			if m.scroll > 0 {
				m.scroll--
			}
		}
	}
	return m, nil
}

func (m reportModel) View() string {
	return renderTable(m.data, m.scroll, m.visibleDays(), true)
}

// renderTable produces the table string. If showFooter is true, navigation hints are appended.
func renderTable(data timetrack.ReportData, scroll, visibleDays int, showFooter bool) string {
	var b strings.Builder

	// Header row
	b.WriteString(headerStyle.Render(padRight("Task", taskColWidth)))
	for i := 0; i < visibleDays; i++ {
		day := scroll + i + 1
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

	// Data rows
	for _, row := range data.Rows {
		label := row.Name
		if len(label) > taskColWidth-12 {
			label = label[:taskColWidth-15] + "..."
		}
		label = fmt.Sprintf("%s [%s]", label, entry.FormatMinutes(row.TotalMinutes))
		b.WriteString(padRight(label, taskColWidth))

		for i := 0; i < visibleDays; i++ {
			day := scroll + i + 1
			b.WriteString(" | ")
			mins := row.Days[day]
			if mins > 0 {
				b.WriteString(padCenter(entry.FormatMinutes(mins), dayColWidth))
			} else {
				b.WriteString(dotStyle.Render(padCenter(".", dayColWidth)))
			}
		}
		b.WriteString("\n")
	}

	// Totals row
	b.WriteString(strings.Repeat("-", taskColWidth))
	for i := 0; i < visibleDays; i++ {
		b.WriteString("-+-")
		b.WriteString(strings.Repeat("-", dayColWidth))
	}
	b.WriteString("\n")

	totalMinutes := 0
	for _, row := range data.Rows {
		totalMinutes += row.TotalMinutes
	}
	totalLabel := fmt.Sprintf("Total [%s]", entry.FormatMinutes(totalMinutes))
	b.WriteString(headerStyle.Render(padRight(totalLabel, taskColWidth)))

	for i := 0; i < visibleDays; i++ {
		day := scroll + i + 1
		b.WriteString(" | ")
		dayTotal := 0
		for _, row := range data.Rows {
			dayTotal += row.Days[day]
		}
		if dayTotal > 0 {
			b.WriteString(headerStyle.Render(padCenter(entry.FormatMinutes(dayTotal), dayColWidth)))
		} else {
			b.WriteString(dotStyle.Render(padCenter(".", dayColWidth)))
		}
	}
	b.WriteString("\n")

	if showFooter {
		b.WriteString("\n")
		b.WriteString(footerStyle.Render(fmt.Sprintf(
			"%s %d  |  ←/→ scroll  |  q quit",
			data.Month, data.Year,
		)))
		b.WriteString("\n")
	}

	return b.String()
}

func runReportTable(cmd *cobra.Command, data timetrack.ReportData) error {
	out := cmd.OutOrStdout()

	// Non-TTY fallback: print static table
	if f, ok := out.(*os.File); !ok || !isatty.IsTerminal(f.Fd()) {
		return printStaticTable(out, data)
	}

	m := reportModel{
		data:      data,
		termWidth: 120, // default, updated by WindowSizeMsg
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithOutput(out))
	_, err := p.Run()
	return err
}

func printStaticTable(w io.Writer, data timetrack.ReportData) error {
	_, err := fmt.Fprint(w, renderTable(data, 0, data.DaysInMonth, false))
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
