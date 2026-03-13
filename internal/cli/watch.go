package cli

import (
	"os"

	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/spf13/cobra"
)

type daemonRunner func(homeDir string) error

func defaultDaemonRunner(homeDir string) error {
	d := watch.NewDaemon(homeDir, watch.DefaultEntryWriter())
	return d.Run()
}

var watchCmd = LeafCommand{
	Use:   "watch",
	Short: "Run the file watcher daemon (used by the OS service)",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runWatch(homeDir, defaultDaemonRunner)
	},
}.Build()

func runWatch(homeDir string, runner daemonRunner) error {
	return runner(homeDir)
}
