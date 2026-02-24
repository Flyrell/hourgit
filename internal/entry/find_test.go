package entry

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindEntryAcrossProjects(t *testing.T) {
	homeDir := t.TempDir()
	slug := "my-project"

	// Create project directory and write an entry
	dir := filepath.Join(homeDir, ".hourgit", slug)
	require.NoError(t, os.MkdirAll(dir, 0755))

	e := Entry{
		ID:        "abc1234",
		Start:     time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   "test work",
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, WriteEntry(homeDir, slug, e))

	found, err := FindEntryAcrossProjects(homeDir, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, slug, found.Slug)
	assert.Equal(t, "abc1234", found.Entry.ID)
	assert.Equal(t, "test work", found.Entry.Message)
}

func TestFindEntryAcrossProjectsNotFound(t *testing.T) {
	homeDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".hourgit", "some-project"), 0755))

	_, err := FindEntryAcrossProjects(homeDir, "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestFindEntryAcrossProjectsSkipsCheckoutEntries(t *testing.T) {
	homeDir := t.TempDir()
	slug := "my-project"

	dir := filepath.Join(homeDir, ".hourgit", slug)
	require.NoError(t, os.MkdirAll(dir, 0755))

	// Write a checkout entry — should be skipped
	ce := CheckoutEntry{
		ID:        "chk1234",
		Type:      "checkout",
		Timestamp: time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		Previous:  "main",
		Next:      "feature",
	}
	require.NoError(t, WriteCheckoutEntry(homeDir, slug, ce))

	_, err := FindEntryAcrossProjects(homeDir, "chk1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be edited")
}

func TestFindEntryAcrossProjectsMultipleProjects(t *testing.T) {
	homeDir := t.TempDir()

	// Create two project directories
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".hourgit", "project-a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(homeDir, ".hourgit", "project-b"), 0755))

	// Write entry in project-b
	e := Entry{
		ID:        "bbb1234",
		Start:     time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		Minutes:   30,
		Message:   "found in b",
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}
	require.NoError(t, WriteEntry(homeDir, "project-b", e))

	found, err := FindEntryAcrossProjects(homeDir, "bbb1234")
	require.NoError(t, err)
	assert.Equal(t, "project-b", found.Slug)
	assert.Equal(t, "found in b", found.Entry.Message)
}

func TestFindEntryAcrossProjectsNoHourgitDir(t *testing.T) {
	homeDir := t.TempDir()
	// No .hourgit dir at all

	_, err := FindEntryAcrossProjects(homeDir, "abc1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
