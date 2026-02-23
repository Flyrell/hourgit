package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	appVersion = "dev"
	appCommit  = "none"
	appDate    = "unknown"
)

func SetVersionInfo(version, commit, date string) {
	appVersion = version
	appCommit = commit
	appDate = date
}

var versionCmd = LeafCommand{
	Use:   "version",
	Short: "Print the version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("hour-git %s (commit: %s, built: %s)", appVersion, appCommit, appDate)))
		return nil
	},
}.Build()
