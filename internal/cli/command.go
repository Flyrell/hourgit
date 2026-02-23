package cli

import "github.com/spf13/cobra"

// BoolFlag defines a boolean flag for a command.
type BoolFlag struct {
	Name    string
	Usage   string
	Default bool
}

// StringFlag defines a string flag for a command.
type StringFlag struct {
	Name    string
	Usage   string
	Default string
}

// LeafCommand defines a command that executes logic.
// Every leaf command file must declare one of these and call Build().
type LeafCommand struct {
	Use       string
	Short     string
	Args      cobra.PositionalArgs
	BoolFlags []BoolFlag
	StrFlags  []StringFlag
	RunE      func(cmd *cobra.Command, args []string) error
}

// Build creates a cobra.Command with all flags registered.
func (lc LeafCommand) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:   lc.Use,
		Short: lc.Short,
		Args:  lc.Args,
		RunE:  lc.RunE,
	}
	for _, f := range lc.BoolFlags {
		cmd.Flags().Bool(f.Name, f.Default, f.Usage)
	}
	for _, f := range lc.StrFlags {
		cmd.Flags().String(f.Name, f.Default, f.Usage)
	}
	return cmd
}

// GroupCommand defines a command that only holds subcommands.
type GroupCommand struct {
	Use         string
	Short       string
	Subcommands []*cobra.Command
}

// Build creates a cobra.Command with all subcommands registered.
func (gc GroupCommand) Build() *cobra.Command {
	cmd := &cobra.Command{
		Use:   gc.Use,
		Short: gc.Short,
	}
	for _, sub := range gc.Subcommands {
		cmd.AddCommand(sub)
	}
	return cmd
}
