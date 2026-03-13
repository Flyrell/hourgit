package cli

import (
	"os"
	"path/filepath"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

// watcherCheckDeps holds injectable dependencies for watcher health check.
type watcherCheckDeps struct {
	homeDir     func() (string, error)
	readConfig  func(string) (*project.Config, error)
	isDaemonRun func(string) (bool, int, error)
	confirm     ConfirmFunc
	binPath     func() (string, error)
	ensureSvc   func(homeDir, binPath string) error
	isTTY       func() bool
}

func defaultWatcherCheckDeps() watcherCheckDeps {
	return watcherCheckDeps{
		homeDir:     os.UserHomeDir,
		readConfig:  project.ReadConfig,
		isDaemonRun: watch.IsDaemonRunning,
		confirm:     NewConfirmFunc(),
		binPath: func() (string, error) {
			p, err := os.Executable()
			if err != nil {
				return "", err
			}
			return filepath.EvalSymlinks(p)
		},
		ensureSvc: watch.EnsureWatcherService,
		isTTY:     func() bool { return isatty.IsTerminal(os.Stdout.Fd()) },
	}
}

// checkWatcherHealth checks if the file watcher daemon is running when needed.
// Called from PersistentPreRunE.
func checkWatcherHealth(cmd *cobra.Command, deps watcherCheckDeps) {
	// Skip in non-interactive contexts
	if !deps.isTTY() {
		return
	}

	skipWatcher, _ := cmd.Flags().GetBool("skip-watcher")
	if skipWatcher {
		return
	}

	if appVersion == "dev" {
		return
	}

	homeDir, err := deps.homeDir()
	if err != nil {
		return
	}

	cfg, err := deps.readConfig(homeDir)
	if err != nil {
		return
	}

	if !project.AnyPreciseProject(cfg) {
		return
	}

	running, _, err := deps.isDaemonRun(homeDir)
	if err != nil || running {
		return
	}

	// Daemon is not running — prompt to restart
	confirmed, err := deps.confirm("File watcher is not running. Restart?")
	if err != nil || !confirmed {
		return
	}

	binPath, err := deps.binPath()
	if err != nil {
		return
	}

	_ = deps.ensureSvc(homeDir, binPath)
}
