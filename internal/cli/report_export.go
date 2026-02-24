package cli

import (
	"fmt"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/line"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/consts/pagesize"
	"github.com/johnfercher/maroto/v2/pkg/props"
)

var (
	pdfHeaderColor = props.Color{Red: 50, Green: 50, Blue: 50}
	pdfMutedColor  = props.Color{Red: 120, Green: 120, Blue: 120}
	pdfLineColor   = props.Color{Red: 200, Green: 200, Blue: 200}
)

// renderExportPDF generates a PDF timesheet from the export data and saves it
// to the given path.
func renderExportPDF(data timetrack.ExportData, outputPath string) error {
	cfg := config.NewBuilder().
		WithPageSize(pagesize.A4).
		WithLeftMargin(15).
		WithTopMargin(15).
		WithRightMargin(15).
		Build()

	m := maroto.New(cfg)

	// Document header
	m.AddRow(14,
		text.NewCol(12, data.ProjectName, props.Text{
			Style: fontstyle.Bold,
			Size:  16,
			Color: &pdfHeaderColor,
		}),
	)
	m.AddRow(8,
		text.NewCol(12, fmt.Sprintf("%s %d", data.Month, data.Year), props.Text{
			Size:  12,
			Color: &pdfMutedColor,
		}),
	)
	m.AddRow(4, line.NewCol(12, props.Line{Color: &pdfLineColor}))
	m.AddRow(4) // spacer

	// Day sections
	for _, day := range data.Days {
		weekday := day.Date.Weekday()
		dayLabel := fmt.Sprintf("%s %d, %s",
			day.Date.Month(), day.Date.Day(), weekday)
		dayTotal := entry.FormatMinutes(day.TotalMinutes)

		// Day header row
		m.AddRow(8,
			text.NewCol(9, dayLabel, props.Text{
				Style: fontstyle.Bold,
				Size:  10,
				Color: &pdfHeaderColor,
			}),
			text.NewCol(3, dayTotal, props.Text{
				Style: fontstyle.Bold,
				Size:  10,
				Align: align.Right,
				Color: &pdfHeaderColor,
			}),
		)

		// Task groups
		for _, group := range day.Groups {
			if len(group.Entries) == 1 && group.Entries[0].Message == group.Task {
				// Single-entry group where task == message: show as standalone row
				m.AddRow(6,
					text.NewCol(9, "  "+group.Task, props.Text{Size: 9}),
					text.NewCol(3, entry.FormatMinutes(group.TotalMinutes), props.Text{
						Size:  9,
						Align: align.Right,
					}),
				)
			} else {
				// Task group header
				m.AddRow(6,
					text.NewCol(9, "  "+group.Task, props.Text{
						Style: fontstyle.Bold,
						Size:  9,
					}),
					text.NewCol(3, entry.FormatMinutes(group.TotalMinutes), props.Text{
						Style: fontstyle.Bold,
						Size:  9,
						Align: align.Right,
					}),
				)

				// Individual entries
				for _, e := range group.Entries {
					m.AddRow(5,
						text.NewCol(9, "    "+e.Message, props.Text{
							Size:  8,
							Color: &pdfMutedColor,
						}),
						text.NewCol(3, entry.FormatMinutes(e.Minutes), props.Text{
							Size:  8,
							Align: align.Right,
							Color: &pdfMutedColor,
						}),
					)
				}
			}
		}

		// Spacer between days
		m.AddRow(4)
	}

	// Grand total footer
	m.AddRow(4, line.NewCol(12, props.Line{Color: &pdfLineColor}))
	m.AddRow(10,
		text.NewCol(9, "Total", props.Text{
			Style: fontstyle.Bold,
			Size:  12,
			Color: &pdfHeaderColor,
		}),
		text.NewCol(3, entry.FormatMinutes(data.TotalMinutes), props.Text{
			Style: fontstyle.Bold,
			Size:  12,
			Align: align.Right,
			Color: &pdfHeaderColor,
		}),
	)

	doc, err := m.Generate()
	if err != nil {
		return fmt.Errorf("generating PDF: %w", err)
	}

	return doc.Save(outputPath)
}
