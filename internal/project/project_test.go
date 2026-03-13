package project

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var hexPattern = regexp.MustCompile(`^[0-9a-f]{7}$`)

func TestHourgitDir(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit", HourgitDir("/home/user"))
}

func TestConfigPath(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit/config.json", ConfigPath("/home/user"))
}

func TestLogDir(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit/my-project", LogDir("/home/user", "my-project"))
}

func TestReadConfigMissing(t *testing.T) {
	home := t.TempDir()

	cfg, err := ReadConfig(home)

	require.NoError(t, err)
	assert.Empty(t, cfg.Projects)
	assert.NotEmpty(t, cfg.Defaults)
}

func TestConfigRoundTrip(t *testing.T) {
	home := t.TempDir()

	original := &Config{
		Defaults: schedule.DefaultSchedules(),
		Projects: []ProjectEntry{
			{ID: "abc1234", Name: "Test", Slug: "test", Repos: []string{"/repo1"}},
		},
	}

	require.NoError(t, WriteConfig(home, original))

	loaded, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, original.Projects, loaded.Projects)
	assert.NotEmpty(t, loaded.Defaults)
}

func TestFindProject(t *testing.T) {
	cfg := &Config{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	assert.NotNil(t, FindProject(cfg, "Alpha"))
	assert.Equal(t, "alpha", FindProject(cfg, "Alpha").Slug)
	assert.Nil(t, FindProject(cfg, "Gamma"))
}

func TestReadRepoConfigMissing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	cfg, err := ReadRepoConfig(dir)

	require.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestRepoConfigRoundTrip(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	original := &RepoConfig{Project: "My Project"}
	require.NoError(t, WriteRepoConfig(dir, original))

	loaded, err := ReadRepoConfig(dir)
	require.NoError(t, err)
	assert.Equal(t, "My Project", loaded.Project)
}

func TestRemoveRepoFromProject(t *testing.T) {
	entry := &ProjectEntry{
		ID:    "abc1234",
		Name:  "Test",
		Slug:  "test",
		Repos: []string{"/repo1", "/repo2", "/repo3"},
	}

	RemoveRepoFromProject(entry, "/repo2")

	assert.Equal(t, []string{"/repo1", "/repo3"}, entry.Repos)
}

func TestResolveProject(t *testing.T) {
	cfg := &Config{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	// Resolve by ID
	found := ResolveProject(cfg, "bbb2222")
	assert.NotNil(t, found)
	assert.Equal(t, "Beta", found.Name)

	// Resolve by name
	found = ResolveProject(cfg, "Alpha")
	assert.NotNil(t, found)
	assert.Equal(t, "aaa1111", found.ID)

	// Not found
	assert.Nil(t, ResolveProject(cfg, "nonexistent"))
}

func TestResolveOrCreateExisting(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	promptCalled := false
	result, err := ResolveOrCreate(home, "My Project", func(_ string) (bool, error) {
		promptCalled = true
		return false, nil
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.Created)
	assert.Equal(t, entry.ID, result.Entry.ID)
	assert.False(t, promptCalled)
}

func TestResolveOrCreateNew(t *testing.T) {
	home := t.TempDir()

	result, err := ResolveOrCreate(home, "New Project", func(name string) (bool, error) {
		assert.Equal(t, "New Project", name)
		return true, nil
	})

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.Created)
	assert.Equal(t, "New Project", result.Entry.Name)
	assert.Regexp(t, hexPattern, result.Entry.ID)

	// Verify project was actually persisted
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, cfg.Projects, 1)
}

func TestResolveOrCreateDeclined(t *testing.T) {
	home := t.TempDir()

	result, err := ResolveOrCreate(home, "New Project", func(_ string) (bool, error) {
		return false, nil
	})

	assert.NoError(t, err)
	assert.Nil(t, result)

	// Verify no project created
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Empty(t, cfg.Projects)
}

func TestCreateProjectNew(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")

	require.NoError(t, err)
	assert.Equal(t, "My Project", entry.Name)
	assert.Equal(t, "my-project", entry.Slug)
	assert.Regexp(t, hexPattern, entry.ID)
	assert.Empty(t, entry.Repos)

	// Verify config
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, cfg.Projects, 1)
	assert.Equal(t, entry.ID, cfg.Projects[0].ID)

	// Verify log dir
	_, err = os.Stat(LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestCreateProjectDuplicate(t *testing.T) {
	home := t.TempDir()

	_, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	_, err = CreateProject(home, "My Project")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestAssignProject(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	err = AssignProject(home, repo, entry)
	require.NoError(t, err)

	// Verify config updated
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Contains(t, cfg.Projects[0].Repos, repo)

	// Verify repo config
	repoCfg, err := ReadRepoConfig(repo)
	require.NoError(t, err)
	assert.Equal(t, "My Project", repoCfg.Project)
	assert.Equal(t, entry.ID, repoCfg.ProjectID)
}

func TestAssignProjectIdempotent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	require.NoError(t, AssignProject(home, repo, entry))
	require.NoError(t, AssignProject(home, repo, entry))

	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Len(t, cfg.Projects[0].Repos, 1)
}

func TestRemoveProject(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	removed, err := RemoveProject(home, entry.Name)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, removed.ID)

	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Empty(t, cfg.Projects)
}

func TestRemoveProjectByID(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	removed, err := RemoveProject(home, entry.ID)
	require.NoError(t, err)
	assert.Equal(t, "My Project", removed.Name)
}

func TestRemoveProjectNotFound(t *testing.T) {
	home := t.TempDir()

	_, err := RemoveProject(home, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveRepoConfig(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	require.NoError(t, WriteRepoConfig(dir, &RepoConfig{Project: "Test"}))

	err := RemoveRepoConfig(dir)
	assert.NoError(t, err)

	cfg, err := ReadRepoConfig(dir)
	assert.NoError(t, err)
	assert.Nil(t, cfg)
}

func TestRemoveRepoConfigMissing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, ".git"), 0755))

	err := RemoveRepoConfig(dir)
	assert.NoError(t, err)
}

func TestRemoveHookFromRepoOnlyHourgit(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))

	hookContent := "#!/bin/sh\n# Installed by hourgit\necho hourgit\n"
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte(hookContent), 0755))

	err := RemoveHookFromRepo(dir)
	assert.NoError(t, err)

	// Hook file should be deleted
	_, err = os.Stat(filepath.Join(hooksDir, "post-checkout"))
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveHookFromRepoMerged(t *testing.T) {
	dir := t.TempDir()
	hooksDir := filepath.Join(dir, ".git", "hooks")
	require.NoError(t, os.MkdirAll(hooksDir, 0755))

	hookContent := "#!/bin/sh\necho existing\n\n# Installed by hourgit\necho hourgit\n"
	require.NoError(t, os.WriteFile(filepath.Join(hooksDir, "post-checkout"), []byte(hookContent), 0755))

	err := RemoveHookFromRepo(dir)
	assert.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(hooksDir, "post-checkout"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "echo existing")
	assert.NotContains(t, string(data), "hourgit")
}

func TestRemoveHookFromRepoMissing(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, ".git", "hooks"), 0755))

	err := RemoveHookFromRepo(dir)
	assert.NoError(t, err)
}

func TestPreciseModeGetSet(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	cfg, err := ReadConfig(home)
	require.NoError(t, err)

	// Default is false
	assert.False(t, GetPreciseMode(cfg, entry.ID))

	// Set to true
	require.NoError(t, SetPreciseMode(home, entry.ID, true))

	cfg, err = ReadConfig(home)
	require.NoError(t, err)
	assert.True(t, GetPreciseMode(cfg, entry.ID))

	// Set back to false
	require.NoError(t, SetPreciseMode(home, entry.ID, false))

	cfg, err = ReadConfig(home)
	require.NoError(t, err)
	assert.False(t, GetPreciseMode(cfg, entry.ID))
}

func TestIdleThresholdGetSet(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	cfg, err := ReadConfig(home)
	require.NoError(t, err)

	// Default returns DefaultIdleThresholdMinutes
	assert.Equal(t, DefaultIdleThresholdMinutes, GetIdleThreshold(cfg, entry.ID))

	// Set custom value
	require.NoError(t, SetIdleThreshold(home, entry.ID, 15))

	cfg, err = ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, 15, GetIdleThreshold(cfg, entry.ID))
}

func TestPreciseModeSetNotFound(t *testing.T) {
	home := t.TempDir()

	err := SetPreciseMode(home, "nonexistent", true)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestIdleThresholdSetNotFound(t *testing.T) {
	home := t.TempDir()

	err := SetIdleThreshold(home, "nonexistent", 10)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestGetPreciseModeNotFound(t *testing.T) {
	cfg := &Config{}
	assert.False(t, GetPreciseMode(cfg, "nonexistent"))
}

func TestGetIdleThresholdNotFound(t *testing.T) {
	cfg := &Config{}
	assert.Equal(t, DefaultIdleThresholdMinutes, GetIdleThreshold(cfg, "nonexistent"))
}

func TestAnyPreciseProject(t *testing.T) {
	cfg := &Config{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Precise: false},
			{ID: "bbb2222", Name: "Beta", Precise: false},
		},
	}
	assert.False(t, AnyPreciseProject(cfg))

	cfg.Projects[1].Precise = true
	assert.True(t, AnyPreciseProject(cfg))
}

func TestPreciseModeJSONRoundTrip(t *testing.T) {
	home := t.TempDir()

	original := &Config{
		Defaults: schedule.DefaultSchedules(),
		Projects: []ProjectEntry{
			{ID: "abc1234", Name: "Test", Slug: "test", Repos: []string{}, Precise: true, IdleThresholdMinutes: 15},
		},
	}

	require.NoError(t, WriteConfig(home, original))

	loaded, err := ReadConfig(home)
	require.NoError(t, err)
	assert.True(t, loaded.Projects[0].Precise)
	assert.Equal(t, 15, loaded.Projects[0].IdleThresholdMinutes)
}

func TestPreciseModeBackwardCompat(t *testing.T) {
	home := t.TempDir()

	// Write config without precise fields (old format)
	original := &Config{
		Defaults: schedule.DefaultSchedules(),
		Projects: []ProjectEntry{
			{ID: "abc1234", Name: "Test", Slug: "test", Repos: []string{}},
		},
	}

	require.NoError(t, WriteConfig(home, original))

	loaded, err := ReadConfig(home)
	require.NoError(t, err)
	// Defaults should be fine
	assert.False(t, loaded.Projects[0].Precise)
	assert.Equal(t, 0, loaded.Projects[0].IdleThresholdMinutes)
}

func TestRenameProjectHappyPath(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "Old Name")
	require.NoError(t, err)

	// Assign a repo
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))
	require.NoError(t, AssignProject(home, repo, entry))

	renamed, err := RenameProject(home, entry.ID, "New Name")
	require.NoError(t, err)
	assert.Equal(t, "New Name", renamed.Name)
	assert.Equal(t, "new-name", renamed.Slug)

	// Verify config
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	assert.Equal(t, "New Name", cfg.Projects[0].Name)
	assert.Equal(t, "new-name", cfg.Projects[0].Slug)

	// Verify directory renamed
	_, err = os.Stat(LogDir(home, "new-name"))
	assert.NoError(t, err)
	_, err = os.Stat(LogDir(home, "old-name"))
	assert.True(t, os.IsNotExist(err))

	// Verify repo config updated
	rc, err := ReadRepoConfig(repo)
	require.NoError(t, err)
	assert.Equal(t, "New Name", rc.Project)
}

func TestRenameProjectSameSlug(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "my-project")
	require.NoError(t, err)

	renamed, err := RenameProject(home, entry.ID, "My Project")
	require.NoError(t, err)
	assert.Equal(t, "My Project", renamed.Name)
	assert.Equal(t, "my-project", renamed.Slug)

	// Directory should still exist (no rename attempted)
	_, err = os.Stat(LogDir(home, "my-project"))
	assert.NoError(t, err)
}

func TestRenameProjectConflict(t *testing.T) {
	home := t.TempDir()

	entryA, err := CreateProject(home, "Project A")
	require.NoError(t, err)
	_, err = CreateProject(home, "Project B")
	require.NoError(t, err)

	_, err = RenameProject(home, entryA.ID, "Project B")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestRenameProjectNotFound(t *testing.T) {
	home := t.TempDir()

	_, err := RenameProject(home, "nonexistent", "New Name")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRenameProjectMissingOldDir(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	// Remove the log dir to simulate a missing directory
	require.NoError(t, os.RemoveAll(LogDir(home, "my-project")))

	renamed, err := RenameProject(home, entry.ID, "New Name")
	require.NoError(t, err)
	assert.Equal(t, "New Name", renamed.Name)
	assert.Equal(t, "new-name", renamed.Slug)
}

func TestRenameProjectMissingRepoDir(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	// Manually add a non-existent repo
	cfg, err := ReadConfig(home)
	require.NoError(t, err)
	cfg.Projects[0].Repos = []string{"/nonexistent/repo"}
	require.NoError(t, WriteConfig(home, cfg))

	renamed, err := RenameProject(home, entry.ID, "New Name")
	require.NoError(t, err)
	assert.Equal(t, "New Name", renamed.Name)
}

func TestFindProjectByID(t *testing.T) {
	cfg := &Config{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	found := FindProjectByID(cfg, "bbb2222")
	assert.NotNil(t, found)
	assert.Equal(t, "Beta", found.Name)

	assert.Nil(t, FindProjectByID(cfg, "nonexistent"))
}

