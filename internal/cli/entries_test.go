package cli

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupEntriesTest(t *testing.T) (homeDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Entries Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, proj
}

func TestLoadProjectEntriesEmpty(t *testing.T) {
	homeDir, proj := setupEntriesTest(t)

	entries, err := LoadProjectEntries(homeDir, proj.Slug)

	require.NoError(t, err)
	assert.Empty(t, entries.Checkouts)
	assert.Empty(t, entries.Logs)
	assert.Empty(t, entries.Commits)
	assert.Empty(t, entries.ActivityStops)
	assert.Empty(t, entries.ActivityStarts)
}

func TestLoadProjectEntriesWithData(t *testing.T) {
	homeDir, proj := setupEntriesTest(t)

	now := time.Date(2025, 6, 11, 10, 0, 0, 0, time.UTC)

	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID: "aaa1111", Start: now, Minutes: 60, Message: "work",
	}))

	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, entry.CheckoutEntry{
		ID: "bbb2222", Timestamp: now, Previous: "main", Next: "feature",
	}))

	require.NoError(t, entry.WriteCommitEntry(homeDir, proj.Slug, entry.CommitEntry{
		ID: "ccc3333", Timestamp: now, Message: "commit", Branch: "feature",
	}))

	require.NoError(t, entry.WriteActivityStopEntry(homeDir, proj.Slug, entry.ActivityStopEntry{
		ID: "ddd4444", Timestamp: now,
	}))

	require.NoError(t, entry.WriteActivityStartEntry(homeDir, proj.Slug, entry.ActivityStartEntry{
		ID: "eee5555", Timestamp: now.Add(30 * time.Minute),
	}))

	entries, err := LoadProjectEntries(homeDir, proj.Slug)

	require.NoError(t, err)
	assert.Len(t, entries.Logs, 1)
	assert.Len(t, entries.Checkouts, 1)
	assert.Len(t, entries.Commits, 1)
	assert.Len(t, entries.ActivityStops, 1)
	assert.Len(t, entries.ActivityStarts, 1)
}

func TestLoadProjectEntriesInvalidSlug(t *testing.T) {
	homeDir := t.TempDir()

	// Non-existent project dir should return empty entries, not error
	entries, err := LoadProjectEntries(homeDir, "nonexistent")

	require.NoError(t, err)
	assert.Empty(t, entries.Logs)
	assert.Empty(t, entries.Checkouts)
	assert.Empty(t, entries.Commits)
	assert.Empty(t, entries.ActivityStops)
	assert.Empty(t, entries.ActivityStarts)
}
