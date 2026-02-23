package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execProjectRemove(homeDir string, identifier string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := projectRemoveCmd
	cmd.SetOut(stdout)
	err := runProjectRemove(cmd, homeDir, identifier, confirm)
	return stdout.String(), err
}

func TestProjectRemoveNoRepos(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout, err := execProjectRemove(home, "My Project", AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' removed")

	// Verify removed from registry
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)
	assert.Empty(t, reg.Projects)
}

func TestProjectRemoveWithReposConfirmed(t *testing.T) {
	home := t.TempDir()

	// Create project and assign a repo
	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	repo := t.TempDir()
	hooksDir := filepath.Join(repo, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\n# Installed by hourgit\necho hourgit\n"), 0755))
	require.NoError(t, project.AssignProject(home, repo, entry))

	stdout, err := execProjectRemove(home, "My Project", AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' removed")

	// Verify repo config cleaned up
	cfg, err := project.ReadRepoConfig(repo)
	assert.NoError(t, err)
	assert.Nil(t, cfg)

	// Verify hook cleaned up
	_, err = os.Stat(filepath.Join(hooksDir, "post-checkout"))
	assert.True(t, os.IsNotExist(err))
}

func TestProjectRemoveWithReposDeclined(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0755))
	require.NoError(t, project.AssignProject(home, repo, entry))

	decline := func(_ string) (bool, error) { return false, nil }
	_, err = execProjectRemove(home, "My Project", decline)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")

	// Verify project still exists
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)
}

func TestProjectRemoveNotFound(t *testing.T) {
	home := t.TempDir()

	_, err := execProjectRemove(home, "nonexistent", AlwaysYes())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProjectRemoveByID(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout, err := execProjectRemove(home, entry.ID, AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' removed")
}

func TestProjectRemoveMissingRepo(t *testing.T) {
	home := t.TempDir()

	// Create project with a repo path that doesn't exist on disk
	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	// Manually add a non-existent repo path
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)
	reg.Projects[0].Repos = []string{"/nonexistent/repo"}
	require.NoError(t, project.WriteRegistry(home, reg))
	entry.Repos = reg.Projects[0].Repos

	stdout, err := execProjectRemove(home, "My Project", AlwaysYes())

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' removed")
}

func TestProjectRemoveRegisteredAsSubcommand(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "remove")
}
