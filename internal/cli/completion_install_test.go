package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectShellZsh(t *testing.T) {
	t.Setenv("SHELL", "/bin/zsh")
	assert.Equal(t, "zsh", detectShell())
}

func TestDetectShellBash(t *testing.T) {
	t.Setenv("SHELL", "/bin/bash")
	assert.Equal(t, "bash", detectShell())
}

func TestDetectShellFish(t *testing.T) {
	t.Setenv("SHELL", "/usr/local/bin/fish")
	assert.Equal(t, "fish", detectShell())
}

func TestDetectShellUnknown(t *testing.T) {
	t.Setenv("SHELL", "/bin/csh")
	assert.Equal(t, "", detectShell())
}

func TestDetectShellEmpty(t *testing.T) {
	t.Setenv("SHELL", "")
	assert.Equal(t, "", detectShell())
}

func TestIsCompletionInstalledTrue(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(zshrc, []byte("# existing config\neval \"$(hourgit completion generate zsh)\"\n"), 0644))

	assert.True(t, isCompletionInstalled("zsh", home))
}

func TestIsCompletionInstalledFalse(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(zshrc, []byte("# existing config\n"), 0644))

	assert.False(t, isCompletionInstalled("zsh", home))
}

func TestIsCompletionInstalledNoFile(t *testing.T) {
	home := t.TempDir()
	assert.False(t, isCompletionInstalled("zsh", home))
}

func TestIsCompletionInstalledUnsupportedShell(t *testing.T) {
	home := t.TempDir()
	assert.False(t, isCompletionInstalled("csh", home))
}

func TestInstallCompletionZsh(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	require.NoError(t, os.WriteFile(zshrc, []byte("# existing config\n"), 0644))

	err := installCompletion("zsh", home)
	assert.NoError(t, err)

	data, err := os.ReadFile(zshrc)
	require.NoError(t, err)
	assert.Contains(t, string(data), "# existing config")
	assert.Contains(t, string(data), `eval "$(hourgit completion generate zsh)"`)
	assert.Contains(t, string(data), "# hourgit shell completion")
}

func TestInstallCompletionBash(t *testing.T) {
	home := t.TempDir()

	err := installCompletion("bash", home)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(home, ".bashrc"))
	require.NoError(t, err)
	assert.Contains(t, string(data), `eval "$(hourgit completion generate bash)"`)
}

func TestInstallCompletionFish(t *testing.T) {
	home := t.TempDir()

	err := installCompletion("fish", home)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(home, ".config", "fish", "config.fish"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "hourgit completion generate fish | source")
}

func TestInstallCompletionPowershell(t *testing.T) {
	home := t.TempDir()

	err := installCompletion("powershell", home)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(home, ".config", "powershell", "Microsoft.PowerShell_profile.ps1"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "hourgit completion generate powershell | Out-String | Invoke-Expression")
}

func TestInstallCompletionAlreadyInstalled(t *testing.T) {
	home := t.TempDir()
	zshrc := filepath.Join(home, ".zshrc")
	content := "# existing config\neval \"$(hourgit completion generate zsh)\"\n"
	require.NoError(t, os.WriteFile(zshrc, []byte(content), 0644))

	err := installCompletion("zsh", home)
	assert.NoError(t, err)

	// File should be unchanged
	data, err := os.ReadFile(zshrc)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestInstallCompletionUnsupportedShell(t *testing.T) {
	home := t.TempDir()
	err := installCompletion("csh", home)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
}
