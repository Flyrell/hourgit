package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var appVersion = "dev"

func SetVersionInfo(version string) {
	appVersion = version
}

var versionCmd = LeafCommand{
	Use:   "version",
	Short: "Print the version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runVersion(cmd)
	},
}.Build()

func runVersion(cmd *cobra.Command) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("hourgit %s", appVersion)))
	return nil
}
