package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDefaultsScheduleReset(homeDir string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsScheduleResetCmd
	cmd.SetOut(stdout)
	err := runDefaultsScheduleReset(cmd, homeDir, confirm)
	return stdout.String(), err
}

func TestDefaultsScheduleResetHappyPath(t *testing.T) {
	homeDir := t.TempDir()

	// Set custom defaults first
	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	stdout, err := execDefaultsScheduleReset(homeDir, AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "defaults reset to factory settings")

	// Verify defaults are back to factory
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Equal(t, schedule.DefaultSchedules(), project.GetDefaults(cfg))
}

func TestDefaultsScheduleResetDeclined(t *testing.T) {
	homeDir := t.TempDir()

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	decline := func(_ string) (bool, error) { return false, nil }
	stdout, err := execDefaultsScheduleReset(homeDir, decline)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "cancelled")

	// Verify defaults unchanged
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Equal(t, custom, project.GetDefaults(cfg))
}

func TestDefaultsScheduleResetRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsScheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "reset")
}
