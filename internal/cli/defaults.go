package cli

import "github.com/spf13/cobra"

var defaultsCmd = GroupCommand{
	Use:   "defaults",
	Short: "Manage default schedule for new projects",
	Subcommands: []*cobra.Command{
		defaultsGetCmd,
		defaultsSetCmd,
		defaultsResetCmd,
		defaultsReadCmd,
	},
}.Build()
