package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupCheckoutTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "Checkout Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, repoDir, proj
}

func execCheckout(homeDir, repoDir, projectFlag, prev, next string) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := checkoutCmd
	cmd.SetOut(stdout)

	err := runCheckout(cmd, homeDir, repoDir, projectFlag, prev, next, fixedNow)
	return stdout.String(), err
}

func TestCheckoutBasic(t *testing.T) {
	homeDir, repoDir, proj := setupCheckoutTest(t)

	stdout, err := execCheckout(homeDir, repoDir, "", "main", "feature-x")

	require.NoError(t, err)
	assert.Contains(t, stdout, "checkout")
	assert.Contains(t, stdout, "main")
	assert.Contains(t, stdout, "feature-x")
	assert.Contains(t, stdout, "Checkout Test")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "main", entries[0].Previous)
	assert.Equal(t, "feature-x", entries[0].Next)
}

func TestCheckoutByProjectFlag(t *testing.T) {
	homeDir, _, proj := setupCheckoutTest(t)

	stdout, err := execCheckout(homeDir, "", proj.Name, "main", "develop")

	require.NoError(t, err)
	assert.Contains(t, stdout, "Checkout Test")

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
}

func TestCheckoutMissingPrev(t *testing.T) {
	homeDir, repoDir, _ := setupCheckoutTest(t)

	_, err := execCheckout(homeDir, repoDir, "", "", "feature-x")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--prev is required")
}

func TestCheckoutMissingNext(t *testing.T) {
	homeDir, repoDir, _ := setupCheckoutTest(t)

	_, err := execCheckout(homeDir, repoDir, "", "main", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--next is required")
}

func TestCheckoutNoProject(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execCheckout(homeDir, "", "", "main", "feature-x")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no project found")
}

func TestCheckoutTimestamp(t *testing.T) {
	homeDir, repoDir, proj := setupCheckoutTest(t)

	_, err := execCheckout(homeDir, repoDir, "", "main", "feature-x")
	require.NoError(t, err)

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)

	expected := time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC)
	assert.Equal(t, expected, entries[0].Timestamp)
}

func TestCheckoutSameBranch(t *testing.T) {
	homeDir, repoDir, proj := setupCheckoutTest(t)

	stdout, err := execCheckout(homeDir, repoDir, "", "main", "main")

	require.NoError(t, err)
	assert.Empty(t, stdout) // silent no-op

	entries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	require.NoError(t, err)
	assert.Len(t, entries, 0) // no entry written
}

func TestCheckoutRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "checkout")
}
