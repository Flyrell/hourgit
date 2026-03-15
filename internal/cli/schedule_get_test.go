package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupScheduleTest(t *testing.T) (homeDir string, repoDir string, entry *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	entry, err := project.CreateProject(homeDir, "Test Project")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, entry))

	// Re-read to get updated entry
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	entry = project.FindProjectByID(cfg, entry.ID)

	return homeDir, repoDir, entry
}

func execScheduleGet(homeDir, repoDir, projectFlag string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := scheduleGetCmd
	cmd.SetOut(stdout)
	err := runScheduleGet(cmd, homeDir, repoDir, projectFlag)
	return stdout.String(), err
}

func TestScheduleGetDefaultSchedule(t *testing.T) {
	homeDir, repoDir, _ := setupScheduleTest(t)

	stdout, err := execScheduleGet(homeDir, repoDir, "")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Schedule for")
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	assert.Contains(t, stdout, "every weekday")
}

func TestScheduleGetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupScheduleTest(t)

	stdout, err := execScheduleGet(homeDir, "", entry.Name)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
}

func TestScheduleGetCustomSchedule(t *testing.T) {
	homeDir, repoDir, entry := setupScheduleTest(t)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "08:00", To: "12:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO,WE,FR"},
		{Ranges: []schedule.TimeRange{{From: "13:00", To: "17:00"}}, RRule: "FREQ=WEEKLY;BYDAY=TU,TH"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	stdout, err := execScheduleGet(homeDir, repoDir, "")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "8:00 AM - 12:00 PM")
	assert.Contains(t, stdout, "1:00 PM - 5:00 PM")
	assert.Contains(t, stdout, "1.")
	assert.Contains(t, stdout, "2.")
}

func TestScheduleGetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execScheduleGet(homeDir, "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestScheduleGetRegisteredAsSubcommand(t *testing.T) {
	commands := scheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "get")
}

func TestScheduleRegisteredUnderProject(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "schedule")
}
