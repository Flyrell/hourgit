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

func testCheckoutEntry(id, prev, next string) CheckoutEntry {
	return CheckoutEntry{
		ID:        id,
		Timestamp: time.Date(2025, 6, 15, 14, 0, 0, 0, time.UTC),
		Previous:  prev,
		Next:      next,
	}
}

// --- Log entry tests ---

func TestWriteAndReadEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testEntry("abc1234", "did some work")

	require.NoError(t, WriteEntry(home, slug, e))

	got, err := ReadEntry(home, slug, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, "log", got.Type)
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

// --- Checkout entry tests ---

func TestWriteAndReadCheckoutEntry(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"
	e := testCheckoutEntry("abc1234", "main", "feature-x")

	require.NoError(t, WriteCheckoutEntry(home, slug, e))

	got, err := ReadCheckoutEntry(home, slug, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, e.ID, got.ID)
	assert.Equal(t, "checkout", got.Type)
	assert.Equal(t, e.Previous, got.Previous)
	assert.Equal(t, e.Next, got.Next)
	assert.Equal(t, e.Timestamp, got.Timestamp)
}

func TestReadCheckoutEntryNotFound(t *testing.T) {
	home := t.TempDir()
	_, err := ReadCheckoutEntry(home, "test-project", "missing")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadAllCheckoutEntries(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	e1 := testCheckoutEntry("aaa1111", "main", "feat-a")
	e2 := testCheckoutEntry("bbb2222", "feat-a", "feat-b")

	require.NoError(t, WriteCheckoutEntry(home, slug, e1))
	require.NoError(t, WriteCheckoutEntry(home, slug, e2))

	entries, err := ReadAllCheckoutEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 2)
}

func TestReadAllCheckoutEntriesEmpty(t *testing.T) {
	home := t.TempDir()
	entries, err := ReadAllCheckoutEntries(home, "nonexistent")
	require.NoError(t, err)
	assert.Nil(t, entries)
}

// --- Cross-type filtering tests ---

func TestReadAllEntriesSkipsCheckouts(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("log1111", "work")))
	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("chk1111", "main", "feat")))

	entries, err := ReadAllEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "log1111", entries[0].ID)
}

func TestReadAllCheckoutEntriesSkipsLogs(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("log1111", "work")))
	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("chk1111", "main", "feat")))

	entries, err := ReadAllCheckoutEntries(home, slug)
	require.NoError(t, err)
	assert.Len(t, entries, 1)
	assert.Equal(t, "chk1111", entries[0].ID)
}

func TestReadEntryRejectsCheckoutType(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("chk1111", "main", "feat")))

	_, err := ReadEntry(home, slug, "chk1111")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestReadCheckoutEntryRejectsLogType(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("log1111", "work")))

	_, err := ReadCheckoutEntry(home, slug, "log1111")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestWriteEntrySetsTypeField(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteEntry(home, slug, testEntry("abc1234", "work")))

	got, err := ReadEntry(home, slug, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, "log", got.Type)
}

func TestWriteCheckoutEntrySetsTypeField(t *testing.T) {
	home := t.TempDir()
	slug := "test-project"

	require.NoError(t, WriteCheckoutEntry(home, slug, testCheckoutEntry("abc1234", "main", "feat")))

	got, err := ReadCheckoutEntry(home, slug, "abc1234")
	require.NoError(t, err)
	assert.Equal(t, "checkout", got.Type)
}
