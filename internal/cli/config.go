package cli

import "github.com/spf13/cobra"

var configCmd = GroupCommand{
	Use:   "config",
	Short: "Manage project configuration",
	Subcommands: []*cobra.Command{
		configGetCmd,
		configSetCmd,
		configResetCmd,
		configReportCmd,
	},
}.Build()
