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

func setupHistoryTest(t *testing.T) (homeDir string, proj *project.ProjectEntry) {
	t.Helper()
	homeDir = t.TempDir()
	repoDir := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(repoDir, ".git"), 0755))

	proj, err := project.CreateProject(homeDir, "History Test")
	require.NoError(t, err)
	require.NoError(t, project.AssignProject(homeDir, repoDir, proj))

	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	proj = project.FindProjectByID(cfg, proj.ID)

	return homeDir, proj
}

func execHistory(homeDir, projectFlag string, limit int) (string, error) {
	stdout := new(bytes.Buffer)
	cmd := historyCmd
	cmd.SetOut(stdout)

	err := runHistory(cmd, homeDir, projectFlag, limit)
	return stdout.String(), err
}

func TestHistoryNoProjects(t *testing.T) {
	homeDir := t.TempDir()

	stdout, err := execHistory(homeDir, "", 50)

	require.NoError(t, err)
	assert.Contains(t, stdout, "no entries found")
}

func TestHistoryLogEntries(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID:        "abc1234",
		Start:     time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
		Minutes:   150,
		Message:   "Fixed login bug",
		CreatedAt: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}))

	stdout, err := execHistory(homeDir, "", 50)

	require.NoError(t, err)
	assert.Contains(t, stdout, "abc1234")
	assert.Contains(t, stdout, "log")
	assert.Contains(t, stdout, "2h 30m")
	assert.Contains(t, stdout, "Fixed login bug")
	assert.Contains(t, stdout, "History Test")
}

func TestHistoryCheckoutEntries(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, entry.CheckoutEntry{
		ID:        "def5678",
		Timestamp: time.Date(2025, 6, 15, 9, 15, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature-auth",
	}))

	stdout, err := execHistory(homeDir, "", 50)

	require.NoError(t, err)
	assert.Contains(t, stdout, "def5678")
	assert.Contains(t, stdout, "checkout")
	assert.Contains(t, stdout, "main → feature-auth")
	assert.Contains(t, stdout, "History Test")
}

func TestHistoryMixedChronologicalOrder(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	// Older entry
	require.NoError(t, entry.WriteCheckoutEntry(homeDir, proj.Slug, entry.CheckoutEntry{
		ID:        "01de001",
		Timestamp: time.Date(2025, 6, 14, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "develop",
	}))

	// Newer entry
	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID:        "0e0e001",
		Start:     time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "Recent work",
		CreatedAt: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}))

	stdout, err := execHistory(homeDir, "", 50)

	require.NoError(t, err)

	// Newer should appear before older (newest first)
	newerIdx := bytes.Index([]byte(stdout), []byte("0e0e001"))
	olderIdx := bytes.Index([]byte(stdout), []byte("01de001"))
	assert.Greater(t, olderIdx, newerIdx, "newer entry should appear before older entry")
}

func TestHistoryProjectFilter(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	// Create a second project
	proj2, err := project.CreateProject(homeDir, "Other Project")
	require.NoError(t, err)

	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID:        "00f1aaa",
		Start:     time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
		Minutes:   30,
		Message:   "First project work",
		CreatedAt: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}))

	require.NoError(t, entry.WriteEntry(homeDir, proj2.Slug, entry.Entry{
		ID:        "00f2bbb",
		Start:     time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Minutes:   45,
		Message:   "Second project work",
		CreatedAt: time.Date(2025, 6, 15, 13, 0, 0, 0, time.UTC),
	}))

	// Filter to first project only
	stdout, err := execHistory(homeDir, proj.Name, 50)

	require.NoError(t, err)
	assert.Contains(t, stdout, "00f1aaa")
	assert.NotContains(t, stdout, "00f2bbb")
}

func TestHistoryLimit(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	for i := 0; i < 5; i++ {
		require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
			ID:        "e00f00" + string(rune('a'+i)),
			Start:     time.Date(2025, 6, 15, 10+i, 0, 0, 0, time.UTC),
			Minutes:   30,
			Message:   "Work item",
			CreatedAt: time.Date(2025, 6, 15, 10+i, 30, 0, 0, time.UTC),
		}))
	}

	stdout, err := execHistory(homeDir, "", 2)

	require.NoError(t, err)
	// Count lines (each entry = 1 line)
	lines := bytes.Count([]byte(stdout), []byte("\n"))
	assert.Equal(t, 2, lines)
}

func TestHistoryLimitZeroShowsAll(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	for i := 0; i < 5; i++ {
		require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
			ID:        "a11110" + string(rune('a'+i)),
			Start:     time.Date(2025, 6, 15, 10+i, 0, 0, 0, time.UTC),
			Minutes:   30,
			Message:   "Work item",
			CreatedAt: time.Date(2025, 6, 15, 10+i, 30, 0, 0, time.UTC),
		}))
	}

	stdout, err := execHistory(homeDir, "", 0)

	require.NoError(t, err)
	lines := bytes.Count([]byte(stdout), []byte("\n"))
	assert.Equal(t, 5, lines)
}

func TestHistoryLogWithTask(t *testing.T) {
	homeDir, proj := setupHistoryTest(t)

	require.NoError(t, entry.WriteEntry(homeDir, proj.Slug, entry.Entry{
		ID:        "1a50123",
		Start:     time.Date(2025, 6, 15, 11, 0, 0, 0, time.UTC),
		Minutes:   45,
		Message:   "Weekly review",
		Task:      "research",
		CreatedAt: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
	}))

	stdout, err := execHistory(homeDir, "", 50)

	require.NoError(t, err)
	assert.Contains(t, stdout, "[research]")
	assert.Contains(t, stdout, "Weekly review")
	assert.Contains(t, stdout, "45m")
}

func TestHistoryProjectNotFound(t *testing.T) {
	homeDir := t.TempDir()

	_, err := execHistory(homeDir, "nonexistent", 50)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "project 'nonexistent' not found")
}

func TestHistoryRegisteredAsSubcommand(t *testing.T) {
	root := newRootCmd()
	names := make([]string, len(root.Commands()))
	for i, cmd := range root.Commands() {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "history")
}
