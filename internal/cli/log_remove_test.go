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

func setupLogRemoveTest(t *testing.T) (homeDir string, repoDir string, proj *project.ProjectEntry, logEntry entry.Entry, checkoutEntry entry.CheckoutEntry) {
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
		ID:        "0010012",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   120,
		Message:   "test work",
		Task:      "coding",
		CreatedAt: time.Date(2025, 6, 16, 11, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, le))

	ce := entry.CheckoutEntry{
		ID:        "00c0012",
		Type:      "checkout",
		Timestamp: time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature",
	}
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, p.Slug, ce))

	return homeDir, repoDir, p, le, ce
}

func execLogRemove(homeDir, repoDir, projectFlag, hash string, confirm ConfirmFunc) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := logRemoveCmd
	cmd.SetOut(stdout)

	err := runLogRemove(cmd, homeDir, repoDir, projectFlag, hash, confirm)
	return stdout.String(), err
}

func TestLogRemoveLogEntry(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupLogRemoveTest(t)

	stdout, err := execLogRemove(homeDir, repoDir, "", "0010012", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "log")
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, proj.Slug, "0010012")
	assert.Error(t, err)
}

func TestLogRemoveCheckoutEntry(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupLogRemoveTest(t)

	stdout, err := execLogRemove(homeDir, repoDir, "", "00c0012", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "checkout")
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadCheckoutEntry(homeDir, proj.Slug, "00c0012")
	assert.Error(t, err)
}

func TestLogRemoveNotFound(t *testing.T) {
	homeDir, repoDir, _, _, _ := setupLogRemoveTest(t)

	_, err := execLogRemove(homeDir, repoDir, "", "nonexist", AlwaysYes())

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestLogRemoveConfirmDeclined(t *testing.T) {
	homeDir, repoDir, proj, _, _ := setupLogRemoveTest(t)

	confirm := func(_ string) (bool, error) {
		return false, nil
	}

	stdout, err := execLogRemove(homeDir, repoDir, "", "0010012", confirm)

	require.NoError(t, err)
	assert.Contains(t, stdout, "cancelled")
	assert.NotContains(t, stdout, "removed entry")

	// Entry should still exist
	_, err = entry.ReadEntry(homeDir, proj.Slug, "0010012")
	assert.NoError(t, err)
}

func TestLogRemoveYesSkipsConfirmation(t *testing.T) {
	homeDir, repoDir, _, _, _ := setupLogRemoveTest(t)

	stdout, err := execLogRemove(homeDir, repoDir, "", "0010012", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")
}

func TestLogRemoveWithProjectFlag(t *testing.T) {
	homeDir, _, proj, _, _ := setupLogRemoveTest(t)

	stdout, err := execLogRemove(homeDir, "", proj.Name, "0010012", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, proj.Slug, "0010012")
	assert.Error(t, err)
}

func TestLogRemoveCrossProjectScan(t *testing.T) {
	homeDir := t.TempDir()

	p, err := project.CreateProject(homeDir, "Scannable Remove")
	require.NoError(t, err)

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	p = project.FindProjectByID(cfg, p.ID)

	e := entry.Entry{
		ID:        "5c00012",
		Start:     time.Date(2025, 6, 16, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "scan work",
		CreatedAt: time.Date(2025, 6, 16, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, entry.WriteEntry(homeDir, p.Slug, e))

	// No repo, no project flag — should scan and find it
	stdout, err := execLogRemove(homeDir, "", "", "5c00012", AlwaysYes())

	require.NoError(t, err)
	assert.Contains(t, stdout, "removed entry")

	_, err = entry.ReadEntry(homeDir, p.Slug, "5c00012")
	assert.Error(t, err)
}

func TestLogRemoveRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	logGroup := findSubcommand(root, "log")
	require.NotNil(t, logGroup, "log group command should exist")

	names := make([]string, len(logGroup.Commands()))
	for i, cmd := range logGroup.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "remove")
}
