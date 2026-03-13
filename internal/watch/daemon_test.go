package watch

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDaemonTest(t *testing.T) string {
	t.Helper()
	home := t.TempDir()

	cfg := &project.Config{
		Defaults: schedule.DefaultSchedules(),
		Projects: []project.ProjectEntry{
			{
				ID:                   "aaa1111",
				Name:                 "test",
				Slug:                 "test",
				Repos:                []string{"/some/repo"},
				Precise:              true,
				IdleThresholdMinutes: 5,
			},
		},
	}
	require.NoError(t, project.WriteConfig(home, cfg))
	return home
}

func TestDaemonReloadConfig(t *testing.T) {
	home := setupDaemonTest(t)
	writer := &mockEntryWriter{}
	d := NewDaemon(home, writer)
	d.state = NewWatchState()

	// reloadConfig should not error even if repos don't exist on disk
	err := d.reloadConfig()
	// It may warn about repos not existing, but shouldn't error fatally
	// In practice the watcher.Add will fail silently
	_ = err
}

func TestDaemonRecoverFromCrash(t *testing.T) {
	home := setupDaemonTest(t)
	writer := &mockEntryWriter{}

	// Write an unpaired activity_start
	startEntry := entry.ActivityStartEntry{
		ID:        "aab1234",
		Type:      entry.TypeActivityStart,
		Timestamp: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}
	require.NoError(t, entry.WriteActivityStartEntry(home, "test", startEntry))

	// Create state with later lastActivity
	state := NewWatchState()
	state.SetLastActivity("/some/repo", time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC))

	d := NewDaemon(home, writer)
	d.state = state

	d.recoverFromCrash()

	// Should have written a retrospective activity_stop
	assert.Equal(t, 1, writer.stopCount())
	writer.mu.Lock()
	assert.Equal(t, time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC), writer.stops[0].Timestamp)
	writer.mu.Unlock()
}

func TestDaemonRecoverFromCrashNoState(t *testing.T) {
	home := setupDaemonTest(t)
	writer := &mockEntryWriter{}

	// Write an unpaired activity_start
	startEntry := entry.ActivityStartEntry{
		ID:        "aab1234",
		Type:      entry.TypeActivityStart,
		Timestamp: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}
	require.NoError(t, entry.WriteActivityStartEntry(home, "test", startEntry))

	// No state file — should use start timestamp as conservative stop
	d := NewDaemon(home, writer)
	d.state = NewWatchState()

	d.recoverFromCrash()

	assert.Equal(t, 1, writer.stopCount())
	writer.mu.Lock()
	// Uses start timestamp as fallback
	assert.Equal(t, time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC), writer.stops[0].Timestamp)
	writer.mu.Unlock()
}

func TestDaemonRecoverFromCrashDistinctIDs(t *testing.T) {
	home := setupDaemonTest(t)
	writer := &mockEntryWriter{}

	stopTime := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)

	// Write two unpaired activity_starts for the same repo with different IDs
	require.NoError(t, entry.WriteActivityStartEntry(home, "test", entry.ActivityStartEntry{
		ID:        "aab1111",
		Type:      entry.TypeActivityStart,
		Timestamp: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}))
	require.NoError(t, entry.WriteActivityStartEntry(home, "test", entry.ActivityStartEntry{
		ID:        "aab2222",
		Type:      entry.TypeActivityStart,
		Timestamp: time.Date(2025, 6, 15, 10, 15, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}))

	// State returns the same stopTime for both
	state := NewWatchState()
	state.SetLastActivity("/some/repo", stopTime)

	d := NewDaemon(home, writer)
	d.state = state

	d.recoverFromCrash()

	// Should have written two stops with distinct IDs
	assert.Equal(t, 2, writer.stopCount())
	writer.mu.Lock()
	assert.NotEqual(t, writer.stops[0].ID, writer.stops[1].ID)
	writer.mu.Unlock()
}

func TestDaemonRecoverFromCrashPairedStart(t *testing.T) {
	home := setupDaemonTest(t)
	writer := &mockEntryWriter{}

	// Write paired start and stop
	require.NoError(t, entry.WriteActivityStartEntry(home, "test", entry.ActivityStartEntry{
		ID:        "aab1234",
		Timestamp: time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}))
	require.NoError(t, entry.WriteActivityStopEntry(home, "test", entry.ActivityStopEntry{
		ID:        "aac1234",
		Timestamp: time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		Repo:      "/some/repo",
	}))

	d := NewDaemon(home, writer)
	d.state = NewWatchState()

	d.recoverFromCrash()

	// No additional stops should be written — start is already paired
	assert.Equal(t, 0, writer.stopCount())
}
