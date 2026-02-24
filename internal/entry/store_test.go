package entry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testEntry(id, msg string) Entry {
	return Entry{
		ID:        id,
		Start:     time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		Minutes:   60,
		Message:   msg,
		CreatedAt: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
	}
}

func TestWriteAndReadEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testEntry("abc1234", "did some work")

	require.NoError(t, WriteEntry(home, slug, e))

	got, err := ReadEntry(home, slug, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, e.Minutes, got.Minutes)
	assert.Equal(t, e.Message, got.Message)
}

func TestReadEntryNotFound(t *testing.T) {
	home := t.TempDir()
	_, err := ReadEntry(home, "test-project", "nope")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadAllEntries(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("aaa1111", "first")))
	require.NoError(t, WriteEntry(home, slug, testEntry("bbb2222", "second")))

	entries, err := ReadAllEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestReadAllEntriesEmptyDir(t *testing.T) {
	home := t.TempDir()
	entries, err := ReadAllEntries(home, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

func TestDeleteEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testEntry("abc1234", "to delete")

	require.NoError(t, WriteEntry(home, slug, e))
	require.NoError(t, DeleteEntry(home, slug, "abc1234"))

	_, err := ReadEntry(home, slug, "abc1234")
	assert.Error(t, err)
}

func TestDeleteEntryNotFound(t *testing.T) {
	home := t.TempDir()
	err := DeleteEntry(home, "test-project", "nope")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
