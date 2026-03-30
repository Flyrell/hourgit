package e2e

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	// idleThreshold is the idle threshold in seconds used for watcher e2e tests.
	// Keep this short so tests complete quickly, but long enough to avoid flakiness.
	idleThreshold = 2

	// waitForIdle is the time to sleep to ensure the idle threshold fires.
	// Must be > idleThreshold to guarantee the debounce timer completes.
	waitForIdle = 4 * time.Second

	// entryTimeout is the maximum time to wait for activity entries to appear on disk.
	entryTimeout = 15 * time.Second
)

// setupWatcherEnv creates a test env with one precise project and one repo,
// then starts the watcher daemon. Returns the env, watcher manager, and project slug.
func setupWatcherEnv(t *testing.T, projectName string) (*TestEnv, *WatcherManager, string) {
	t.Helper()
	env := NewTestEnv(t)
	env.AddRepo("repo")

	// Init project in precise mode
	env.MustRunInRepo("repo", "init", "--project", projectName, "--mode", "precise", "--yes")

	p := env.FindProject(projectName)
	require.True(t, p.Precise)

	wm := env.StartWatcher(idleThreshold)
	return env, wm, p.Slug
}

func TestPipeline4_WatcherBasicStartStop(t *testing.T) {
	env, wm, slug := setupWatcherEnv(t, "Watcher Basic")

	// Burst of file changes
	env.TouchFiles("repo", 3)

	// Wait for idle threshold to fire
	time.Sleep(waitForIdle)

	// Should have exactly 1 start and 1 stop
	starts, stops := env.WaitForActivityEntries(slug, 1, 1, entryTimeout)

	assert.Len(t, starts, 1)
	assert.Len(t, stops, 1)
	assert.True(t, starts[0].Timestamp.Before(stops[0].Timestamp),
		"start should be before stop")
	assert.Equal(t, env.Repos["repo"].Dir, starts[0].Repo)
	assert.Equal(t, env.Repos["repo"].Dir, stops[0].Repo)

	wm.Stop()
}

func TestPipeline4_WatcherMultipleSessions(t *testing.T) {
	env, wm, slug := setupWatcherEnv(t, "Watcher Sessions")

	// Session 1
	env.TouchFiles("repo", 3)
	time.Sleep(waitForIdle)
	env.WaitForActivityEntries(slug, 1, 1, entryTimeout)

	// Session 2
	env.TouchFiles("repo", 3)
	time.Sleep(waitForIdle)
	starts, stops := env.WaitForActivityEntries(slug, 2, 2, entryTimeout)

	assert.Len(t, starts, 2)
	assert.Len(t, stops, 2)

	// Sort by timestamp for reliable ordering
	sort.Slice(starts, func(i, j int) bool { return starts[i].Timestamp.Before(starts[j].Timestamp) })
	sort.Slice(stops, func(i, j int) bool { return stops[i].Timestamp.Before(stops[j].Timestamp) })

	// Verify chronological ordering: start1 < stop1 < start2 < stop2
	assert.True(t, starts[0].Timestamp.Before(stops[0].Timestamp))
	assert.True(t, stops[0].Timestamp.Before(starts[1].Timestamp))
	assert.True(t, starts[1].Timestamp.Before(stops[1].Timestamp))

	wm.Stop()
}

func TestPipeline4_WatcherGracefulShutdown(t *testing.T) {
	env, wm, slug := setupWatcherEnv(t, "Watcher Shutdown")

	// Trigger activity but do NOT wait for idle
	env.TouchFiles("repo", 3)

	// Brief pause to ensure the daemon has processed at least one event
	time.Sleep(500 * time.Millisecond)

	// Graceful stop — daemon should write final activity_stop
	wm.Stop()

	// Give filesystem a moment to flush
	time.Sleep(500 * time.Millisecond)

	starts := env.ReadActivityStartEntries(slug)
	stops := env.ReadActivityStopEntries(slug)

	assert.Len(t, starts, 1, "should have 1 activity_start from file changes")
	assert.Len(t, stops, 1, "graceful shutdown should write activity_stop")
}

func TestPipeline4_WatcherCrashRecovery(t *testing.T) {
	env, wm, slug := setupWatcherEnv(t, "Watcher Recovery")

	// Trigger activity
	env.TouchFiles("repo", 3)

	// Brief pause then crash (SIGKILL)
	time.Sleep(500 * time.Millisecond)
	wm.Kill()

	// After crash: should have 1 start but 0 stops (daemon didn't clean up)
	time.Sleep(500 * time.Millisecond)
	starts := env.ReadActivityStartEntries(slug)
	stops := env.ReadActivityStopEntries(slug)
	assert.Len(t, starts, 1, "should have 1 activity_start")
	assert.Len(t, stops, 0, "crash should leave no activity_stop")

	// Start a new daemon — crash recovery should pair the unpaired start
	wm2 := env.StartWatcher(idleThreshold)

	// Poll for recovery to complete
	starts, stops = env.WaitForActivityEntries(slug, 1, 1, entryTimeout)
	assert.Len(t, starts, 1, "still 1 activity_start")
	assert.Len(t, stops, 1, "recovery should write retrospective activity_stop")

	wm2.Stop()
}

func TestPipeline4_WatcherMultiRepo(t *testing.T) {
	env := NewTestEnv(t)
	env.AddRepo("repo-a")
	env.AddRepo("repo-b")

	// Init project with first repo
	env.MustRunInRepo("repo-a", "init", "--project", "Watcher Multi", "--mode", "precise", "--yes")
	// Init second repo and assign to same project
	env.MustRunInRepo("repo-b", "init", "--yes")
	env.MustRunInRepo("repo-b", "project", "assign", "Watcher Multi", "--force", "--yes")

	p := env.FindProject("Watcher Multi")
	require.True(t, p.Precise)
	require.Len(t, p.Repos, 2)
	slug := p.Slug

	wm := env.StartWatcher(idleThreshold)

	// Activity in repo-a
	env.TouchFiles("repo-a", 3)
	time.Sleep(waitForIdle)
	env.WaitForActivityEntries(slug, 1, 1, entryTimeout)

	// Activity in repo-b
	env.TouchFiles("repo-b", 3)
	time.Sleep(waitForIdle)
	starts, stops := env.WaitForActivityEntries(slug, 2, 2, entryTimeout)

	assert.Len(t, starts, 2)
	assert.Len(t, stops, 2)

	// Verify repo field distinguishes the two repos
	repos := map[string]bool{}
	for _, s := range starts {
		repos[s.Repo] = true
	}
	assert.True(t, repos[env.Repos["repo-a"].Dir], "should have start for repo-a")
	assert.True(t, repos[env.Repos["repo-b"].Dir], "should have start for repo-b")

	wm.Stop()
}

func TestPipeline4_WatcherIgnoresGitDir(t *testing.T) {
	env, wm, slug := setupWatcherEnv(t, "Watcher Gitignore")

	// Write files inside .git/ — these should be ignored
	env.TouchFile("repo", ".git/test-ignored.txt")
	env.TouchFile("repo", ".git/refs/test.txt")

	// Wait longer than the idle threshold
	time.Sleep(waitForIdle)

	starts := env.ReadActivityStartEntries(slug)
	stops := env.ReadActivityStopEntries(slug)

	assert.Empty(t, starts, "changes in .git/ should not trigger activity_start")
	assert.Empty(t, stops, "changes in .git/ should not trigger activity_stop")

	// Positive control: touch a non-.git file to prove the daemon is alive and watching
	env.TouchFiles("repo", 3)
	time.Sleep(waitForIdle)
	env.WaitForActivityEntries(slug, 1, 1, entryTimeout)

	wm.Stop()
}
