package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
			{Name: "Test", Slug: "test", Repos: []string{"/repo1"}},
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
			{Name: "Alpha", Slug: "alpha"},
			{Name: "Beta", Slug: "beta"},
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
		Name:  "Test",
		Slug:  "test",
		Repos: []string{"/repo1", "/repo2", "/repo3"},
	}

	RemoveRepoFromProject(entry, "/repo2")

	assert.Equal(t, []string{"/repo1", "/repo3"}, entry.Repos)
}

func TestRegisterProjectNew(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))

	entry, created, err := RegisterProject(home, repo, "My Project")

	require.NoError(t, err)
	assert.True(t, created)
	assert.Equal(t, "My Project", entry.Name)
	assert.Equal(t, "my-project", entry.Slug)
	assert.Contains(t, entry.Repos, repo)

	// Verify registry file
	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)

	// Verify log dir
	_, err = os.Stat(LogDir(home, "my-project"))
	assert.NoError(t, err)

	// Verify repo config
	cfg, err := ReadRepoConfig(repo)
	require.NoError(t, err)
	assert.Equal(t, "My Project", cfg.Project)
}

func TestRegisterProjectExisting(t *testing.T) {
	home := t.TempDir()
	repo1 := t.TempDir()
	repo2 := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo1, ".git"), 0755))
	require.NoError(t, os.Mkdir(filepath.Join(repo2, ".git"), 0755))

	_, created1, err := RegisterProject(home, repo1, "My Project")
	require.NoError(t, err)
	assert.True(t, created1)

	_, created2, err := RegisterProject(home, repo2, "My Project")
	require.NoError(t, err)
	assert.False(t, created2)

	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects, 1)
	assert.Len(t, reg.Projects[0].Repos, 2)
}

func TestRegisterProjectIdempotentRepo(t *testing.T) {
	home := t.TempDir()
	repo := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(repo, ".git"), 0755))

	_, _, err := RegisterProject(home, repo, "My Project")
	require.NoError(t, err)

	_, _, err = RegisterProject(home, repo, "My Project")
	require.NoError(t, err)

	reg, err := ReadRegistry(home)
	require.NoError(t, err)
	assert.Len(t, reg.Projects[0].Repos, 1)
}
