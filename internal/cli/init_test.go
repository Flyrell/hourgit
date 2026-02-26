package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
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
	cmd := newRootCmd()
	cmd.SetOut(stdout)
	cmd.SetErr(stderr)
	cmd.SetArgs(append([]string{"init"}, args...))
	err := cmd.Execute()
	if err != nil {
		fmt.Fprint(stderr, "error: "+err.Error()+"\n")
	}
	return stdout.String(), stderr.String(), err
}

func execInitDirect(dir, homeDir, projectName string, force, merge bool, binPath string, confirm ConfirmFunc, selectFn SelectFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := initCmd
	cmd.SetOut(stdout)
	err := runInit(cmd, dir, homeDir, projectName, force, merge, binPath, confirm, selectFn)
	return stdout.String(), err
}

func TestInitInGitRepo(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()
	t.Setenv("SHELL", "")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	stdout, _, err := execInit()

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	hookPath := filepath.Join(dir, ".git", "hooks", "post-checkout")
	content, err := os.ReadFile(hookPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), project.HookMarker)
	assert.Contains(t, string(content), "#!/bin/sh")
	assert.Contains(t, string(content), "sync")
	assert.Contains(t, string(content), `[ "$3" = "0" ] && exit 0`)

	info, err := os.Stat(hookPath)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode().Perm())
}

func TestInitNotGitRepo(t *testing.T) {
	_, cleanup := setupInitTest(t)
	defer cleanup()

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "not a git repository")
}

func TestInitAlreadyInitialized(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\n"+project.HookMarker+"\n"), 0755))

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "hourgit is already initialized")
}

func TestInitHookExistsNoFlag(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	_, stderr, err := execInit()

	assert.Error(t, err)
	assert.Contains(t, stderr, "post-checkout hook already exists (use --force to overwrite or --merge to append)")
}

func TestInitHookExistsForce(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()
	t.Setenv("SHELL", "")

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	stdout, _, err := execInit("--force")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	content, err := os.ReadFile(filepath.Join(hooksDir, "post-checkout"))
	require.NoError(t, err)
	assert.Contains(t, string(content), project.HookMarker)
	assert.NotContains(t, string(content), "echo existing")
}

func TestInitHookExistsMerge(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()
	t.Setenv("SHELL", "")

	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte("#!/bin/sh\necho existing"), 0755))

	stdout, _, err := execInit("--merge")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")

	content, err := os.ReadFile(filepath.Join(hooksDir, "post-checkout"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "echo existing")
	assert.Contains(t, string(content), project.HookMarker)
}

func TestInitWithProjectFlag(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	stdout, _, err := execInit("--project", "My Project", "--yes")

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project 'My Project' created (")
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")
	assert.Contains(t, stdout, "hourgit initialized successfully")

	// Verify config.json created
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, appCfg.Projects, 1)
	assert.Equal(t, "My Project", appCfg.Projects[0].Name)
	assert.NotEmpty(t, appCfg.Projects[0].ID)

	// Verify .git/.hourgit written
	repoCfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "My Project", repoCfg.Project)
	assert.Equal(t, appCfg.Projects[0].ID, repoCfg.ProjectID)

	// Verify log dir created
	_, err = os.Stat(project.LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestInitWithProjectFlagByID(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	home := t.TempDir()
	t.Setenv("HOME", home)

	// Create a project in the registry first
	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	projectID := entry.ID

	// Init with the project ID
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))
	stdout, _, err := execInit("--project", projectID)

	assert.NoError(t, err)
	assert.NotContains(t, stdout, "created")
	assert.Contains(t, stdout, "repository assigned to project 'My Project'")
	assert.Contains(t, stdout, "hourgit initialized successfully")

	// Verify repo config uses name, not ID
	repoCfg, err := project.ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "My Project", repoCfg.Project)
	assert.Equal(t, projectID, repoCfg.ProjectID)
}

func TestInitWithProjectFlagConflict(t *testing.T) {
	dir, cleanup := setupInitTest(t)
	defer cleanup()

	home := t.TempDir()
	t.Setenv("HOME", home)

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	// Pre-assign to a different project
	require.NoError(t, project.WriteRepoConfig(dir, &project.RepoConfig{Project: "Old Project"}))

	_, stderr, err := execInit("--project", "New Project", "--yes")

	assert.Error(t, err)
	assert.Contains(t, stderr, "repository is already assigned to project 'Old Project'")
	assert.Contains(t, stderr, "use 'project assign --force' to reassign")
}

func TestInitWithProjectFlagDeclined(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	decline := func(_ string) (bool, error) { return false, nil }
	skipSelect := func(_ string, _ []string) (int, error) { return 1, nil }
	stdout, err := execInitDirect(dir, home, "My Project", false, false, "/usr/local/bin/hourgit", decline, skipSelect)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "project assignment skipped")
	assert.Contains(t, stdout, "hourgit initialized successfully")

	// Verify no project created
	appCfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Empty(t, appCfg.Projects)
}

func TestInitPromptsForCompletion(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/zsh")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	noConfirm := func(_ string) (bool, error) { return true, nil }
	selectCalls := 0
	installSelect := func(_ string, _ []string) (int, error) {
		selectCalls++
		return 0, nil
	}
	stdout, err := execInitDirect(dir, home, "", false, false, "/usr/local/bin/hourgit", noConfirm, installSelect)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")
	assert.Contains(t, stdout, "shell completions installed for")
	assert.Equal(t, 1, selectCalls)

	// Verify the eval line was appended
	assert.True(t, isCompletionInstalled("zsh", home))
}

func TestInitCompletionSkipped(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/zsh")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	noConfirm := func(_ string) (bool, error) { return true, nil }
	skipSelect := func(_ string, _ []string) (int, error) { return 1, nil }
	stdout, err := execInitDirect(dir, home, "", false, false, "/usr/local/bin/hourgit", noConfirm, skipSelect)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")
	assert.NotContains(t, stdout, "shell completions installed")
	assert.False(t, isCompletionInstalled("zsh", home))
}

func TestInitCompletionAlreadyInstalled(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/zsh")

	// Pre-install completion
	require.NoError(t, os.WriteFile(filepath.Join(home, ".zshrc"), []byte(`eval "$(hourgit completion generate zsh)"`), 0644))
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	noConfirm := func(_ string) (bool, error) { return true, nil }
	selectCalls := 0
	trackSelect := func(_ string, _ []string) (int, error) {
		selectCalls++
		return 0, nil
	}
	stdout, err := execInitDirect(dir, home, "", false, false, "/usr/local/bin/hourgit", noConfirm, trackSelect)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")
	assert.NotContains(t, stdout, "shell completions installed")
	// Should not have prompted since completion is already installed
	assert.Equal(t, 0, selectCalls)
}

func TestInitCompletionUnknownShell(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/csh")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	noConfirm := func(_ string) (bool, error) { return true, nil }
	selectCalls := 0
	trackSelect := func(_ string, _ []string) (int, error) {
		selectCalls++
		return 0, nil
	}
	stdout, err := execInitDirect(dir, home, "", false, false, "/usr/local/bin/hourgit", noConfirm, trackSelect)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")
	assert.NotContains(t, stdout, "shell completions installed")
	// Should not have prompted since shell is unknown
	assert.Equal(t, 0, selectCalls)
}

func TestInitYesAutoInstallsCompletion(t *testing.T) {
	dir := t.TempDir()
	home := t.TempDir()
	t.Setenv("SHELL", "/bin/zsh")

	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	noConfirm := func(_ string) (bool, error) { return true, nil }
	autoInstall := func(_ string, _ []string) (int, error) { return 0, nil }
	stdout, err := execInitDirect(dir, home, "", false, false, "/usr/local/bin/hourgit", noConfirm, autoInstall)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "hourgit initialized successfully")
	assert.Contains(t, stdout, "shell completions installed for")
	assert.True(t, isCompletionInstalled("zsh", home))
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

func TestHookScript(t *testing.T) {
	script := hookScript("/usr/local/bin/hourgit", "1.2.3")

	assert.Contains(t, script, "#!/bin/sh")
	assert.Contains(t, script, project.HookMarker)
	assert.Contains(t, script, "(version: 1.2.3)")
	assert.Contains(t, script, `/usr/local/bin/hourgit sync`)
	assert.Contains(t, script, `[ "$3" = "0" ] && exit 0`)
	assert.Contains(t, script, `[ "$1" = "$2" ] && exit 0`)
	assert.NotContains(t, script, `checkout --prev`)
	assert.NotContains(t, script, `git name-rev`)
	assert.NotContains(t, script, `git symbolic-ref`)
	assert.NotContains(t, script, `git rev-parse --git-dir`)
	assert.NotContains(t, script, `rebase-merge`)
}

func TestInitRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "init")
}
