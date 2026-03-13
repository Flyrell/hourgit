package watch

import (
	"sync"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockEntryWriter records written entries for assertions.
type mockEntryWriter struct {
	mu     sync.Mutex
	stops  []entry.ActivityStopEntry
	starts []entry.ActivityStartEntry
}

func (m *mockEntryWriter) WriteActivityStop(_ string, _ string, e entry.ActivityStopEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.stops = append(m.stops, e)
	return nil
}

func (m *mockEntryWriter) WriteActivityStart(_ string, _ string, e entry.ActivityStartEntry) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.starts = append(m.starts, e)
	return nil
}

func (m *mockEntryWriter) stopCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.stops)
}

func (m *mockEntryWriter) startCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.starts)
}

func TestDebouncerFirstEventWritesStart(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 100*time.Millisecond, writer, state)

	now := time.Now()
	db.OnFileEvent(now)

	assert.Equal(t, 1, writer.startCount())
	assert.Equal(t, 0, writer.stopCount())
	assert.False(t, db.IsIdle())

	// Cleanup
	db.Shutdown()
}

func TestDebouncerIdleAfterThreshold(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 50*time.Millisecond, writer, state)

	now := time.Now()
	db.OnFileEvent(now)

	// Wait for debounce timer to fire
	time.Sleep(100 * time.Millisecond)

	assert.Equal(t, 1, writer.startCount())
	assert.Equal(t, 1, writer.stopCount())
	assert.True(t, db.IsIdle())

	// Stop timestamp should be the lastActivity, not the timer fire time
	writer.mu.Lock()
	assert.Equal(t, now, writer.stops[0].Timestamp)
	writer.mu.Unlock()
}

func TestDebouncerResetOnActivity(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 80*time.Millisecond, writer, state)

	t1 := time.Now()
	db.OnFileEvent(t1)

	// Send another event before threshold
	time.Sleep(40 * time.Millisecond)
	t2 := time.Now()
	db.OnFileEvent(t2)

	// Should not have gone idle yet
	assert.Equal(t, 0, writer.stopCount())

	// Wait for debounce timer
	time.Sleep(120 * time.Millisecond)

	assert.Equal(t, 1, writer.stopCount())
	assert.Equal(t, 1, writer.startCount()) // Only one start since we never went idle
}

func TestDebouncerIdleToActiveTransition(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 50*time.Millisecond, writer, state)

	// First activity
	db.OnFileEvent(time.Now())
	time.Sleep(80 * time.Millisecond) // Go idle

	require.Equal(t, 1, writer.startCount())
	require.Equal(t, 1, writer.stopCount())

	// Resume activity
	db.OnFileEvent(time.Now())

	assert.Equal(t, 2, writer.startCount())
	assert.False(t, db.IsIdle())

	db.Shutdown()
}

func TestDebouncerShutdownWritesStop(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 10*time.Second, writer, state)

	now := time.Now()
	db.OnFileEvent(now)
	assert.False(t, db.IsIdle())

	db.Shutdown()

	assert.Equal(t, 1, writer.stopCount())
	assert.True(t, db.IsIdle())
}

func TestDebouncerShutdownIdleNoOp(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 10*time.Second, writer, state)

	// Never active, shutdown should not write stop
	db.Shutdown()
	assert.Equal(t, 0, writer.stopCount())
}

func TestDebouncerUpdatesState(t *testing.T) {
	writer := &mockEntryWriter{}
	state := NewWatchState()
	db := NewRepoDebouncer("/repo", "test", "/home", 10*time.Second, writer, state)

	now := time.Now()
	db.OnFileEvent(now)

	got, ok := state.GetLastActivity("/repo")
	assert.True(t, ok)
	assert.Equal(t, now, got)

	db.Shutdown()
}
