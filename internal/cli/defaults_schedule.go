package cli

import "github.com/spf13/cobra"

var defaultsScheduleCmd = GroupCommand{
	Use:   "schedule",
	Short: "Manage default schedule for new projects",
	Subcommands: []*cobra.Command{
		defaultsScheduleGetCmd,
		defaultsScheduleSetCmd,
		defaultsScheduleResetCmd,
		defaultsScheduleReportCmd,
	},
}.Build()
