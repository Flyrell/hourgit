package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execScheduleReset(homeDir, repoDir, projectFlag string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := scheduleResetCmd
	cmd.SetOut(stdout)
	err := runScheduleReset(cmd, homeDir, repoDir, projectFlag, confirm)
	return stdout.String(), err
}

func TestScheduleResetHappyPath(t *testing.T) {
	homeDir, repoDir, entry := setupScheduleTest(t)

	// Set a custom schedule first
	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	stdout, err := execScheduleReset(homeDir, repoDir, "", AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "reset to default")

	// Verify schedules are back to default
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Equal(t, schedule.DefaultSchedules(), schedules)
}

func TestScheduleResetDeclined(t *testing.T) {
	homeDir, repoDir, entry := setupScheduleTest(t)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	decline := func(_ string) (bool, error) { return false, nil }
	stdout, err := execScheduleReset(homeDir, repoDir, "", decline)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "cancelled")

	// Verify schedule unchanged
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Equal(t, custom, schedules)
}

func TestScheduleResetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupScheduleTest(t)

	stdout, err := execScheduleReset(homeDir, "", entry.Name, AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "reset to default")
}

func TestScheduleResetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execScheduleReset(homeDir, "", "", AlwaysYes())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestScheduleResetRegisteredAsSubcommand(t *testing.T) {
	commands := scheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "reset")
}
