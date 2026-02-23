package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectRegisteredAsSubcommand(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "project")
}
