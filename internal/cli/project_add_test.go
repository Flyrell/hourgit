package cli

import (
	"bytes"
	"os"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execProjectAdd(homeDir string, name string, mode ...string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := projectAddCmd
	cmd.SetOut(stdout)
	m := ""
	if len(mode) > 0 {
		m = mode[0]
	}
	err := runProjectAdd(cmd, homeDir, name, m, "/usr/local/bin/hourgit")
	return stdout.String(), err
}

func TestProjectAddHappyPath(t *testing.T) {
	home := t.TempDir()

	stdout, err := execProjectAdd(home, "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' created (")

	// Verify config
	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, cfg.Projects, 1)
	assert.Equal(t, "My Project", cfg.Projects[0].Name)
	assert.Empty(t, cfg.Projects[0].Repos)

	// Verify log dir
	_, err = os.Stat(project.LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestProjectAddDuplicate(t *testing.T) {
	home := t.TempDir()

	_, err := execProjectAdd(home, "My Project")
	require.NoError(t, err)

	_, err = execProjectAdd(home, "My Project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectAddModePrecise(t *testing.T) {
	home := t.TempDir()

	stdout, err := execProjectAdd(home, "Precise Project", "precise")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'Precise Project' created (")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	require.Len(t, cfg.Projects, 1)
	assert.True(t, cfg.Projects[0].Precise)
	assert.Equal(t, project.DefaultIdleThresholdMinutes, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectAddModeStandard(t *testing.T) {
	home := t.TempDir()

	stdout, err := execProjectAdd(home, "Standard Project", "standard")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'Standard Project' created (")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	require.Len(t, cfg.Projects, 1)
	assert.False(t, cfg.Projects[0].Precise)
	assert.Equal(t, 0, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectAddModeInvalid(t *testing.T) {
	home := t.TempDir()

	_, err := execProjectAdd(home, "Bad Mode", "foobar")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --mode value")
}

func TestProjectAddRegisteredAsSubcommand(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "add")
}
