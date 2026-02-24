package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func execCompletion(shell string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := newCompletionGenerateCmd()
	cmd.SetOut(stdout)
	err := runCompletion(cmd, shell)
	return stdout.String(), err
}

func TestCompletionBash(t *testing.T) {
	stdout, err := execCompletion("bash")
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout)
	assert.Contains(t, stdout, "bash")
}

func TestCompletionZsh(t *testing.T) {
	stdout, err := execCompletion("zsh")
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestCompletionFish(t *testing.T) {
	stdout, err := execCompletion("fish")
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestCompletionPowershell(t *testing.T) {
	stdout, err := execCompletion("powershell")
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestCompletionInvalidShell(t *testing.T) {
	_, err := execCompletion("invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell: invalid")
}

func TestCompletionAutoDetect(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	stdout := new(bytes.Buffer)
	cmd := newCompletionGenerateCmd()
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.NoError(t, err)
	assert.NotEmpty(t, stdout.String())
}

func TestCompletionAutoDetectUnknown(t *testing.T) {
	t.Setenv("SHELL", "/bin/csh")
	stdout := new(bytes.Buffer)
	cmd := newCompletionGenerateCmd()
	cmd.SetOut(stdout)
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "could not detect shell")
}

func TestCompletionRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "completion")

	// Verify completion is a group with generate and install subcommands
	var completionGroup *cobra.Command
	for _, cmd := range commands {
		if cmd.Name() == "completion" {
			completionGroup = cmd
			break
		}
	}
	assert.NotNil(t, completionGroup)
	subNames := make([]string, len(completionGroup.Commands()))
	for i, sub := range completionGroup.Commands() {
		subNames[i] = sub.Name()
	}
	assert.Contains(t, subNames, "generate")
	assert.Contains(t, subNames, "install")
}
