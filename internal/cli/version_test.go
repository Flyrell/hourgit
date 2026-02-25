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
	SetVersionInfo("dev")

	out, err := execVersion()

	assert.NoError(t, err)
	assert.Contains(t, out, "hourgit dev")
}

func TestVersionRelease(t *testing.T) {
	SetVersionInfo("1.0.0")

	out, err := execVersion()

	assert.NoError(t, err)
	assert.Contains(t, out, "hourgit 1.0.0")
}
