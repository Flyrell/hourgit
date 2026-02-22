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

func setupProjectTest(t *testing.T) string {
	t.Helper()

	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create .git dir with hourgit hook
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(hooksDir, "post-checkout"),
		[]byte(hookContent), 0755,
	))

	return dir
}

func execProjectSet(dir string, args ...string) (string, string, error) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	home := os.Getenv("HOME")

	force := false
	var projectName string
	for _, a := range args {
		if a == "--force" {
			force = true
		} else {
			projectName = a
		}
	}

	cmd := projectSetCmd
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := runProjectSet(cmd, dir, home, projectName, force)
	return stdout.String(), stderr.String(), err
}

func TestProjectSetHappyPath(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	stdout, _, err := execProjectSet(dir, "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' created")
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")

	// Verify registry
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)
	assert.Equal(t, "My Project", reg.Projects[0].Name)
	assert.Contains(t, reg.Projects[0].Repos, dir)

	// Verify repo config
	cfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "My Project", cfg.Project)

	// Verify log dir
	_, err = os.Stat(project.LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestProjectSetNotInitialized(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create .git but no hook
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	_, _, err := execProjectSet(dir, "My Project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hourgit is not initialized")
}

func TestProjectSetSameProjectNoop(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	// First assignment
	_, _, err := execProjectSet(dir, "My Project")
	require.NoError(t, err)

	// Same project again
	stdout, _, err := execProjectSet(dir, "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "repository is already assigned to project 'My Project'")

	// Verify still one repo in registry
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects[0].Repos, 1)
}

func TestProjectSetDifferentProjectNoForce(t *testing.T) {
	dir := setupProjectTest(t)

	_, _, err := execProjectSet(dir, "Project A")
	require.NoError(t, err)

	_, _, err = execProjectSet(dir, "Project B")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository is already assigned to project 'Project A'")
	assert.Contains(t, err.Error(), "use --force to reassign")
}

func TestProjectSetDifferentProjectWithForce(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	_, _, err := execProjectSet(dir, "Project A")
	require.NoError(t, err)

	stdout, _, err := execProjectSet(dir, "--force", "Project B")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'Project B' created")
	assert.Contains(t, stdout, "repository assigned to project 'Project B'")

	// Verify repo removed from old project
	reg, err := project.ReadRegistry(home)
	require.NoError(t, err)

	oldEntry := project.FindProject(reg, "Project A")
	require.NotNil(t, oldEntry)
	assert.Empty(t, oldEntry.Repos)

	newEntry := project.FindProject(reg, "Project B")
	require.NotNil(t, newEntry)
	assert.Contains(t, newEntry.Repos, dir)

	// Verify repo config updated
	cfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "Project B", cfg.Project)
}

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

	// Create two repo dirs with hourgit hooks (inline setup to share HOME)
	dir1 := t.TempDir()
	hooksDir1 := filepath.Join(dir1, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir1, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir1, "post-checkout"), []byte(hookContent), 0755))

	dir2 := t.TempDir()
	hooksDir2 := filepath.Join(dir2, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir2, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir2, "post-checkout"), []byte(hookContent), 0755))

	// Create two projects with repos
	_, _, err := execProjectSet(dir1, "Project A")
	require.NoError(t, err)
	_, _, err = execProjectSet(dir2, "Project B")
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
	_, _, err := execProjectSet(dir1, "Project A")
	require.NoError(t, err)
	_, _, err = execProjectSet(dir2, "Project A")
	require.NoError(t, err)

	stdout, err := execProjectList(home)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "├── "+dir1)
	assert.Contains(t, stdout, "└── "+dir2)
}

func TestProjectListNoRepos(t *testing.T) {
	home := t.TempDir()

	// Write a registry with a project that has no repos
	reg := &project.ProjectRegistry{
		Projects: []project.ProjectEntry{
			{Name: "Empty Project", Slug: "empty-project", Repos: []string{}},
		},
	}
	require.NoError(t, project.WriteRegistry(home, reg))

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

func TestProjectRegisteredAsSubcommand(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "project")
}
