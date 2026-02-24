package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execCompletionInstall(shell, homeDir string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := &cobra.Command{Use: "test"}
	cmd.SetOut(stdout)
	err := runCompletionInstall(cmd, shell, homeDir, confirm)
	return stdout.String(), err
}

func TestCompletionInstallHappyPath(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# existing\n"), 0644))

	t.Setenv("SHELL", "/bin/zsh")
	stdout, err := execCompletionInstall("zsh", home, AlwaysYes())
	assert.NoError(t, err)
	assert.Contains(t, stdout, "shell completions installed")
	assert.Contains(t, stdout, "zsh")

	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `eval "$(hourgit completion generate zsh)"`)
}

func TestCompletionInstallAlreadyInstalled(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, ".zshrc"), []byte("eval \"$(hourgit completion generate zsh)\"\n"), 0644))

	stdout, err := execCompletionInstall("zsh", home, AlwaysYes())
	assert.NoError(t, err)
	assert.Contains(t, stdout, "already installed")
}

func TestCompletionInstallExplicitShell(t *testing.T) {
	home := t.TempDir()

	stdout, err := execCompletionInstall("bash", home, AlwaysYes())
	assert.NoError(t, err)
	assert.Contains(t, stdout, "shell completions installed")
	assert.Contains(t, stdout, "bash")

	data, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `eval "$(hourgit completion generate bash)"`)
}

func TestCompletionInstallDeclined(t *testing.T) {
	home := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(home, ".zshrc"), []byte("# existing\n"), 0644))

	declined := func(msg string) (bool, error) { return false, nil }
	stdout, err := execCompletionInstall("zsh", home, declined)
	assert.NoError(t, err)
	assert.Empty(t, stdout)

	// File should be unchanged
	data, err := os.ReadFile(filepath.Join(home, ".zshrc"))
	require.NoError(t, err)
	assert.Equal(t, "# existing\n", string(data))
}

func TestCompletionInstallUnsupportedShell(t *testing.T) {
	home := t.TempDir()

	_, err := execCompletionInstall("csh", home, AlwaysYes())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}
