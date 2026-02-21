package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRootHasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()

	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}

	assert.Contains(t, names, "hello")
	assert.Contains(t, names, "version")
}

func TestRootUseName(t *testing.T) {
	assert.Equal(t, "hour-git", rootCmd.Use)
}
