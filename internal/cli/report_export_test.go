package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderExportPDF_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test.pdf")

	data := timetrack.ExportData{
		ProjectName:  "Test Project",
		Year:         2025,
		Month:        time.January,
		TotalMinutes: 330,
		Days: []timetrack.ExportDay{
			{
				Date:         time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
				TotalMinutes: 330,
				Groups: []timetrack.ExportTaskGroup{
					{
						Task:         "feature-auth",
						TotalMinutes: 210,
						Entries: []timetrack.ExportEntry{
							{Start: time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC), Minutes: 90, Message: "Login flow"},
							{Start: time.Date(2025, 1, 2, 11, 30, 0, 0, time.UTC), Minutes: 120, Message: "Token refresh"},
						},
					},
					{
						Task:         "API design research",
						TotalMinutes: 120,
						Entries: []timetrack.ExportEntry{
							{Start: time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC), Minutes: 120, Message: "API design research"},
						},
					},
				},
			},
		},
	}

	err := renderExportPDF(data, outPath)
	require.NoError(t, err)

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)
}

func TestRenderExportPDF_MultipleDays(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "multi.pdf")

	data := timetrack.ExportData{
		ProjectName:  "Multi Day Project",
		Year:         2025,
		Month:        time.February,
		TotalMinutes: 600,
		Days: []timetrack.ExportDay{
			{
				Date:         time.Date(2025, 2, 3, 0, 0, 0, 0, time.UTC),
				TotalMinutes: 300,
				Groups: []timetrack.ExportTaskGroup{
					{
						Task:         "feature-x",
						TotalMinutes: 300,
						Entries: []timetrack.ExportEntry{
							{Start: time.Date(2025, 2, 3, 9, 0, 0, 0, time.UTC), Minutes: 300, Message: "feature-x"},
						},
					},
				},
			},
			{
				Date:         time.Date(2025, 2, 4, 0, 0, 0, 0, time.UTC),
				TotalMinutes: 300,
				Groups: []timetrack.ExportTaskGroup{
					{
						Task:         "feature-y",
						TotalMinutes: 300,
						Entries: []timetrack.ExportEntry{
							{Start: time.Date(2025, 2, 4, 9, 0, 0, 0, time.UTC), Minutes: 300, Message: "feature-y"},
						},
					},
				},
			},
		},
	}

	err := renderExportPDF(data, outPath)
	require.NoError(t, err)

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)
}

func TestRenderExportPDF_EmptyData(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "empty.pdf")

	data := timetrack.ExportData{
		ProjectName:  "Empty Project",
		Year:         2025,
		Month:        time.January,
		TotalMinutes: 0,
		Days:         nil,
	}

	err := renderExportPDF(data, outPath)
	require.NoError(t, err)

	info, err := os.Stat(outPath)
	require.NoError(t, err)
	assert.True(t, info.Size() > 0)
}
