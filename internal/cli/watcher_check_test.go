package cli

import (
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCmd() *cobra.Command {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().Bool("skip-watcher", false, "")
	return cmd
}

func setupWatcherCheckTest(t *testing.T, precise bool) (string, watcherCheckDeps) {
	t.Helper()
	home := t.TempDir()

	// Set non-dev version so the dev guard doesn't skip
	SetVersionInfo("1.0.0")
	t.Cleanup(func() { SetVersionInfo("dev") })

	cfg := &project.Config{
		Defaults: schedule.DefaultSchedules(),
		Projects: []project.ProjectEntry{
			{
				ID:      "aaa1111",
				Name:    "test",
				Slug:    "test",
				Repos:   []string{"/some/repo"},
				Precise: precise,
			},
		},
	}
	require.NoError(t, project.WriteConfig(home, cfg))

	deps := watcherCheckDeps{
		homeDir:    func() (string, error) { return home, nil },
		readConfig: project.ReadConfig,
		isDaemonRun: func(_ string) (bool, int, error) {
			return false, 0, nil
		},
		confirm: func(_ string) (bool, error) {
			return false, nil
		},
		binPath: func() (string, error) {
			return "/usr/local/bin/hourgit", nil
		},
		ensureSvc: func(_, _ string) error {
			return nil
		},
		isTTY: func() bool { return true },
	}

	return home, deps
}

func TestWatcherCheckNoPreciseProjects(t *testing.T) {
	_, deps := setupWatcherCheckTest(t, false)

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	cmd := newTestCmd()
	checkWatcherHealth(cmd, deps)
	assert.False(t, confirmCalled, "should not prompt when no precise projects")
}

func TestWatcherCheckDaemonRunning(t *testing.T) {
	_, deps := setupWatcherCheckTest(t, true)

	deps.isDaemonRun = func(_ string) (bool, int, error) {
		return true, 1234, nil
	}

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	cmd := newTestCmd()
	checkWatcherHealth(cmd, deps)
	assert.False(t, confirmCalled, "should not prompt when daemon is running")
}

func TestWatcherCheckSkipFlag(t *testing.T) {
	_, deps := setupWatcherCheckTest(t, true)

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	cmd := newTestCmd()
	_ = cmd.Flags().Set("skip-watcher", "true")
	checkWatcherHealth(cmd, deps)
	assert.False(t, confirmCalled, "should not prompt when skip-watcher flag is set")
}

func TestWatcherCheckPromptRestart(t *testing.T) {
	_, deps := setupWatcherCheckTest(t, true)

	ensureCalled := false
	deps.confirm = func(_ string) (bool, error) {
		return true, nil
	}
	deps.ensureSvc = func(_, _ string) error {
		ensureCalled = true
		return nil
	}

	cmd := newTestCmd()
	checkWatcherHealth(cmd, deps)
	assert.True(t, ensureCalled, "should call ensureSvc when user confirms restart")
}

func TestWatcherCheckNonTTY(t *testing.T) {
	_, deps := setupWatcherCheckTest(t, true)
	deps.isTTY = func() bool { return false }

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	cmd := newTestCmd()
	checkWatcherHealth(cmd, deps)
	assert.False(t, confirmCalled, "should not prompt in non-TTY context")
}
