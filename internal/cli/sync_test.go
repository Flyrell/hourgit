package cli

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupSyncTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Sync Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, repoDir, proj
}

func fakeReflog(output string) GitReflogFunc {
	return func(repoDir string, since *time.Time) (string, error) {
		return output, nil
	}
}

func fakeReflogWithSinceCheck(output string, sinceCalled *bool) GitReflogFunc {
	return func(repoDir string, since *time.Time) (string, error) {
		if since != nil {
			*sinceCalled = true
		}
		return output, nil
	}
}

func execSync(homeDir, repoDir, projectFlag string, gitReflog GitReflogFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := syncCmd
	cmd.SetOut(stdout)

	err := runSync(cmd, homeDir, repoDir, projectFlag, gitReflog)
	return stdout.String(), err
}

func TestSyncBasic(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "synced 1 checkout(s)")
	assert.Contains(t, stdout, "Sync Test")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "main", entries[0].Previous)
	assert.Equal(t, "feature-x", entries[0].Next)
	assert.Equal(t, "abc1234", entries[0].CommitRef)
}

func TestSyncDeduplication(t *testing.T) {
	homeDir, repoDir, _ := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	// First sync
	_, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)

	// Second sync with same data
	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)
	assert.Contains(t, stdout, "already up to date")
}

func TestSyncIdempotentIDs(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	_, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)

	entries1, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	require.Len(t, entries1, 1)
	id1 := entries1[0].ID

	// Delete and re-sync — should produce the same ID
	require.NoError(t, entry.DeleteEntry(homeDir, proj.Slug, id1))

	_, err = execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)

	entries2, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	require.Len(t, entries2, 1)
	assert.Equal(t, id1, entries2[0].ID)
}

func TestSyncSkipsDetachedHead(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from a1b2c3d to feature-x
def5678 HEAD@{2025-06-15 14:00:00 +0000}: checkout: moving from main to a1b2c3d`

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "already up to date")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestSyncSkipsRemoteRefs(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from remotes/origin/main to feature-x`

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "already up to date")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestSyncSkipsSameBranch(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to main`

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "already up to date")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 0)
}

func TestSyncEmptyReflog(t *testing.T) {
	homeDir, repoDir, _ := setupSyncTest(t)

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(""))

	require.NoError(t, err)
	assert.Contains(t, stdout, "already up to date")
}

func TestSyncBranchNamesWithSlashes(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from feature/ENG-641/item to release/v2.0`

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "synced 1 checkout(s)")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "feature/ENG-641/item", entries[0].Previous)
	assert.Equal(t, "release/v2.0", entries[0].Next)
}

func TestSyncTimestampFromReflog(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	_, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	expected := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, entries[0].Timestamp)
}

func TestSyncSinceOptimization(t *testing.T) {
	homeDir, repoDir, _ := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	// First sync — no --since
	sinceCalled := false
	_, err := execSync(homeDir, repoDir, "", fakeReflogWithSinceCheck(reflogOutput, &sinceCalled))
	require.NoError(t, err)
	assert.False(t, sinceCalled, "first sync should not pass --since")

	// Second sync — should pass --since
	newReflog := `def5678 HEAD@{2025-06-15 16:00:00 +0000}: checkout: moving from feature-x to develop`
	sinceCalled = false
	_, err = execSync(homeDir, repoDir, "", fakeReflogWithSinceCheck(newReflog, &sinceCalled))
	require.NoError(t, err)
	assert.True(t, sinceCalled, "subsequent sync should pass --since")
}

func TestSyncLastSyncUpdated(t *testing.T) {
	homeDir, repoDir, _ := setupSyncTest(t)

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	_, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))
	require.NoError(t, err)

	repoCfg, err := project.ReadRepoConfig(repoDir)
	require.NoError(t, err)
	require.NotNil(t, repoCfg.LastSync)

	expected := time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC)
	assert.Equal(t, expected, *repoCfg.LastSync)
}

func TestSyncByProjectFlag(t *testing.T) {
	homeDir, _, proj := setupSyncTest(t)

	// Use a different repoDir without .git/.hourgit — rely on --project flag
	repoDir2 := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(repoDir2, ".git"), 0755))
	// Write repo config manually so ReadRepoConfig succeeds
	require.NoError(t, project.WriteRepoConfig(repoDir2, &project.RepoConfig{Project: proj.Name, ProjectID: proj.ID}))

	reflogOutput := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	stdout, err := execSync(homeDir, repoDir2, proj.Name, fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "Sync Test")
}

func TestSyncNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execSync(homeDir, "", "", fakeReflog(""))

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestSyncMultipleCheckouts(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	reflogOutput := fmt.Sprintf(
		"%s\n%s\n%s",
		`abc1234 HEAD@{2025-06-15 16:00:00 +0000}: checkout: moving from develop to main`,
		`def5678 HEAD@{2025-06-15 15:00:00 +0000}: checkout: moving from feature-x to develop`,
		`aaa9012 HEAD@{2025-06-15 14:00:00 +0000}: checkout: moving from main to feature-x`,
	)

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "synced 3 checkout(s)")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 3)
}

func TestSyncSameCommitRefDifferentDirections(t *testing.T) {
	homeDir, repoDir, proj := setupSyncTest(t)

	// Both checkouts share the same commit ref (HEAD didn't move),
	// which happens when switching A→B and back B→A without new commits.
	reflogOutput := fmt.Sprintf(
		"%s\n%s",
		`abc1234 HEAD@{2025-06-15 15:00:00 +0000}: checkout: moving from feature-x to main`,
		`abc1234 HEAD@{2025-06-15 14:00:00 +0000}: checkout: moving from main to feature-x`,
	)

	stdout, err := execSync(homeDir, repoDir, "", fakeReflog(reflogOutput))

	require.NoError(t, err)
	assert.Contains(t, stdout, "synced 2 checkout(s)")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)

	// Verify both directions were recorded
	nexts := map[string]bool{}
	for _, e := range entries {
		nexts[e.Next] = true
	}
	assert.True(t, nexts["main"], "checkout to main should be recorded")
	assert.True(t, nexts["feature-x"], "checkout to feature-x should be recorded")
}

func TestSyncRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "sync")
}
