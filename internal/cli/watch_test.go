package cli

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunWatchCallsRunner(t *testing.T) {
	called := false
	runner := func(homeDir string) error {
		called = true
		assert.Equal(t, "/test/home", homeDir)
		return nil
	}

	err := runWatch("/test/home", runner)

	assert.NoError(t, err)
	assert.True(t, called)
}

func TestRunWatchPropagatesError(t *testing.T) {
	runner := func(homeDir string) error {
		return fmt.Errorf("daemon failed")
	}

	err := runWatch("/test/home", runner)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "daemon failed")
}

func TestWatchRegistered(t *testing.T) {
	commands := rootCmd.Commands()
	names := make([]string, len(commands))
	for i, cmd := range commands {
		names[i] = cmd.Name()
	}
	assert.Contains(t, names, "watch")
}
