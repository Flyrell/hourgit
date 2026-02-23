package cli

import "github.com/spf13/cobra"

var projectCmd = GroupCommand{
	Use:   "project",
	Short: "Manage projects",
	Subcommands: []*cobra.Command{
		projectAddCmd,
		projectAssignCmd,
		projectListCmd,
		projectRemoveCmd,
	},
}.Build()
