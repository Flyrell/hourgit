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

func execDefaultsScheduleReport(homeDir, monthFlag, yearFlag string, now time.Time) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsScheduleReportCmd
	cmd.SetOut(stdout)
	err := runDefaultsScheduleReport(cmd, homeDir, monthFlag, yearFlag, now)
	return stdout.String(), err
}

func TestDefaultsScheduleReportFactoryDefaults(t *testing.T) {
	homeDir := t.TempDir()
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	stdout, err := execDefaultsScheduleReport(homeDir, "", "", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Default working hours")
	assert.Contains(t, stdout, "February 2026")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	// Weekdays should appear
	assert.Contains(t, stdout, "Mon Feb  2")
	// Weekends should not
	assert.NotContains(t, stdout, "Sat ")
	assert.NotContains(t, stdout, "Sun ")
}

func TestDefaultsScheduleReportCustomDefaults(t *testing.T) {
	homeDir := t.TempDir()
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "10:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	stdout, err := execDefaultsScheduleReport(homeDir, "", "", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "10:00 AM - 2:00 PM")
	// Daily schedule — should include weekends
	assert.Contains(t, stdout, "Sat ")
	assert.Contains(t, stdout, "Sun ")
}

func TestDefaultsScheduleReportWithMonthAndYearFlags(t *testing.T) {
	homeDir := t.TempDir()
	now := time.Date(2026, 2, 15, 12, 0, 0, 0, time.UTC)

	stdout, err := execDefaultsScheduleReport(homeDir, "3", "2025", now)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "March 2025")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
}

func TestDefaultsScheduleReportRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsScheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "report")
}
