package watch

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// StatePath returns the path to the state file.
func StatePath(homeDir string) string {
	return filepath.Join(homeDir, ".hourgit", "watch.state")
}

// RepoState holds the last activity time for a single repo.
type RepoState struct {
	LastActivity time.Time `json:"last_activity"`
}

// WatchState holds the daemon's state, flushed periodically to disk.
type WatchState struct {
	mu    sync.Mutex
	Repos map[string]RepoState `json:"repos"`
}

// NewWatchState creates a new empty WatchState.
func NewWatchState() *WatchState {
	return &WatchState{
		Repos: make(map[string]RepoState),
	}
}

// SetLastActivity updates the last activity time for a repo.
func (s *WatchState) SetLastActivity(repo string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Repos[repo] = RepoState{LastActivity: t}
}

// GetLastActivity returns the last activity time for a repo.
func (s *WatchState) GetLastActivity(repo string) (time.Time, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rs, ok := s.Repos[repo]
	return rs.LastActivity, ok
}

// RemoveRepo removes a repo from the state.
func (s *WatchState) RemoveRepo(repo string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.Repos, repo)
}

// Flush writes the state to disk.
func (s *WatchState) Flush(homeDir string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	path := StatePath(homeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// LoadWatchState reads the state from disk. Returns a new empty state if the file doesn't exist.
func LoadWatchState(homeDir string) (*WatchState, error) {
	data, err := os.ReadFile(StatePath(homeDir))
	if os.IsNotExist(err) {
		return NewWatchState(), nil
	}
	if err != nil {
		return nil, err
	}
	var s WatchState
	if err := json.Unmarshal(data, &s); err != nil {
		return NewWatchState(), nil
	}
	if s.Repos == nil {
		s.Repos = make(map[string]RepoState)
	}
	return &s, nil
}

// RemoveState removes the state file.
func RemoveState(homeDir string) error {
	err := os.Remove(StatePath(homeDir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}
