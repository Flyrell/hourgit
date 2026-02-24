package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersionDefault(t *testing.T) {
	SetVersionInfo("dev", "none", "unknown")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "hourgit dev (commit: none, built: unknown)\n", buf.String())
}

func TestVersionRelease(t *testing.T) {
	SetVersionInfo("1.0.0", "abc1234", "2025-01-01")

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"version"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "hourgit 1.0.0 (commit: abc1234, built: 2025-01-01)\n", buf.String())
}
