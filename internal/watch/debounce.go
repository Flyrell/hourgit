package watch

import (
	"sync"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
)

// EntryWriter abstracts entry writing for testability.
type EntryWriter interface {
	WriteActivityStop(homeDir, slug string, e entry.ActivityStopEntry) error
	WriteActivityStart(homeDir, slug string, e entry.ActivityStartEntry) error
}

// defaultEntryWriter uses the real entry package functions.
type defaultEntryWriter struct{}

func (d defaultEntryWriter) WriteActivityStop(homeDir, slug string, e entry.ActivityStopEntry) error {
	return entry.WriteActivityStopEntry(homeDir, slug, e)
}

func (d defaultEntryWriter) WriteActivityStart(homeDir, slug string, e entry.ActivityStartEntry) error {
	return entry.WriteActivityStartEntry(homeDir, slug, e)
}

// DefaultEntryWriter returns the real entry writer.
func DefaultEntryWriter() EntryWriter {
	return defaultEntryWriter{}
}

// RepoDebouncer manages the debounce state machine for a single repo.
type RepoDebouncer struct {
	mu           sync.Mutex
	repo         string
	slug         string
	homeDir      string
	threshold    time.Duration
	lastActivity time.Time
	idle         bool
	timer        *time.Timer
	writer       EntryWriter
	state        *WatchState
}

// NewRepoDebouncer creates a debouncer for a single repo.
func NewRepoDebouncer(repo, slug, homeDir string, threshold time.Duration, writer EntryWriter, state *WatchState) *RepoDebouncer {
	return &RepoDebouncer{
		repo:      repo,
		slug:      slug,
		homeDir:   homeDir,
		threshold: threshold,
		idle:      true, // start as idle until first file event
		writer:    writer,
		state:     state,
	}
}

// OnFileEvent is called when a file change is detected in the repo.
func (d *RepoDebouncer) OnFileEvent(now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// If idle, write activity_start
	if d.idle {
		d.idle = false
		_ = d.writer.WriteActivityStart(d.homeDir, d.slug, entry.ActivityStartEntry{
			ID:        hashutil.GenerateID(d.repo + now.String()),
			Timestamp: now,
			Repo:      d.repo,
		})
	}

	d.lastActivity = now
	d.state.SetLastActivity(d.repo, now)

	// Reset or start debounce timer
	if d.timer != nil {
		d.timer.Stop()
	}
	d.timer = time.AfterFunc(d.threshold, func() {
		d.onIdle()
	})
}

// onIdle is called when the debounce timer fires (no file changes for threshold duration).
func (d *RepoDebouncer) onIdle() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.idle {
		return
	}

	d.idle = true
	// Write activity_stop with lastActivity timestamp (not current time)
	_ = d.writer.WriteActivityStop(d.homeDir, d.slug, entry.ActivityStopEntry{
		ID:        hashutil.GenerateID(d.repo + d.lastActivity.String()),
		Timestamp: d.lastActivity,
		Repo:      d.repo,
	})
}

// Shutdown writes activity_stop if currently active and stops the timer.
func (d *RepoDebouncer) Shutdown() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.timer != nil {
		d.timer.Stop()
		d.timer = nil
	}

	if !d.idle && !d.lastActivity.IsZero() {
		d.idle = true
		_ = d.writer.WriteActivityStop(d.homeDir, d.slug, entry.ActivityStopEntry{
			ID:        hashutil.GenerateID(d.repo + d.lastActivity.String() + "shutdown"),
			Timestamp: d.lastActivity,
			Repo:      d.repo,
		})
	}
}

// IsIdle returns whether the debouncer is in idle state.
func (d *RepoDebouncer) IsIdle() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.idle
}

// LastActivity returns the last observed file change time.
func (d *RepoDebouncer) LastActivity() time.Time {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastActivity
}
