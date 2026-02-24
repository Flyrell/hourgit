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

func execProjectAssign(dir string, confirm ConfirmFunc, args ...string) (string, string, error) {
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

	cmd := projectAssignCmd
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)

	err := runProjectAssign(cmd, dir, home, projectName, force, confirm)
	return stdout.String(), stderr.String(), err
}

func TestProjectAssignHappyPathNewProject(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	stdout, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' created (")
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")

	// Verify config
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, appCfg.Projects, 1)
	assert.Equal(t, "My Project", appCfg.Projects[0].Name)
	assert.NotEmpty(t, appCfg.Projects[0].ID)
	assert.Contains(t, appCfg.Projects[0].Repos, dir)

	// Verify repo config
	repoCfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "My Project", repoCfg.Project)
	assert.Equal(t, appCfg.Projects[0].ID, repoCfg.ProjectID)

	// Verify log dir
	_, err = os.Stat(project.LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestProjectAssignExistingProject(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	// Pre-create the project
	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")

	assert.NoError(t, err)
	assert.NotContains(t, stdout, "created")
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")
}

func TestProjectAssignNewProjectDeclined(t *testing.T) {
	dir := setupProjectTest(t)

	decline := func(_ string) (bool, error) { return false, nil }
	_, _, err := execProjectAssign(dir, decline, "My Project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "aborted")
}

func TestProjectAssignNotInitialized(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create .git but no hook
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	_, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "hourgit is not initialized")
}

func TestProjectAssignSameProjectNoop(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	// First assignment
	_, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")
	require.NoError(t, err)

	// Same project again
	stdout, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "repository is already assigned to project 'My Project'")

	// Verify still one repo in config
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, appCfg.Projects[0].Repos, 1)
}

func TestProjectAssignByID(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	// Create a project first
	_, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")
	require.NoError(t, err)

	// Get the project ID
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	projectID := appCfg.Projects[0].ID

	// Set up a second repo
	dir2 := t.TempDir()
	hooksDir := filepath.Join(dir2, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte(hookContent), 0755))

	// Assign by ID
	stdout, _, err := execProjectAssign(dir2, AlwaysYes(), projectID)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")
	assert.NotContains(t, stdout, "created")

	// Verify repo config uses name, not ID
	repoCfg, err := project.ReadRepoConfig(dir2)
	require.NoError(t, err)
	assert.Equal(t, "My Project", repoCfg.Project)
	assert.Equal(t, projectID, repoCfg.ProjectID)
}

func TestProjectAssignSameProjectByID(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	// Create a project
	_, _, err := execProjectAssign(dir, AlwaysYes(), "My Project")
	require.NoError(t, err)

	// Get the project ID
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	projectID := appCfg.Projects[0].ID

	// Try to assign same project by ID — should be a noop
	stdout, _, err := execProjectAssign(dir, AlwaysYes(), projectID)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "repository is already assigned to project 'My Project'")
}

func TestProjectAssignDifferentProjectNoForce(t *testing.T) {
	dir := setupProjectTest(t)

	_, _, err := execProjectAssign(dir, AlwaysYes(), "Project A")
	require.NoError(t, err)

	_, _, err = execProjectAssign(dir, AlwaysYes(), "Project B")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "repository is already assigned to project 'Project A'")
	assert.Contains(t, err.Error(), "use --force to reassign")
}

func TestProjectAssignDifferentProjectWithForce(t *testing.T) {
	dir := setupProjectTest(t)
	home := os.Getenv("HOME")

	_, _, err := execProjectAssign(dir, AlwaysYes(), "Project A")
	require.NoError(t, err)

	stdout, _, err := execProjectAssign(dir, AlwaysYes(), "--force", "Project B")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'Project B' created (")
	assert.Contains(t, stdout, "repository assigned to project 'Project B'")

	// Verify repo removed from old project
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)

	oldEntry := project.FindProject(appCfg, "Project A")
	require.NotNil(t, oldEntry)
	assert.Empty(t, oldEntry.Repos)

	newEntry := project.FindProject(appCfg, "Project B")
	require.NotNil(t, newEntry)
	assert.Contains(t, newEntry.Repos, dir)

	// Verify repo config updated
	repoCfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "Project B", repoCfg.Project)
}

func TestProjectAssignRegisteredAsSubcommand(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "assign")
}
