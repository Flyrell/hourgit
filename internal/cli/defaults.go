package cli

import "github.com/spf13/cobra"

var defaultsCmd = GroupCommand{
	Use:   "defaults",
	Short: "Manage defaults for new projects",
	Subcommands: []*cobra.Command{
		defaultsScheduleCmd,
	},
}.Build()
