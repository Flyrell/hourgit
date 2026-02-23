package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/Flyrell/hour-git/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupConfigTest(t *testing.T) (homeDir string, repoDir string, entry *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	entry, err := project.CreateProject(homeDir, "Test Project")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, entry))

	// Re-read to get updated entry
	reg, err := project.ReadRegistry(homeDir)
	require.NoError(t, err)
	entry = project.FindProjectByID(reg, entry.ID)

	return homeDir, repoDir, entry
}

func execConfigGet(homeDir, repoDir, projectFlag string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := configGetCmd
	cmd.SetOut(stdout)
	err := runConfigGet(cmd, homeDir, repoDir, projectFlag)
	return stdout.String(), err
}

func TestConfigGetDefaultSchedule(t *testing.T) {
	homeDir, repoDir, _ := setupConfigTest(t)

	stdout, err := execConfigGet(homeDir, repoDir, "")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Schedule for")
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	assert.Contains(t, stdout, "every weekday")
}

func TestConfigGetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupConfigTest(t)

	stdout, err := execConfigGet(homeDir, "", entry.Name)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Test Project")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
}

func TestConfigGetCustomSchedule(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "08:00", To: "12:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO,WE,FR"},
		{Ranges: []schedule.TimeRange{{From: "13:00", To: "17:00"}}, RRule: "FREQ=WEEKLY;BYDAY=TU,TH"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	stdout, err := execConfigGet(homeDir, repoDir, "")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "8:00 AM - 12:00 PM")
	assert.Contains(t, stdout, "1:00 PM - 5:00 PM")
	assert.Contains(t, stdout, "1.")
	assert.Contains(t, stdout, "2.")
}

func TestConfigGetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execConfigGet(homeDir, "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestConfigGetRegisteredAsSubcommand(t *testing.T) {
	commands := configCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "get")
}
