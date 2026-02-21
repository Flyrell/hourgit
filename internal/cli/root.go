package cli

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "hour-git",
	Short: "A Git time-tracking CLI tool",
}

func init() {
	rootCmd.AddCommand(helloCmd)
	rootCmd.AddCommand(versionCmd)
}

func Execute() error {
	return rootCmd.Execute()
}
