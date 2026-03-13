package cli

import (
	"os"

	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/spf13/cobra"
)

var watchCmd = LeafCommand{
	Use:   "watch",
	Short: "Run the file watcher daemon (used by the OS service)",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runWatch(homeDir)
	},
}.Build()

func runWatch(homeDir string) error {
	d := watch.NewDaemon(homeDir, watch.DefaultEntryWriter())
	return d.Run()
}
