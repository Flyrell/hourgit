package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDefaultsGet(homeDir string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsGetCmd
	cmd.SetOut(stdout)
	err := runDefaultsGet(cmd, homeDir)
	return stdout.String(), err
}

func TestDefaultsGetFactoryDefaults(t *testing.T) {
	homeDir := t.TempDir()

	stdout, err := execDefaultsGet(homeDir)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Default schedule for new projects")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	assert.Contains(t, stdout, "every weekday")
}

func TestDefaultsGetCustomDefaults(t *testing.T) {
	homeDir := t.TempDir()

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "08:00", To: "12:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO,WE,FR"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	stdout, err := execDefaultsGet(homeDir)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "8:00 AM - 12:00 PM")
}

func TestDefaultsGetRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "get")
}
