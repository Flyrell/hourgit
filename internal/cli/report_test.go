package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
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

	err := runReport(cmd, homeDir, repoDir, projectFlag, monthFlag, "", yearFlag, "", false, false, false, fixedNow)
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

func TestParseReportDateRange_DefaultMonth(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	from, to, year, month, err := parseReportDateRange("", "", "", false, false, false, now)
	require.NoError(t, err)
	assert.Equal(t, 2025, year)
	assert.Equal(t, time.March, month)
	assert.Equal(t, "2025-03-01", from.Format("2006-01-02"))
	assert.Equal(t, "2025-03-31", to.Format("2006-01-02"))
}

func TestParseReportDateRange_ExplicitMonth(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	from, to, year, month, err := parseReportDateRange("6", "", "", true, false, false, now)
	require.NoError(t, err)
	assert.Equal(t, 2025, year)
	assert.Equal(t, time.June, month)
	assert.Equal(t, "2025-06-01", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-30", to.Format("2006-01-02"))
}

func TestParseReportDateRange_MonthWithYear(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	from, to, year, month, err := parseReportDateRange("2", "", "2024", true, false, true, now)
	require.NoError(t, err)
	assert.Equal(t, 2024, year)
	assert.Equal(t, time.February, month)
	assert.Equal(t, "2024-02-01", from.Format("2006-01-02"))
	assert.Equal(t, "2024-02-29", to.Format("2006-01-02")) // 2024 is leap year
}

func TestParseReportDateRange_WeekCurrent(t *testing.T) {
	// June 15, 2025 is a Sunday, ISO week 24
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to, _, _, err := parseReportDateRange("", "", "", false, true, false, now)
	require.NoError(t, err)

	// Week 24 of 2025: Mon Jun 9 - Sun Jun 15
	assert.Equal(t, time.Monday, from.Weekday())
	assert.Equal(t, time.Sunday, to.Weekday())
	assert.Equal(t, "2025-06-09", from.Format("2006-01-02"))
	assert.Equal(t, "2025-06-15", to.Format("2006-01-02"))
}

func TestParseReportDateRange_WeekExplicit(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to, _, _, err := parseReportDateRange("", "1", "", false, true, false, now)
	require.NoError(t, err)

	// Week 1 of 2025
	assert.Equal(t, time.Monday, from.Weekday())
	assert.Equal(t, time.Sunday, to.Weekday())
}

func TestParseReportDateRange_WeekWithYear(t *testing.T) {
	now := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	from, to, _, _, err := parseReportDateRange("", "22", "2025", false, true, true, now)
	require.NoError(t, err)
	assert.Equal(t, time.Monday, from.Weekday())
	assert.Equal(t, time.Sunday, to.Weekday())
}

func TestParseReportDateRange_MonthAndWeekError(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseReportDateRange("3", "10", "", true, true, false, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be used together")
}

func TestParseReportDateRange_YearAloneError(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseReportDateRange("", "", "2024", false, false, true, now)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be used with")
}

func TestParseReportDateRange_InvalidWeek(t *testing.T) {
	now := time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC)
	_, _, _, _, err := parseReportDateRange("", "0", "", false, true, false, now)
	assert.Error(t, err)

	_, _, _, _, err = parseReportDateRange("", "54", "", false, true, false, now)
	assert.Error(t, err)

	_, _, _, _, err = parseReportDateRange("", "abc", "", false, true, false, now)
	assert.Error(t, err)
}

func TestIsoWeekStart(t *testing.T) {
	t.Run("week 1 of 2025", func(t *testing.T) {
		monday := isoWeekStart(2025, 1)
		isoYear, isoWeek := monday.ISOWeek()
		assert.Equal(t, 2025, isoYear)
		assert.Equal(t, 1, isoWeek)
		assert.Equal(t, time.Monday, monday.Weekday())
	})

	t.Run("week 52 of 2025", func(t *testing.T) {
		monday := isoWeekStart(2025, 52)
		isoYear, isoWeek := monday.ISOWeek()
		assert.Equal(t, 2025, isoYear)
		assert.Equal(t, 52, isoWeek)
		assert.Equal(t, time.Monday, monday.Weekday())
	})

	t.Run("week 24 of 2025", func(t *testing.T) {
		monday := isoWeekStart(2025, 24)
		assert.Equal(t, time.Monday, monday.Weekday())
		assert.Equal(t, "2025-06-09", monday.Format("2006-01-02"))
	})
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

	e := entry.Entry{
		ID:        "1e50010",
		Start:     time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		Minutes:   120,
		Message:   "research",
		Task:      "research",
		CreatedAt: time.Date(2025, 6, 2, 12, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

	now := time.Date(2025, 7, 1, 0, 0, 0, 0, time.UTC)
	inputs, err := loadReportInputs(homeDir, repoDir, "", "6", "", "2025", true, false, true, now)
	require.NoError(t, err)

	data := timetrack.BuildDetailedReport(inputs.checkouts, inputs.logs, inputs.schedules, inputs.from, inputs.to, now)
	assert.Equal(t, 1, len(data.Rows))
	assert.Equal(t, "research", data.Rows[0].Name)
	assert.Equal(t, 120, data.Rows[0].TotalMinutes)
}

func TestIsSubmitted(t *testing.T) {
	from := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)

	t.Run("no submits", func(t *testing.T) {
		assert.False(t, isSubmitted(nil, from, to))
	})

	t.Run("overlapping submit", func(t *testing.T) {
		submits := []entry.SubmitEntry{
			{From: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 1, 31, 0, 0, 0, 0, time.UTC)},
		}
		assert.True(t, isSubmitted(submits, from, to))
	})

	t.Run("non-overlapping submit", func(t *testing.T) {
		submits := []entry.SubmitEntry{
			{From: time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC), To: time.Date(2025, 2, 28, 0, 0, 0, 0, time.UTC)},
		}
		assert.False(t, isSubmitted(submits, from, to))
	})
}

func execReportWithOutput(t *testing.T, homeDir, repoDir, monthFlag, yearFlag, outputFlag string) (string, error) {
	t.Helper()
	stdout := new(bytes.Buffer)

	cmd := LeafCommand{
		Use:   "report",
		Short: "Generate a monthly time report",
		StrFlags: []StringFlag{
			{Name: "month", Usage: "month number 1-12 (default: current)"},
			{Name: "week", Usage: "ISO week number"},
			{Name: "year", Usage: "year"},
			{Name: "project", Usage: "project name or ID"},
			{Name: "output", Usage: "export report as PDF"},
		},
		RunE: func(c *cobra.Command, args []string) error {
			of, _ := c.Flags().GetString("output")
			mf, _ := c.Flags().GetString("month")
			yf, _ := c.Flags().GetString("year")
			mc := c.Flags().Changed("month")
			yc := c.Flags().Changed("year")
			return runReport(c, homeDir, repoDir, "", mf, "", yf, of, mc, false, yc, fixedNow)
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
		ID:        "0df0010",
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
		ID:        "a010010",
		Start:     time.Date(2025, 6, 2, 10, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "work",
		Task:      "task",
		CreatedAt: time.Date(2025, 6, 2, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, e))

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
