package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupResolveTest(t *testing.T) (homeDir string, repoDir string, entry *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	// Create .git dir for repo config
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	// Create project and assign repo
	entry, err := project.CreateProject(homeDir, "Test Project")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, entry))

	// Re-read entry to get updated state
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	entry = project.FindProjectByID(cfg, entry.ID)

	return homeDir, repoDir, entry
}

func TestResolveProjectContextByFlag(t *testing.T) {
	homeDir, _, entry := setupResolveTest(t)

	got, err := ResolveProjectContext(homeDir, "", entry.Name)

	assert.NoError(t, err)
	assert.Equal(t, entry.ID, got.ID)
}

func TestResolveProjectContextByFlagID(t *testing.T) {
	homeDir, _, entry := setupResolveTest(t)

	got, err := ResolveProjectContext(homeDir, "", entry.ID)

	assert.NoError(t, err)
	assert.Equal(t, entry.Name, got.Name)
}

func TestResolveProjectContextByRepo(t *testing.T) {
	homeDir, repoDir, entry := setupResolveTest(t)

	got, err := ResolveProjectContext(homeDir, repoDir, "")

	assert.NoError(t, err)
	assert.Equal(t, entry.ID, got.ID)
}

func TestResolveProjectContextFlagNotFound(t *testing.T) {
	homeDir := t.TempDir()

	_, err := ResolveProjectContext(homeDir, "", "nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveProjectContextNoProjectNoRepo(t *testing.T) {
	homeDir := t.TempDir()

	_, err := ResolveProjectContext(homeDir, "", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestResolveProjectContextRepoNotAssigned(t *testing.T) {
	homeDir := t.TempDir()
	repoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	_, err := ResolveProjectContext(homeDir, repoDir, "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}
