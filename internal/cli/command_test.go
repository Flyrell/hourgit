package cli

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLeafCommandBuild(t *testing.T) {
	cmd := LeafCommand{
		Use:   "test",
		Short: "A test command",
		Args:  cobra.ExactArgs(1),
		BoolFlags: []BoolFlag{
			{Name: "verbose", Usage: "enable verbose output", Default: false},
			{Name: "dry-run", Usage: "simulate execution", Default: true},
		},
		StrFlags: []StringFlag{
			{Name: "output", Usage: "output file", Default: "out.txt"},
		},
		RunE: func(cmd *cobra.Command, args []string) error { return nil },
	}.Build()

	assert.Equal(t, "test", cmd.Use)
	assert.Equal(t, "A test command", cmd.Short)
	assert.NotNil(t, cmd.RunE)
	assert.NotNil(t, cmd.Args)

	verbose := cmd.Flags().Lookup("verbose")
	require.NotNil(t, verbose)
	assert.Equal(t, "false", verbose.DefValue)

	dryRun := cmd.Flags().Lookup("dry-run")
	require.NotNil(t, dryRun)
	assert.Equal(t, "true", dryRun.DefValue)

	output := cmd.Flags().Lookup("output")
	require.NotNil(t, output)
	assert.Equal(t, "out.txt", output.DefValue)
}

func TestLeafCommandBuildNoFlags(t *testing.T) {
	cmd := LeafCommand{
		Use:   "simple",
		Short: "A simple command",
		RunE:  func(cmd *cobra.Command, args []string) error { return nil },
	}.Build()

	assert.Equal(t, "simple", cmd.Use)
	assert.False(t, cmd.HasFlags())
}

func TestGroupCommandBuild(t *testing.T) {
	sub1 := &cobra.Command{Use: "sub1"}
	sub2 := &cobra.Command{Use: "sub2"}

	cmd := GroupCommand{
		Use:         "group",
		Short:       "A group command",
		Subcommands: []*cobra.Command{sub1, sub2},
	}.Build()

	assert.Equal(t, "group", cmd.Use)
	assert.Equal(t, "A group command", cmd.Short)
	assert.Nil(t, cmd.RunE)

	names := make([]string, len(cmd.Commands()))
	for i, c := range cmd.Commands() {
		names[i] = c.Name()
	}
	assert.Contains(t, names, "sub1")
	assert.Contains(t, names, "sub2")
}

func TestGroupCommandBuildNoSubcommands(t *testing.T) {
	cmd := GroupCommand{
		Use:   "empty",
		Short: "An empty group",
	}.Build()

	assert.Equal(t, "empty", cmd.Use)
	assert.Empty(t, cmd.Commands())
}
