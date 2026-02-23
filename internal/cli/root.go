package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := GroupCommand{
		Use:   "hour-git",
		Short: "A Git time-tracking CLI tool",
		Subcommands: []*cobra.Command{
			initCmd,
			versionCmd,
			projectCmd,
			configCmd,
		},
	}.Build()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.CompletionOptions.DisableDefaultCmd = true
	return cmd
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		w := rootCmd.ErrOrStderr()
		msg := Error("error: " + err.Error()) + "\n"
		_, _ = fmt.Fprint(w, msg)
	}
	return err
}
