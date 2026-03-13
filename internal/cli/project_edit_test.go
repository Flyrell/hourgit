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

func execProjectEdit(homeDir, repoDir, identifier, nameFlag, modeFlag string, idleThreshold int) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := projectEditCmd
	cmd.SetOut(stdout)

	pk := PromptKit{
		Confirm: AlwaysYes(),
	}

	err := runProjectEdit(cmd, homeDir, repoDir, identifier, nameFlag, modeFlag, idleThreshold, "/usr/local/bin/hourgit", pk)
	return stdout.String(), err
}

func TestProjectEditRenameHappyPath(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "Old Name")
	require.NoError(t, err)

	// Assign a repo so we can verify repo config update
	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0755))
	require.NoError(t, project.AssignProject(home, repo, entry))

	stdout, err := execProjectEdit(home, "", "Old Name", "New Name", "", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Old Name")
	assert.Contains(t, stdout, "New Name")

	// Verify config updated
	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	require.Len(t, cfg.Projects, 1)
	assert.Equal(t, "New Name", cfg.Projects[0].Name)
	assert.Equal(t, "new-name", cfg.Projects[0].Slug)

	// Verify data directory renamed
	_, err = os.Stat(project.LogDir(home, "new-name"))
	assert.NoError(t, err)
	_, err = os.Stat(project.LogDir(home, "old-name"))
	assert.True(t, os.IsNotExist(err))

	// Verify repo config updated
	rc, err := project.ReadRepoConfig(repo)
	require.NoError(t, err)
	assert.Equal(t, "New Name", rc.Project)
}

func TestProjectEditRenameConflict(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "Project A")
	require.NoError(t, err)
	_, err = project.CreateProject(home, "Project B")
	require.NoError(t, err)

	_, err = execProjectEdit(home, "", "Project A", "Project B", "", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestProjectEditRenameSameSlug(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "my-project")
	require.NoError(t, err)

	// "My Project" slugifies to the same "my-project" — no dir rename needed
	stdout, err := execProjectEdit(home, "", "my-project", "My Project", "", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "My Project")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "My Project", cfg.Projects[0].Name)
	assert.Equal(t, "my-project", cfg.Projects[0].Slug)
}

func TestProjectEditRenameMissingRepoDir(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "Old Name")
	require.NoError(t, err)

	// Add a non-existent repo path
	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	cfg.Projects[0].Repos = []string{"/nonexistent/repo"}
	require.NoError(t, project.WriteConfig(home, cfg))
	_ = entry

	stdout, err := execProjectEdit(home, "", "Old Name", "New Name", "", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "New Name")
}

func TestProjectEditModeStandardToPrecise(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout, err := execProjectEdit(home, "", "My Project", "", "precise", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "standard")
	assert.Contains(t, stdout, "precise")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.True(t, cfg.Projects[0].Precise)
	assert.Equal(t, project.DefaultIdleThresholdMinutes, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectEditModePreciseToStandard(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))

	stdout, err := execProjectEdit(home, "", "My Project", "", "standard", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "precise")
	assert.Contains(t, stdout, "standard")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.False(t, cfg.Projects[0].Precise)
}

func TestProjectEditNameAndMode(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "Old Name")
	require.NoError(t, err)

	stdout, err := execProjectEdit(home, "", "Old Name", "New Name", "precise", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "New Name")
	assert.Contains(t, stdout, "precise")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "New Name", cfg.Projects[0].Name)
	assert.True(t, cfg.Projects[0].Precise)
}

func TestProjectEditNoChanges(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	// Same name, same mode (standard is default)
	stdout, err := execProjectEdit(home, "", "My Project", "My Project", "standard", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "no changes")
}

func TestProjectEditNotFound(t *testing.T) {
	home := t.TempDir()

	_, err := execProjectEdit(home, "", "nonexistent", "New Name", "", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestProjectEditInvalidMode(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	_, err = execProjectEdit(home, "", "My Project", "", "foobar", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --mode value")
}

func TestProjectEditNoProjectSpecified(t *testing.T) {
	home := t.TempDir()

	_, err := execProjectEdit(home, "", "", "New Name", "", 0)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project specified")
}

func TestProjectEditResolvesFromRepoConfig(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	repo := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repo, ".git"), 0755))
	require.NoError(t, project.AssignProject(home, repo, entry))

	// No identifier — should resolve from repo config
	stdout, err := execProjectEdit(home, repo, "", "Renamed", "", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Renamed")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", cfg.Projects[0].Name)
}

func TestProjectEditByID(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout, err := execProjectEdit(home, "", entry.ID, "Renamed", "", 0)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "Renamed")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "Renamed", cfg.Projects[0].Name)
}

func TestProjectEditInteractiveMode(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	stdout := new(bytes.Buffer)
	cmd := projectEditCmd
	cmd.SetOut(stdout)

	promptCalls := 0
	pk := PromptKit{
		Confirm: AlwaysYes(),
		PromptWithDefault: func(prompt, defaultValue string) (string, error) {
			promptCalls++
			if promptCalls == 1 {
				assert.Equal(t, "My Project", defaultValue)
				return "New Name", nil
			}
			// Idle threshold prompt — accept default
			return defaultValue, nil
		},
		Select: func(title string, options []string) (int, error) {
			// First option is "standard" (current mode), pick "precise" (index 1)
			return 1, nil
		},
	}

	err = runProjectEdit(cmd, home, "", "My Project", "", "", 0, "/usr/local/bin/hourgit", pk)

	assert.NoError(t, err)
	assert.Equal(t, 2, promptCalls, "should prompt for name and idle threshold")
	assert.Contains(t, stdout.String(), "New Name")
	assert.Contains(t, stdout.String(), "precise")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "New Name", cfg.Projects[0].Name)
	assert.True(t, cfg.Projects[0].Precise)
}

func TestProjectEditIdleThresholdHappyPath(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))
	require.NoError(t, project.SetIdleThreshold(home, entry.ID, project.DefaultIdleThresholdMinutes))

	stdout, err := execProjectEdit(home, "", "My Project", "", "", 15)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "idle threshold")
	assert.Contains(t, stdout, "10m")
	assert.Contains(t, stdout, "15m")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, 15, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectEditIdleThresholdOnStandardProject(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	_, err = execProjectEdit(home, "", "My Project", "", "", 15)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only valid for precise mode")
}

func TestProjectEditIdleThresholdNoChange(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))
	require.NoError(t, project.SetIdleThreshold(home, entry.ID, 10))

	stdout, err := execProjectEdit(home, "", "My Project", "", "", 10)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "no changes")
}

func TestProjectEditIdleThresholdWithModeChange(t *testing.T) {
	home := t.TempDir()

	_, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)

	// Switch to precise and set idle threshold in one command
	stdout, err := execProjectEdit(home, "", "My Project", "", "precise", 20)

	assert.NoError(t, err)
	assert.Contains(t, stdout, "precise")
	assert.Contains(t, stdout, "idle threshold")
	assert.Contains(t, stdout, "10m")
	assert.Contains(t, stdout, "20m")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.True(t, cfg.Projects[0].Precise)
	assert.Equal(t, 20, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectEditIdleThresholdWithModeChangeToStandard(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))

	// Switching to standard while setting idle threshold should error
	_, err = execProjectEdit(home, "", "My Project", "", "standard", 15)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only valid for precise mode")
}

func TestProjectEditIdleThresholdInvalidInteractive(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))
	require.NoError(t, project.SetIdleThreshold(home, entry.ID, 10))

	stdout := new(bytes.Buffer)
	cmd := projectEditCmd
	cmd.SetOut(stdout)

	promptCalls := 0
	pk := PromptKit{
		Confirm: AlwaysYes(),
		PromptWithDefault: func(prompt, defaultValue string) (string, error) {
			promptCalls++
			if promptCalls == 1 {
				return defaultValue, nil
			}
			return "abc", nil // invalid threshold
		},
		Select: func(title string, options []string) (int, error) {
			return 0, nil
		},
	}

	err = runProjectEdit(cmd, home, "", "My Project", "", "", 0, "/usr/local/bin/hourgit", pk)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid idle threshold")
}

func TestProjectEditInteractivePreciseIdleThreshold(t *testing.T) {
	home := t.TempDir()

	entry, err := project.CreateProject(home, "My Project")
	require.NoError(t, err)
	require.NoError(t, project.SetPreciseMode(home, entry.ID, true))
	require.NoError(t, project.SetIdleThreshold(home, entry.ID, 10))

	stdout := new(bytes.Buffer)
	cmd := projectEditCmd
	cmd.SetOut(stdout)

	promptCalls := 0
	pk := PromptKit{
		Confirm: AlwaysYes(),
		PromptWithDefault: func(prompt, defaultValue string) (string, error) {
			promptCalls++
			if promptCalls == 1 {
				return defaultValue, nil // keep name
			}
			// Idle threshold prompt — change to 20
			assert.Equal(t, "10", defaultValue)
			return "20", nil
		},
		Select: func(title string, options []string) (int, error) {
			// "precise" is first (current mode), keep it
			return 0, nil
		},
	}

	err = runProjectEdit(cmd, home, "", "My Project", "", "", 0, "/usr/local/bin/hourgit", pk)

	assert.NoError(t, err)
	assert.Equal(t, 2, promptCalls, "should prompt for name and idle threshold")
	assert.Contains(t, stdout.String(), "idle threshold")
	assert.Contains(t, stdout.String(), "10m")
	assert.Contains(t, stdout.String(), "20m")

	cfg, err := project.ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, 20, cfg.Projects[0].IdleThresholdMinutes)
}

func TestProjectEditRegisteredAsSubcommand(t *testing.T) {
	commands := projectCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "edit")
}
