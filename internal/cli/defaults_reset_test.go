package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDefaultsReset(homeDir string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsResetCmd
	cmd.SetOut(stdout)
	err := runDefaultsReset(cmd, homeDir, confirm)
	return stdout.String(), err
}

func TestDefaultsResetHappyPath(t *testing.T) {
	homeDir := t.TempDir()

	// Set custom defaults first
	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	stdout, err := execDefaultsReset(homeDir, AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "defaults reset to factory settings")

	// Verify defaults are back to factory
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Equal(t, schedule.DefaultSchedules(), project.GetDefaults(cfg))
}

func TestDefaultsResetDeclined(t *testing.T) {
	homeDir := t.TempDir()

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "06:00", To: "14:00"}}, RRule: "FREQ=DAILY"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	decline := func(_ string) (bool, error) { return false, nil }
	stdout, err := execDefaultsReset(homeDir, decline)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "cancelled")

	// Verify defaults unchanged
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Equal(t, custom, project.GetDefaults(cfg))
}

func TestDefaultsResetRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "reset")
}
