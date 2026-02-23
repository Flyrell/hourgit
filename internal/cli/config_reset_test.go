package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/Flyrell/hour-git/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execConfigReset(homeDir, repoDir, projectFlag string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := configResetCmd
	cmd.SetOut(stdout)
	err := runConfigReset(cmd, homeDir, repoDir, projectFlag, confirm)
	return stdout.String(), err
}

func TestConfigResetHappyPath(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	// Set a custom schedule first
	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	stdout, err := execConfigReset(homeDir, repoDir, "", AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "reset to default")

	// Verify schedules are back to default
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Equal(t, schedule.DefaultSchedules(), schedules)
}

func TestConfigResetDeclined(t *testing.T) {
	homeDir, repoDir, entry := setupConfigTest(t)

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetSchedules(homeDir, entry.ID, custom))

	decline := func(_ string) (bool, error) { return false, nil }
	_, err := execConfigReset(homeDir, repoDir, "", decline)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")

	// Verify schedule unchanged
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	schedules := project.GetSchedules(cfg, entry.ID)
	assert.Equal(t, custom, schedules)
}

func TestConfigResetByProjectFlag(t *testing.T) {
	homeDir, _, entry := setupConfigTest(t)

	stdout, err := execConfigReset(homeDir, "", entry.Name, AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "reset to default")
}

func TestConfigResetNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execConfigReset(homeDir, "", "", AlwaysYes())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestConfigResetRegisteredAsSubcommand(t *testing.T) {
	commands := configCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "reset")
}
