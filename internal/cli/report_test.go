package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/timetrack"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupReportTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Report Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, repoDir, proj
}

func execReport(homeDir, repoDir, projectFlag, monthFlag, yearFlag string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := reportCmd
	cmd.SetOut(stdout)

	err := runReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, "", fixedNow)
	return stdout.String(), err
}

func TestParseMonthYearFlags_Default(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	year, month, err := parseMonthYearFlags("", "", now)
	require.NoError(t, err)
	assert.Equal(t, 2025, year)
	assert.Equal(t, time.March, month)
}

func TestParseMonthYearFlags_ValidMonth(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	year, month, err := parseMonthYearFlags("1", "", now)
	require.NoError(t, err)
	assert.Equal(t, 2025, year)
	assert.Equal(t, time.January, month)
}

func TestParseMonthYearFlags_ValidYear(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	year, month, err := parseMonthYearFlags("", "2024", now)
	require.NoError(t, err)
	assert.Equal(t, 2024, year)
	assert.Equal(t, time.March, month)
}

func TestParseMonthYearFlags_ValidMonthAndYear(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	year, month, err := parseMonthYearFlags("6", "2024", now)
	require.NoError(t, err)
	assert.Equal(t, 2024, year)
	assert.Equal(t, time.June, month)
}

func TestParseMonthYearFlags_InvalidMonth(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

	_, _, err := parseMonthYearFlags("0", "", now)
	assert.Error(t, err)

	_, _, err = parseMonthYearFlags("13", "", now)
	assert.Error(t, err)

	_, _, err = parseMonthYearFlags("abc", "", now)
	assert.Error(t, err)
}

func TestParseMonthYearFlags_InvalidYear(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)

	_, _, err := parseMonthYearFlags("", "0", now)
	assert.Error(t, err)

	_, _, err = parseMonthYearFlags("", "-1", now)
	assert.Error(t, err)

	_, _, err = parseMonthYearFlags("", "abc", now)
	assert.Error(t, err)
}

func TestReportNoProject(t *testing.T) {
	homeDir := t.TempDir()
	_, err := execReport(homeDir, "", "", "", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestReportEmptyMonth(t *testing.T) {
	homeDir, repoDir, _ := setupReportTest(t)

	stdout, err := execReport(homeDir, repoDir, "", "6", "")
	require.NoError(t, err)
	assert.Contains(t, stdout, "No time entries")
}

func TestReportWithLogEntries(t *testing.T) {
	homeDir, repoDir, proj := setupReportTest(t)

	// Write a log entry for June 2025 (fixedNow month)
	e := entry.Entry{
		ID:        "test01",
		Start:     time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC), // Mon Jun 2
		Minutes:   120,
		Message:   "research",
		Task:      "research",
		CreatedAt: time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	// runReport outputs via bubbletea (interactive), which won't work in test.
	// Test loadReportInputs + BuildReport directly.
	now := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	inputs, err := loadReportInputs(homeDir, repoDir, "", "6", "2025", now)
	require.NoError(t, err)

	data := timetrack.BuildReport(inputs.checkouts, inputs.logs, inputs.schedules, inputs.year, inputs.month, now, inputs.genDays)
	assert.Equal(t, 1, len(data.Rows))
	assert.Equal(t, "research", data.Rows[0].Name)
	assert.Equal(t, 120, data.Rows[0].TotalMinutes)
}

func TestRenderTable(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2025,
		Month:       time.January,
		DaysInMonth: 31,
		Rows: []timetrack.TaskRow{
			{
				Name:         "feature-x",
				TotalMinutes: 600,
				Days:         map[int]int{2: 480, 3: 120},
			},
		},
	}

	output := renderTable(data, 0, 5, false)

	assert.Contains(t, output, "Task")
	assert.Contains(t, output, "feature-x")
	assert.Contains(t, output, "10h")  // total 600 min
	assert.Contains(t, output, "8h")   // day 2 = 480 min
	assert.Contains(t, output, "2h")   // day 3 = 120 min
	assert.Contains(t, output, ".")    // zero days show dots

	// Totals footer row
	assert.Contains(t, output, "Total")
	// Total row should show the same values (single task = totals match)
	lines := strings.Split(output, "\n")
	lastDataLine := ""
	for _, l := range lines {
		if strings.HasPrefix(l, "Total") || strings.Contains(l, "Total") {
			lastDataLine = l
		}
	}
	assert.NotEmpty(t, lastDataLine)
	assert.Contains(t, lastDataLine, "8h")
	assert.Contains(t, lastDataLine, "2h")
}

func TestRenderTable_WithFooter(t *testing.T) {
	data := timetrack.ReportData{
		Year:        2025,
		Month:       time.January,
		DaysInMonth: 31,
		Rows: []timetrack.TaskRow{
			{
				Name:         "task-a",
				TotalMinutes: 60,
				Days:         map[int]int{1: 60},
			},
		},
	}

	output := renderTable(data, 0, 3, true)
	assert.Contains(t, output, "January 2025")
	assert.Contains(t, output, "←/→ scroll")
	assert.Contains(t, output, "q quit")
}

func execReportWithOutput(t *testing.T, homeDir, repoDir, monthFlag, yearFlag, outputFlag string) (string, error) {
	t.Helper()
	stdout := new(bytes.Buffer)

	// Build a fresh report command to avoid sharing state across tests
	cmd := LeafCommand{
		Use:   "report",
		Short: "Generate a monthly time report",
		StrFlags: []StringFlag{
			{Name: "month", Usage: "month number 1-12 (default: current)"},
			{Name: "year", Usage: "year (default: current)"},
			{Name: "project", Usage: "project name or ID"},
			{Name: "output", Usage: "export report as PDF"},
		},
		RunE: func(c *cobra.Command, args []string) error {
			of, _ := c.Flags().GetString("output")
			mf, _ := c.Flags().GetString("month")
			yf, _ := c.Flags().GetString("year")
			return runReport(c, homeDir, repoDir, "", mf, yf, of, fixedNow)
		},
	}.Build()

	cmd.SetOut(stdout)

	cmdArgs := []string{}
	if monthFlag != "" {
		cmdArgs = append(cmdArgs, "--month", monthFlag)
	}
	if yearFlag != "" {
		cmdArgs = append(cmdArgs, "--year", yearFlag)
	}
	if outputFlag != "" {
		cmdArgs = append(cmdArgs, "--output", outputFlag)
	} else {
		cmdArgs = append(cmdArgs, "--output=")
	}
	cmd.SetArgs(cmdArgs)

	err := cmd.Execute()
	return stdout.String(), err
}

func TestReportOutputFlag_GeneratesPDF(t *testing.T) {
	homeDir, repoDir, proj := setupReportTest(t)

	e := entry.Entry{
		ID:        "pdf01",
		Start:     time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		Minutes:   120,
		Message:   "research",
		Task:      "research",
		CreatedAt: time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "test-output.pdf")

	stdout, err := execReportWithOutput(t, homeDir, repoDir, "6", "2025", outPath)
	require.NoError(t, err)
	assert.Contains(t, stdout, "Exported report to")

	info, sErr := os.Stat(outPath)
	require.NoError(t, sErr)
	assert.True(t, info.Size() > 0)
}

func TestReportOutputFlag_EmptyMonth(t *testing.T) {
	homeDir, repoDir, _ := setupReportTest(t)

	outDir := t.TempDir()
	outPath := filepath.Join(outDir, "empty.pdf")

	stdout, err := execReportWithOutput(t, homeDir, repoDir, "1", "2025", outPath)
	require.NoError(t, err)
	assert.Contains(t, stdout, "No time entries")

	_, sErr := os.Stat(outPath)
	assert.True(t, os.IsNotExist(sErr))
}

func TestReportOutputFlag_AutoName(t *testing.T) {
	homeDir, repoDir, proj := setupReportTest(t)

	e := entry.Entry{
		ID:        "auto01",
		Start:     time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "work",
		Task:      "task",
		CreatedAt: time.Date(2025, 6, 2, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	// Change to temp dir so auto-named file lands there
	origDir, _ := os.Getwd()
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	stdout, err := execReportWithOutput(t, homeDir, repoDir, "6", "2025", "")
	require.NoError(t, err)

	expectedName := fmt.Sprintf("%s-2025-06.pdf", proj.Slug)
	assert.Contains(t, stdout, expectedName)

	info, sErr := os.Stat(filepath.Join(tmpDir, expectedName))
	require.NoError(t, sErr)
	assert.True(t, info.Size() > 0)
}

func TestReportRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "report")
}
