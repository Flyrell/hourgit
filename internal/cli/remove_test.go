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

func setupRemoveTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry, logEntry entry.Entry, checkoutEntry entry.CheckoutEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir = t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	p, err := project.CreateProject(homeDir, "Remove Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, p))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	le := entry.Entry{
		ID:        "rmlog12",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   120,
		Message:   "test work",
		Task:      "coding",
		CreatedAt: time.Date(2025, 6, 16, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, le))

	ce := entry.CheckoutEntry{
		ID:        "rmchk12",
		Type:      "checkout",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, p.Slug, ce))

	return homeDir, repoDir, p, le, ce
}

func execRemove(homeDir, repoDir, projectFlag, hash string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := removeCmd
	cmd.SetOut(stdout)

	err := runRemove(cmd, homeDir, repoDir, projectFlag, hash, confirm)
	return stdout.String(), err
}

func TestRemoveLogEntry(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupRemoveTest(t)

	stdout, err := execRemove(homeDir, repoDir, "", "rmlog12", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "log")
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, proj.Slug, "rmlog12")
	assert.Error(t, err)
}

func TestRemoveCheckoutEntry(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupRemoveTest(t)

	stdout, err := execRemove(homeDir, repoDir, "", "rmchk12", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "checkout")
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadCheckoutEntry(homeDir, proj.Slug, "rmchk12")
	assert.Error(t, err)
}

func TestRemoveNotFound(t *testing.T) {
	homeDir, repoDir, _, _, _ := setupRemoveTest(t)

	_, err := execRemove(homeDir, repoDir, "", "nonexist", AlwaysYes())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestRemoveConfirmDeclined(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupRemoveTest(t)

	confirm := func(_ string) (bool, error) {
		return false, nil
	}

	stdout, err := execRemove(homeDir, repoDir, "", "rmlog12", confirm)

	require.NoError(t, err)
	assert.Contains(t, stdout, "cancelled")
	assert.NotContains(t, stdout, "removed entry")

	// Entry should still exist
	_, err = entry.ReadEntry(homeDir, proj.Slug, "rmlog12")
	assert.NoError(t, err)
}

func TestRemoveYesSkipsConfirmation(t *testing.T) {
	homeDir, repoDir, _, _, _ := setupRemoveTest(t)

	stdout, err := execRemove(homeDir, repoDir, "", "rmlog12", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")
}

func TestRemoveWithProjectFlag(t *testing.T) {
	homeDir, _, proj, _, _ := setupRemoveTest(t)

	stdout, err := execRemove(homeDir, "", proj.Name, "rmlog12", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, proj.Slug, "rmlog12")
	assert.Error(t, err)
}

func TestRemoveCrossProjectScan(t *testing.T) {
	homeDir := t.TempDir()

	p, err := project.CreateProject(homeDir, "Scannable Remove")
	require.NoError(t, err)

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	e := entry.Entry{
		ID:        "scnrm12",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "scan work",
		CreatedAt: time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	// No repo, no project flag — should scan and find it
	stdout, err := execRemove(homeDir, "", "", "scnrm12", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, p.Slug, "scnrm12")
	assert.Error(t, err)
}

func TestRemoveRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "remove")
}
