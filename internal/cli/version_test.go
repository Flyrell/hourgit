package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func execVersion() (string, error) {
	buf := new(bytes.Buffer)
	cmd := versionCmd
	cmd.SetOut(buf)
	err := runVersion(cmd)
	return buf.String(), err
}

func TestVersionDefault(t *testing.T) {
	SetVersionInfo("dev", "none", "unknown")

	out, err := execVersion()

	assert.NoError(t, err)
	assert.Contains(t, out, "hourgit dev (commit: none, built: unknown)")
}

func TestVersionRelease(t *testing.T) {
	SetVersionInfo("1.0.0", "abc1234", "2025-01-01")

	out, err := execVersion()

	assert.NoError(t, err)
	assert.Contains(t, out, "hourgit 1.0.0 (commit: abc1234, built: 2025-01-01)")
}
