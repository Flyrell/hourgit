package cli

import "github.com/spf13/cobra"

var logCmd = GroupCommand{
	Use:   "log",
	Short: "Log and manage time entries",
	Subcommands: []*cobra.Command{
		logAddCmd,
		logEditCmd,
		logRemoveCmd,
	},
}.Build()
