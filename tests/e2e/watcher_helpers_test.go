package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/stretchr/testify/require"
)

// WatcherManager manages a hourgit watch daemon process for e2e tests.
type WatcherManager struct {
	env     *TestEnv
	cmd     *exec.Cmd
	stopped bool
}

// StartWatcher starts the hourgit watch daemon as a background process.
// The daemon uses the test env's HOME dir and a short idle threshold.
func (env *TestEnv) StartWatcher(thresholdSeconds int) *WatcherManager {
	env.T.Helper()

	cmd := exec.Command(binaryPath, "--skip-updates", "watch")
	cmd.Env = append(filterHostEnv(),
		"HOME="+env.HomeDir,
		fmt.Sprintf("HOURGIT_IDLE_THRESHOLD=%d", thresholdSeconds),
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(env.T, cmd.Start(), "failed to start watcher daemon")

	wm := &WatcherManager{env: env, cmd: cmd}

	// Register cleanup to ensure process is killed if test fails
	env.T.Cleanup(func() {
		wm.Stop()
	})

	// Wait for PID file to appear (daemon is ready)
	wm.waitForPID()

	return wm
}

// Stop gracefully stops the watcher daemon via SIGTERM.
func (wm *WatcherManager) Stop() {
	if wm.stopped {
		return
	}
	wm.stopped = true

	if wm.cmd.Process == nil {
		return
	}

	// Send SIGTERM for graceful shutdown
	_ = wm.cmd.Process.Signal(syscall.SIGTERM)

	// Wait with timeout
	done := make(chan error, 1)
	go func() { done <- wm.cmd.Wait() }()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		// Force kill if graceful shutdown takes too long
		_ = wm.cmd.Process.Kill()
		<-done
	}
}

// Kill forcefully kills the watcher daemon (simulates crash).
// The PID file and unpaired activity entries will remain.
func (wm *WatcherManager) Kill() {
	if wm.stopped {
		return
	}
	wm.stopped = true

	if wm.cmd.Process == nil {
		return
	}

	_ = wm.cmd.Process.Kill()
	_ = wm.cmd.Wait()
}

// waitForPID polls until the daemon's PID file appears.
func (wm *WatcherManager) waitForPID() {
	wm.env.T.Helper()
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		running, _, err := watch.IsDaemonRunning(wm.env.HomeDir)
		if err == nil && running {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	wm.env.T.Fatal("watcher daemon did not start within 10 seconds")
}

// WaitForActivityEntries polls until the expected number of activity entries appear.
func (env *TestEnv) WaitForActivityEntries(slug string, wantStarts, wantStops int, timeout time.Duration) ([]entry.ActivityStartEntry, []entry.ActivityStopEntry) {
	env.T.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		starts, _ := entry.ReadAllActivityStartEntries(env.HomeDir, slug)
		stops, _ := entry.ReadAllActivityStopEntries(env.HomeDir, slug)
		if len(starts) >= wantStarts && len(stops) >= wantStops {
			return starts, stops
		}
		time.Sleep(200 * time.Millisecond)
	}

	// Final read for error message
	starts, _ := entry.ReadAllActivityStartEntries(env.HomeDir, slug)
	stops, _ := entry.ReadAllActivityStopEntries(env.HomeDir, slug)
	env.T.Fatalf("timed out waiting for activity entries: got %d starts (want %d), %d stops (want %d)",
		len(starts), wantStarts, len(stops), wantStops)
	return nil, nil
}

// TouchFile creates or modifies a file in a repo to trigger fsnotify events.
func (env *TestEnv) TouchFile(repoName, filename string) {
	env.T.Helper()
	repo := env.Repos[repoName]
	require.NotNil(env.T, repo, "repo %q not found", repoName)

	path := filepath.Join(repo.Dir, filename)
	n := fileCounter.Add(1)
	require.NoError(env.T, os.WriteFile(path, []byte(fmt.Sprintf("touch %d\n", n)), 0644))
}

// TouchFiles creates multiple files in quick succession to simulate an activity burst.
func (env *TestEnv) TouchFiles(repoName string, count int) {
	env.T.Helper()
	for i := 0; i < count; i++ {
		env.TouchFile(repoName, fmt.Sprintf("activity-%d.txt", fileCounter.Add(1)))
		time.Sleep(50 * time.Millisecond) // small gap between writes
	}
}
