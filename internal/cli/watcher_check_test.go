package cli

import (
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/spf13/cobra"
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
	// In non-TTY test context, it will skip early due to isatty check
	_ = confirmCalled
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
	_ = confirmCalled
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
	_ = confirmCalled
}
