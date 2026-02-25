package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/charmbracelet/lipgloss"
)

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

// isWeekend returns true if the given date falls on Saturday or Sunday.
func isWeekend(year int, month time.Month, day int) bool {
	wd := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Weekday()
	return wd == time.Saturday || wd == time.Sunday
}

// dayAbbrev returns a 3-letter weekday abbreviation for the given date.
func dayAbbrev(year int, month time.Month, day int) string {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Weekday().String()[:3]
}

// renderDetailedTable produces the table string from DetailedReportData with cursor highlighting.
func renderDetailedTable(data timetrack.DetailedReportData, scrollX, scrollY, visibleDays, visibleRows, cursorRow, cursorCol int, submitted bool, footerMsg string) string {
	var b strings.Builder

	// Warning banner for submitted periods
	if submitted {
		b.WriteString(Warning("Previously submitted. Changes require re-submission."))
		b.WriteString("\n")
	}

	// Title line
	title := fmt.Sprintf("--- %s %d ---", data.Month, data.Year)
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n")

	// Header row
	b.WriteString(headerStyle.Render(padRight("Task", taskColWidth)))
	b.WriteString(" | ")
	b.WriteString(headerStyle.Render(padCenter("Sum", dayColWidth)))
	for i := 0; i < visibleDays; i++ {
		day := scrollX + i + 1
		b.WriteString(" | ")
		label := fmt.Sprintf("%s %d", dayAbbrev(data.Year, data.Month, day), day)
		if isWeekend(data.Year, data.Month, day) {
			b.WriteString(weekendStyle.Bold(true).Render(padCenter(label, dayColWidth)))
		} else {
			b.WriteString(headerStyle.Render(padCenter(label, dayColWidth)))
		}
	}
	b.WriteString("\n")

	// Separator
	b.WriteString(strings.Repeat("-", taskColWidth))
	b.WriteString("-+-")
	b.WriteString(strings.Repeat("-", dayColWidth))
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
		if len(label) > taskColWidth {
			label = label[:taskColWidth-3] + "..."
		}
		b.WriteString(padRight(label, taskColWidth))

		// Sum column
		b.WriteString(" | ")
		b.WriteString(padCenter(entry.FormatMinutes(row.TotalMinutes), dayColWidth))

		for i := 0; i < visibleDays; i++ {
			day := scrollX + i + 1
			colIdx := scrollX + i // 0-indexed day column
			b.WriteString(" | ")

			weekend := isWeekend(data.Year, data.Month, day)
			scheduled := data.ScheduledDays[day]

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
			} else if !scheduled {
				cellText = padCenter("x", dayColWidth)
			} else {
				cellText = padCenter(".", dayColWidth)
			}

			if rowIdx == cursorRow && colIdx == cursorCol {
				b.WriteString(selectedStyle.Render(cellText))
			} else if cd != nil && cd.TotalMinutes > 0 {
				if weekend {
					b.WriteString(weekendStyle.Render(cellText))
				} else {
					b.WriteString(cellText)
				}
			} else if !scheduled {
				b.WriteString(unscheduledStyle.Render(cellText))
			} else {
				b.WriteString(dotStyle.Render(cellText))
			}
		}
		b.WriteString("\n")
	}

	// Totals separator
	b.WriteString(strings.Repeat("-", taskColWidth))
	b.WriteString("-+-")
	b.WriteString(strings.Repeat("-", dayColWidth))
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
	b.WriteString(headerStyle.Render(padRight("Total", taskColWidth)))

	// Grand total in Sum column
	b.WriteString(" | ")
	b.WriteString(headerStyle.Render(padCenter(entry.FormatMinutes(totalMinutes), dayColWidth)))

	for i := 0; i < visibleDays; i++ {
		day := scrollX + i + 1
		b.WriteString(" | ")

		weekend := isWeekend(data.Year, data.Month, day)
		scheduled := data.ScheduledDays[day]

		dayTotal := 0
		for _, row := range data.Rows {
			cd := row.Days[day]
			if cd != nil {
				dayTotal += cd.TotalMinutes
			}
		}
		if dayTotal > 0 {
			if weekend {
				b.WriteString(weekendStyle.Bold(true).Render(padCenter(entry.FormatMinutes(dayTotal), dayColWidth)))
			} else {
				b.WriteString(headerStyle.Render(padCenter(entry.FormatMinutes(dayTotal), dayColWidth)))
			}
		} else if !scheduled {
			b.WriteString(unscheduledStyle.Render(padCenter("x", dayColWidth)))
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
