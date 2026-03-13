package entry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testActivityStopEntry(id string) ActivityStopEntry {
	return ActivityStopEntry{
		ID:        id,
		Timestamp: time.Date(2025, 6, 15, 10, 15, 0, 0, time.UTC),
		Repo:      "/path/to/repo",
	}
}

func testActivityStartEntry(id string) ActivityStartEntry {
	return ActivityStartEntry{
		ID:        id,
		Timestamp: time.Date(2025, 6, 15, 10, 45, 0, 0, time.UTC),
		Repo:      "/path/to/repo",
	}
}

func TestWriteAndReadActivityStopEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testActivityStopEntry("aad1234")

	require.NoError(t, WriteActivityStopEntry(home, slug, e))

	entries, err := ReadAllActivityStopEntries(home, slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, e.ID, entries[0].ID)
	assert.Equal(t, TypeActivityStop, entries[0].Type)
	assert.Equal(t, e.Timestamp, entries[0].Timestamp)
	assert.Equal(t, e.Repo, entries[0].Repo)
}

func TestWriteAndReadActivityStartEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testActivityStartEntry("aae1234")

	require.NoError(t, WriteActivityStartEntry(home, slug, e))

	entries, err := ReadAllActivityStartEntries(home, slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, e.ID, entries[0].ID)
	assert.Equal(t, TypeActivityStart, entries[0].Type)
	assert.Equal(t, e.Timestamp, entries[0].Timestamp)
	assert.Equal(t, e.Repo, entries[0].Repo)
}

func TestReadAllActivityStopEntriesEmpty(t *testing.T) {
	home := t.TempDir()
	entries, err := ReadAllActivityStopEntries(home, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestReadAllActivityStartEntriesEmpty(t *testing.T) {
	home := t.TempDir()
	entries, err := ReadAllActivityStartEntries(home, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestReadAllActivityStopEntriesSkipsOtherTypes(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("a0a1111", "work")))
	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("c0c1111", "main", "feat")))
	require.NoError(t, WriteActivityStopEntry(home, slug, testActivityStopEntry("e0e1111")))
	require.NoError(t, WriteActivityStartEntry(home, slug, testActivityStartEntry("f0f1111")))

	entries, err := ReadAllActivityStopEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "e0e1111", entries[0].ID)
}

func TestReadAllActivityStartEntriesSkipsOtherTypes(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("a0a1111", "work")))
	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("c0c1111", "main", "feat")))
	require.NoError(t, WriteActivityStopEntry(home, slug, testActivityStopEntry("e0e1111")))
	require.NoError(t, WriteActivityStartEntry(home, slug, testActivityStartEntry("f0f1111")))

	entries, err := ReadAllActivityStartEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "f0f1111", entries[0].ID)
}

func TestReadAllEntriesSkipsActivityEntries(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("a0a1111", "work")))
	require.NoError(t, WriteActivityStopEntry(home, slug, testActivityStopEntry("e0e1111")))
	require.NoError(t, WriteActivityStartEntry(home, slug, testActivityStartEntry("f0f1111")))

	entries, err := ReadAllEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "a0a1111", entries[0].ID)
}

func TestWriteActivityStopEntrySetsTypeField(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteActivityStopEntry(home, slug, testActivityStopEntry("aad1234")))

	entries, err := ReadAllActivityStopEntries(home, slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, TypeActivityStop, entries[0].Type)
}

func TestWriteActivityStartEntrySetsTypeField(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteActivityStartEntry(home, slug, testActivityStartEntry("aae1234")))

	entries, err := ReadAllActivityStartEntries(home, slug)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, TypeActivityStart, entries[0].Type)
}
