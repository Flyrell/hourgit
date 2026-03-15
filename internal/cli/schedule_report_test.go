package cli

import (
	"bytes"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execScheduleReport(homeDir, repoDir, projectFlag, monthFlag, yearFlag string, now time.Time) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := scheduleReportCmd
	cmd.SetOut(stdout)
	err := runScheduleReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, now)
	return stdout.String(), err
}

func TestScheduleReportDefaultSchedule(t *testing.T) {
	homeDir, repoDir, _ := setupScheduleTest(t)
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC) // mid-February

	stdout, err := execScheduleReport(homeDir, repoDir, "", "", "", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Working hours for")
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "February 2026")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	// Feb 2026 has 20 weekdays
	assert.Contains(t, stdout, "Mon Feb  2")
	assert.Contains(t, stdout, "Fri Feb 27" /* space-padded by Go's time format */)
	// Weekends should not appear
	assert.NotContains(t, stdout, "Sat ")
	assert.NotContains(t, stdout, "Sun ")
}

func TestScheduleReportByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupScheduleTest(t)
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	stdout, err := execScheduleReport(homeDir, "", entry.Name, "", "", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
}

func TestScheduleReportMultipleWindows(t *testing.T) {
	homeDir, repoDir, entry := setupScheduleTest(t)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "09:00", To: "12:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO"},
		{Ranges: []schedule.TimeRange{{From: "13:00", To: "17:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	stdout, err := execScheduleReport(homeDir, repoDir, "", "", "", now)

	assert.NoError(t, err)
	// Default is accumulate: both windows appear comma-separated
	assert.Contains(t, stdout, "9:00 AM - 12:00 PM, 1:00 PM - 5:00 PM")
}

func TestScheduleReportNoProject(t *testing.T) {
	homeDir := t.TempDir()
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	_, err := execScheduleReport(homeDir, "", "", "", "", now)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestScheduleReportNoWorkingHours(t *testing.T) {
	homeDir, repoDir, entry := setupScheduleTest(t)

	// Set schedule to a specific date outside the test month
	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "09:00", To: "17:00"}}, RRule: "DTSTART:20260315T000000Z\nRRULE:FREQ=DAILY;COUNT=1"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)
	stdout, err := execScheduleReport(homeDir, repoDir, "", "", "", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "No working hours scheduled this month")
}

func TestScheduleReportWithMonthAndYearFlags(t *testing.T) {
	homeDir, repoDir, _ := setupScheduleTest(t)
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	stdout, err := execScheduleReport(homeDir, repoDir, "", "1", "2025", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "January 2025")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
}

func TestScheduleReportRegisteredAsSubcommand(t *testing.T) {
	commands := scheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "report")
}
