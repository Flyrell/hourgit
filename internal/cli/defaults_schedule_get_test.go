package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDefaultsScheduleGet(homeDir string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsScheduleGetCmd
	cmd.SetOut(stdout)
	err := runDefaultsScheduleGet(cmd, homeDir)
	return stdout.String(), err
}

func TestDefaultsScheduleGetFactoryDefaults(t *testing.T) {
	homeDir := t.TempDir()

	stdout, err := execDefaultsScheduleGet(homeDir)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Default schedule for new projects")
	assert.Contains(t, stdout, "9:00 AM - 5:00 PM")
	assert.Contains(t, stdout, "every weekday")
}

func TestDefaultsScheduleGetCustomDefaults(t *testing.T) {
	homeDir := t.TempDir()

	custom := []schedule.ScheduleEntry{
		{Ranges: []schedule.TimeRange{{From: "08:00", To: "12:00"}}, RRule: "FREQ=WEEKLY;BYDAY=MO,WE,FR"},
	}
	require.NoError(t, project.SetDefaults(homeDir, custom))

	stdout, err := execDefaultsScheduleGet(homeDir)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "8:00 AM - 12:00 PM")
}

func TestDefaultsScheduleGetRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsScheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "get")
}

func TestDefaultsScheduleRegisteredUnderDefaults(t *testing.T) {
	commands := defaultsCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "schedule")
}
