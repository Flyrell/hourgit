package watch

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWatchStateSetGetActivity(t *testing.T) {
	s := NewWatchState()
	now := time.Date(2025, 6, 15, 10, 0, 0, 0, time.UTC)

	s.SetLastActivity("/repo/a", now)

	got, ok := s.GetLastActivity("/repo/a")
	assert.True(t, ok)
	assert.Equal(t, now, got)

	_, ok = s.GetLastActivity("/repo/b")
	assert.False(t, ok)
}

func TestWatchStateRemoveRepo(t *testing.T) {
	s := NewWatchState()
	s.SetLastActivity("/repo/a", time.Now())

	s.RemoveRepo("/repo/a")

	_, ok := s.GetLastActivity("/repo/a")
	assert.False(t, ok)
}

func TestWatchStateFlushAndLoad(t *testing.T) {
	home := t.TempDir()
	s := NewWatchState()
	now := time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC)
	s.SetLastActivity("/repo/a", now)

	require.NoError(t, s.Flush(home))

	loaded, err := LoadWatchState(home)
	require.NoError(t, err)
	got, ok := loaded.GetLastActivity("/repo/a")
	assert.True(t, ok)
	assert.Equal(t, now, got)
}

func TestLoadWatchStateMissing(t *testing.T) {
	home := t.TempDir()

	s, err := LoadWatchState(home)
	require.NoError(t, err)
	assert.NotNil(t, s.Repos)
	assert.Empty(t, s.Repos)
}

func TestRemoveState(t *testing.T) {
	home := t.TempDir()
	s := NewWatchState()
	s.SetLastActivity("/repo/a", time.Now())
	require.NoError(t, s.Flush(home))

	require.NoError(t, RemoveState(home))

	loaded, err := LoadWatchState(home)
	require.NoError(t, err)
	assert.Empty(t, loaded.Repos)
}

func TestRemoveStateMissing(t *testing.T) {
	home := t.TempDir()
	assert.NoError(t, RemoveState(home))
}
