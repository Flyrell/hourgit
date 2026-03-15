package cli

import (
	"bytes"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execDefaultsScheduleSet(homeDir string, kit PromptKit) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := defaultsScheduleSetCmd
	cmd.SetOut(stdout)
	err := runDefaultsScheduleSet(cmd, homeDir, kit)
	return stdout.String(), err
}

func TestDefaultsScheduleSetQuitImmediately(t *testing.T) {
	homeDir := t.TempDir()

	kit := testKit(
		mockSelect(3), // Save & quit
		mockPrompt(),
		mockConfirm(false),
		mockMultiSelect(nil),
	)
	stdout, err := execDefaultsScheduleSet(homeDir, kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")
}

func TestDefaultsScheduleSetAddAndQuit(t *testing.T) {
	homeDir := t.TempDir()

	// Add(0): recurring(0) > every weekend(1) > 10am-2pm > no more ranges
	// Save&quit(3)
	kit := testKit(
		mockSelectSequence(0, 0, 1, 3),
		mockPrompt("10am", "2pm"),
		mockConfirmSequence(false, false), // no more ranges, no overlap
		mockMultiSelect(nil),
	)
	stdout, err := execDefaultsScheduleSet(homeDir, kit)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "saved")

	// Verify defaults were updated
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	defaults := project.GetDefaults(cfg)
	assert.Len(t, defaults, 2) // original default + new
}

func TestDefaultsScheduleSetRegisteredAsSubcommand(t *testing.T) {
	commands := defaultsScheduleCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "set")
}
