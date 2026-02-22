package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "hour-git",
	Short:         "A Git time-tracking CLI tool",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(initCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(projectCmd)
}

func Execute() error {
	err := rootCmd.Execute()
	if err != nil {
		w := rootCmd.ErrOrStderr()
		msg := "error: " + err.Error()
		if f, ok := w.(*os.File); ok && isTerminal(f) {
			msg = "\033[31m" + msg + "\033[0m"
		}
		msg += "\n"
		_, _ = fmt.Fprint(w, msg)
	}
	return err
}
