package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupInitTest(t *testing.T) (string, func()) {
	t.Helper()

	orig, err := os.Getwd()
	require.NoError(t, err)

	dir := t.TempDir()
	require.NoError(t, os.Chdir(dir))

	return dir, func() {
		require.NoError(t, os.Chdir(orig))
	}
}

func execInit(args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(append([]string{"init"}, args...))
	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

func TestInitInGitRepo(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	stdout, _, err := execInit()

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	hookPath := filepath.Join(dir, ".git", "hooks", "post-checkout")
	content, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), hookMarker)
	assert.Contains(t, string(content), "#!/bin/sh")

	info, err := os.Stat(hookPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestInitNotGitRepo(t *testing.T) {
	_, cleanup := setupInitTest(t)
	defer cleanup()

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "error: not a git repository")
}

func TestInitAlreadyInitialized(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte(hookContent), 0755))

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "error: hourgit is already initialized")
}

func TestInitHookExistsNoFlag(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "error: post-checkout hook already exists (use --force to overwrite or --merge to append)")
}

func TestInitHookExistsForce(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	stdout, _, err := execInit("--force")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	content, err := os.ReadFile(filepath.Join(hooksDir, "post-checkout"))
	require.NoError(t, err)
	assert.Contains(t, string(content), hookMarker)
	assert.NotContains(t, string(content), "echo existing")
}

func TestInitHookExistsMerge(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	stdout, _, err := execInit("--merge")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	content, err := os.ReadFile(filepath.Join(hooksDir, "post-checkout"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "echo existing")
	assert.Contains(t, string(content), hookMarker)
}

func TestInitWithProjectFlag(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	stdout, _, err := execInit("--project", "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project: My Project (not yet implemented)")
	assert.Contains(t, stdout, "hourgit initialized successfully")
}

func TestInitCreateHooksDir(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	_, _, err := execInit()

	assert.NoError(t, err)

	hookPath := filepath.Join(dir, ".git", "hooks", "post-checkout")
	_, err = os.Stat(hookPath)
	assert.NoError(t, err)
}

func TestInitRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "init")
}
