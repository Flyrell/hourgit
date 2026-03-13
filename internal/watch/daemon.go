package watch

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/fsnotify/fsnotify"
)

const stateFlushInterval = 60 * time.Second

// DaemonConfig holds the daemon's runtime configuration for a single repo.
type DaemonConfig struct {
	Repo      string
	Slug      string
	Threshold time.Duration
}

// Daemon is the central background watcher that monitors repos with precise mode.
type Daemon struct {
	homeDir    string
	writer     EntryWriter
	state      *WatchState
	mu         sync.Mutex
	debouncers map[string]*RepoDebouncer // repo path -> debouncer
	watchers   map[string]*fsnotify.Watcher
	cancel     context.CancelFunc
}

// NewDaemon creates a new daemon instance.
func NewDaemon(homeDir string, writer EntryWriter) *Daemon {
	return &Daemon{
		homeDir:    homeDir,
		writer:     writer,
		debouncers: make(map[string]*RepoDebouncer),
		watchers:   make(map[string]*fsnotify.Watcher),
	}
}

// Run starts the daemon, loads config, sets up watchers, and blocks until stopped.
func (d *Daemon) Run() error {
	// Write PID file
	if err := WritePID(d.homeDir); err != nil {
		return err
	}
	defer func() { _ = RemovePID(d.homeDir) }()

	// Load or create state
	state, err := LoadWatchState(d.homeDir)
	if err != nil {
		state = NewWatchState()
	}
	d.state = state

	// Recover from crash — check for unpaired activity_start entries
	d.recoverFromCrash()

	ctx, cancel := context.WithCancel(context.Background())
	d.cancel = cancel

	// Load initial config and set up watchers
	if err := d.reloadConfig(); err != nil {
		log.Printf("warning: failed to load config: %v", err)
	}

	// Watch config file for changes
	configWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("warning: cannot watch config file: %v", err)
	} else {
		configPath := project.ConfigPath(d.homeDir)
		_ = configWatcher.Add(filepath.Dir(configPath))
		go d.watchConfigChanges(ctx, configWatcher, configPath)
	}

	// State flush ticker
	flushTicker := time.NewTicker(stateFlushInterval)
	defer flushTicker.Stop()
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-flushTicker.C:
				_ = d.state.Flush(d.homeDir)
			}
		}
	}()

	// Wait for shutdown signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case <-sigCh:
	case <-ctx.Done():
	}

	// Graceful shutdown
	d.shutdown()
	if configWatcher != nil {
		_ = configWatcher.Close()
	}

	return nil
}

// Stop signals the daemon to stop.
func (d *Daemon) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
}

// shutdown stops all watchers and writes final activity_stop entries.
func (d *Daemon) shutdown() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, db := range d.debouncers {
		db.Shutdown()
	}
	for _, w := range d.watchers {
		_ = w.Close()
	}
	_ = d.state.Flush(d.homeDir)
	_ = RemoveState(d.homeDir)
}

// reloadConfig reads the config and updates watchers for repos with precise mode.
func (d *Daemon) reloadConfig() error {
	cfg, err := project.ReadConfig(d.homeDir)
	if err != nil {
		return err
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Collect repos that should be watched
	wanted := make(map[string]DaemonConfig)
	for _, p := range cfg.Projects {
		if !p.Precise {
			continue
		}
		threshold := p.IdleThresholdMinutes
		if threshold <= 0 {
			threshold = project.DefaultIdleThresholdMinutes
		}
		for _, repo := range p.Repos {
			wanted[repo] = DaemonConfig{
				Repo:      repo,
				Slug:      p.Slug,
				Threshold: time.Duration(threshold) * time.Minute,
			}
		}
	}

	// Remove watchers for repos no longer wanted
	for repo, db := range d.debouncers {
		if _, ok := wanted[repo]; !ok {
			db.Shutdown()
			if w, ok := d.watchers[repo]; ok {
				_ = w.Close()
			}
			delete(d.debouncers, repo)
			delete(d.watchers, repo)
		}
	}

	// Add watchers for new repos
	for repo, dc := range wanted {
		if _, ok := d.debouncers[repo]; ok {
			continue
		}
		if err := d.addRepoWatcher(dc); err != nil {
			log.Printf("warning: cannot watch %s: %v", repo, err)
		}
	}

	return nil
}

// addRepoWatcher sets up an fsnotify watcher and debouncer for a repo.
func (d *Daemon) addRepoWatcher(dc DaemonConfig) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	// Walk directory tree and add non-.git directories
	err = filepath.WalkDir(dc.Repo, func(path string, info os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible
		}
		if !info.IsDir() {
			return nil
		}
		if ShouldIgnore(dc.Repo, path) {
			return filepath.SkipDir
		}
		return watcher.Add(path)
	})
	if err != nil {
		_ = watcher.Close()
		return err
	}

	db := NewRepoDebouncer(dc.Repo, dc.Slug, d.homeDir, dc.Threshold, d.writer, d.state)
	d.debouncers[dc.Repo] = db
	d.watchers[dc.Repo] = watcher

	go d.watchRepo(watcher, db, dc.Repo)
	return nil
}

// watchRepo processes fsnotify events for a single repo.
func (d *Daemon) watchRepo(watcher *fsnotify.Watcher, db *RepoDebouncer, repoDir string) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			// Only care about writes and creates
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			if ShouldIgnore(repoDir, event.Name) {
				continue
			}
			db.OnFileEvent(time.Now())
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// watchConfigChanges watches for config file changes and reloads.
func (d *Daemon) watchConfigChanges(ctx context.Context, watcher *fsnotify.Watcher, configPath string) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Name != configPath {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create) == 0 {
				continue
			}
			// Small delay to avoid reading partial writes
			time.Sleep(100 * time.Millisecond)
			if err := d.reloadConfig(); err != nil {
				log.Printf("warning: config reload failed: %v", err)
			}
		case _, ok := <-watcher.Errors:
			if !ok {
				return
			}
		}
	}
}

// recoverFromCrash checks for unpaired activity_start entries and writes
// retrospective activity_stop entries using state file timestamps.
func (d *Daemon) recoverFromCrash() {
	cfg, err := project.ReadConfig(d.homeDir)
	if err != nil {
		return
	}

	for _, p := range cfg.Projects {
		if !p.Precise {
			continue
		}

		starts, err := entry.ReadAllActivityStartEntries(d.homeDir, p.Slug)
		if err != nil {
			continue
		}
		stops, err := entry.ReadAllActivityStopEntries(d.homeDir, p.Slug)
		if err != nil {
			continue
		}

		// Build set of stop timestamps to find unpaired starts
		stopTimes := make(map[string]bool)
		for _, s := range stops {
			stopTimes[s.Repo+s.Timestamp.Format(time.RFC3339)] = true
		}

		// Find the latest stop per repo
		latestStop := make(map[string]time.Time)
		for _, s := range stops {
			if s.Timestamp.After(latestStop[s.Repo]) {
				latestStop[s.Repo] = s.Timestamp
			}
		}

		// Check each start for a matching stop after it
		for _, start := range starts {
			stopAfter := latestStop[start.Repo]
			if stopAfter.After(start.Timestamp) || stopAfter.Equal(start.Timestamp) {
				continue // Has a stop after this start
			}

			// Unpaired start — write retrospective stop
			stopTime := start.Timestamp // conservative default
			if lastAct, ok := d.state.GetLastActivity(start.Repo); ok && lastAct.After(start.Timestamp) {
				stopTime = lastAct
			}

			_ = d.writer.WriteActivityStop(d.homeDir, p.Slug, entry.ActivityStopEntry{
				ID:        hashutil.GenerateID(start.Repo + stopTime.String() + "recovery"),
				Timestamp: stopTime,
				Repo:      start.Repo,
			})
		}
	}
}
