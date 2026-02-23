package project

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var hexPattern = regexp.MustCompile(`^[0-9a-f]{7}$`)

func TestHourgitDir(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit", HourgitDir("/home/user"))
}

func TestRegistryPath(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit/projects.json", RegistryPath("/home/user"))
}

func TestLogDir(t *testing.T) {
	assert.Equal(t, "/home/user/.hourgit/my-project", LogDir("/home/user", "my-project"))
}

func TestReadRegistryMissing(t *testing.T) {
	home := t.TempDir()

	reg, err := ReadRegistry(home)

	require.NoError(t, err)
	assert.Empty(t, reg.Projects)
}

func TestRegistryRoundTrip(t *testing.T) {
	home := t.TempDir()

	original := &ProjectRegistry{
		Projects: []ProjectEntry{
			{ID: "abc1234", Name: "Test", Slug: "test", Repos: []string{"/repo1"}},
		},
	}

	require.NoError(t, WriteRegistry(home, original))

	loaded, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Equal(t, original.Projects, loaded.Projects)
}

func TestFindProject(t *testing.T) {
	reg := &ProjectRegistry{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	assert.NotNil(t, FindProject(reg, "Alpha"))
	assert.Equal(t, "alpha", FindProject(reg, "Alpha").Slug)
	assert.Nil(t, FindProject(reg, "Gamma"))
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
	reg := &ProjectRegistry{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	// Resolve by ID
	found := ResolveProject(reg, "bbb2222")
	assert.NotNil(t, found)
	assert.Equal(t, "Beta", found.Name)

	// Resolve by name
	found = ResolveProject(reg, "Alpha")
	assert.NotNil(t, found)
	assert.Equal(t, "aaa1111", found.ID)

	// Not found
	assert.Nil(t, ResolveProject(reg, "nonexistent"))
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
	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)
}

func TestResolveOrCreateDeclined(t *testing.T) {
	home := t.TempDir()

	result, err := ResolveOrCreate(home, "New Project", func(_ string) (bool, error) {
		return false, nil
	})

	assert.NoError(t, err)
	assert.Nil(t, result)

	// Verify no project created
	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Empty(t, reg.Projects)
}

func TestCreateProjectNew(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")

	require.NoError(t, err)
	assert.Equal(t, "My Project", entry.Name)
	assert.Equal(t, "my-project", entry.Slug)
	assert.Regexp(t, hexPattern, entry.ID)
	assert.Empty(t, entry.Repos)

	// Verify registry
	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)
	assert.Equal(t, entry.ID, reg.Projects[0].ID)

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

	// Verify registry updated
	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Contains(t, reg.Projects[0].Repos, repo)

	// Verify repo config
	cfg, err := ReadRepoConfig(repo)
	require.NoError(t, err)
	assert.Equal(t, "My Project", cfg.Project)
	assert.Equal(t, entry.ID, cfg.ProjectID)
}

func TestAssignProjectIdempotent(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	require.NoError(t, AssignProject(home, repo, entry))
	require.NoError(t, AssignProject(home, repo, entry))

	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects[0].Repos, 1)
}

func TestRemoveProject(t *testing.T) {
	home := t.TempDir()

	entry, err := CreateProject(home, "My Project")
	require.NoError(t, err)

	removed, err := RemoveProject(home, entry.Name)
	require.NoError(t, err)
	assert.Equal(t, entry.ID, removed.ID)

	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Empty(t, reg.Projects)
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

func TestFindProjectByID(t *testing.T) {
	reg := &ProjectRegistry{
		Projects: []ProjectEntry{
			{ID: "aaa1111", Name: "Alpha", Slug: "alpha"},
			{ID: "bbb2222", Name: "Beta", Slug: "beta"},
		},
	}

	found := FindProjectByID(reg, "bbb2222")
	assert.NotNil(t, found)
	assert.Equal(t, "Beta", found.Name)

	assert.Nil(t, FindProjectByID(reg, "nonexistent"))
}
