package cli

import "github.com/spf13/cobra"

var scheduleCmd = GroupCommand{
	Use:   "schedule",
	Short: "Manage project schedule",
	Subcommands: []*cobra.Command{
		scheduleGetCmd,
		scheduleSetCmd,
		scheduleResetCmd,
		scheduleReportCmd,
	},
}.Build()
