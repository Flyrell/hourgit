package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func execProjectList(homeDir string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := projectListCmd
	cmd.SetOut(stdout)
	err := runProjectList(cmd, homeDir)
	return stdout.String(), err
}

func TestProjectListEmpty(t *testing.T) {
	home := t.TempDir()

	stdout, err := execProjectList(home)

	assert.NoError(t, err)
	assert.Equal(t, "No projects found.\n", stdout)
}

func TestProjectListWithProjects(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create two repo dirs with hourgit hooks
	dir1 := t.TempDir()
	hooksDir1 := filepath.Join(dir1, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir1, "post-checkout"), []byte(hookContent), 0755))

	dir2 := t.TempDir()
	hooksDir2 := filepath.Join(dir2, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir2, "post-checkout"), []byte(hookContent), 0755))

	// Create two projects with repos
	_, _, err := execProjectAssign(dir1, AlwaysYes(), "Project A")
	require.NoError(t, err)
	_, _, err = execProjectAssign(dir2, AlwaysYes(), "Project B")
	require.NoError(t, err)

	stdout, err := execProjectList(home)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Project A")
	assert.Contains(t, stdout, "└── "+dir1)
	assert.Contains(t, stdout, "Project B")
	assert.Contains(t, stdout, "└── "+dir2)
}

func TestProjectListMultipleRepos(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	dir1 := t.TempDir()
	hooksDir1 := filepath.Join(dir1, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir1, "post-checkout"), []byte(hookContent), 0755))

	dir2 := t.TempDir()
	hooksDir2 := filepath.Join(dir2, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir2, "post-checkout"), []byte(hookContent), 0755))

	// Assign both repos to the same project
	_, _, err := execProjectAssign(dir1, AlwaysYes(), "Project A")
	require.NoError(t, err)
	_, _, err = execProjectAssign(dir2, AlwaysYes(), "Project A")
	require.NoError(t, err)

	stdout, err := execProjectList(home)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "├── "+dir1)
	assert.Contains(t, stdout, "└── "+dir2)
}

func TestProjectListNoRepos(t *testing.T) {
	home := t.TempDir()

	// Write a config with a project that has no repos
	cfg := &project.Config{
		Projects: []project.ProjectEntry{
			{ID: "abc1234", Name: "Empty Project", Slug: "empty-project", Repos: []string{}},
		},
	}
	require.NoError(t, project.WriteConfig(home, cfg))

	stdout, err := execProjectList(home)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Empty Project")
	assert.Contains(t, stdout, "└── (no repositories assigned)")
}

func TestProjectListRegisteredAsSubcommand(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "list")
}
