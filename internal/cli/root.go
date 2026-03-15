package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var rootCmd = newRootCmd()

func newRootCmd() *cobra.Command {
	cmd := GroupCommand{
		Use:   "hourgit",
		Short: "A Git time-tracking CLI tool",
		Subcommands: []*cobra.Command{
			initCmd,
			logCmd,
			editCmd,
			removeCmd,
			syncCmd,
			reportCmd,
			historyCmd,
			statusCmd,
			versionCmd,
			projectCmd,
			defaultsCmd,
			completionCmd,
			updateCmd,
			watchCmd,
		},
	}.Build()
	cmd.SilenceUsage = true
	cmd.SilenceErrors = true
	cmd.CompletionOptions.DisableDefaultCmd = true
	cmd.SetHelpFunc(colorizedHelpFunc())
	cmd.PersistentFlags().Bool("skip-updates", false, "skip the automatic update check")
	cmd.PersistentFlags().Bool("skip-watcher", false, "skip the file watcher health check")
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		checkForUpdate(cmd, defaultUpdateDeps())
		checkWatcherHealth(cmd, defaultWatcherCheckDeps())
		return nil
	}
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
