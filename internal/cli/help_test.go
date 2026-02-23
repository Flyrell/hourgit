package cli

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestColorizeLineSectionHeader(t *testing.T) {
	result := colorizeLine("Available Commands:")
	assert.Contains(t, result, "Available Commands:")
}

func TestColorizeLineCommandListing(t *testing.T) {
	result := colorizeLine("  init          Initialize hourgit in a git repository")
	assert.Contains(t, result, "init")
	assert.Contains(t, result, "Initialize hourgit")
}

func TestColorizeLineFlagLine(t *testing.T) {
	result := colorizeLine("  -f, --force   overwrite existing post-checkout hook")
	assert.Contains(t, result, "--force")
	assert.Contains(t, result, "overwrite")
}

func TestColorizeLineFooter(t *testing.T) {
	result := colorizeLine(`Use "hour-git [command] --help" for more information about a command.`)
	assert.Contains(t, result, "hour-git")
}

func TestColorizeLinePlainText(t *testing.T) {
	result := colorizeLine("A Git time-tracking CLI tool")
	assert.Contains(t, result, "A Git time-tracking CLI tool")
}

func TestColorizedHelpFuncProducesOutput(t *testing.T) {
	// Use a standalone command to avoid re-parenting shared subcommands
	cmd := &cobra.Command{
		Use:   "test-app",
		Short: "A test CLI app",
	}
	cmd.AddCommand(&cobra.Command{Use: "sub", Short: "A subcommand"})
	cmd.SetHelpFunc(colorizedHelpFunc())

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	helpFunc := colorizedHelpFunc()
	helpFunc(cmd, []string{})

	output := buf.String()
	assert.Contains(t, output, "test-app")
	assert.Contains(t, output, "Flags:")
}

func TestColorizedHelpFuncRestoresWriter(t *testing.T) {
	cmd := &cobra.Command{
		Use:   "test-app",
		Short: "A test CLI app",
	}
	cmd.SetHelpFunc(colorizedHelpFunc())

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	helpFunc := colorizedHelpFunc()
	helpFunc(cmd, []string{})

	// After help runs, writing should still go to our buffer
	buf.Reset()
	cmd.Print("test")
	assert.Equal(t, "test", buf.String())
}
