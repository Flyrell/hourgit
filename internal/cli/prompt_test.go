package cli

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfirmFuncYes(t *testing.T) {
	in := strings.NewReader("y\n")
	out := new(bytes.Buffer)
	confirm := NewConfirmFunc(in, out)

	result, err := confirm("Do it?")

	require.NoError(t, err)
	assert.True(t, result)
	assert.Contains(t, out.String(), "Do it? [y/N]")
}

func TestNewConfirmFuncYesFullWord(t *testing.T) {
	in := strings.NewReader("yes\n")
	out := new(bytes.Buffer)
	confirm := NewConfirmFunc(in, out)

	result, err := confirm("Do it?")

	require.NoError(t, err)
	assert.True(t, result)
}

func TestNewConfirmFuncNo(t *testing.T) {
	in := strings.NewReader("n\n")
	out := new(bytes.Buffer)
	confirm := NewConfirmFunc(in, out)

	result, err := confirm("Do it?")

	require.NoError(t, err)
	assert.False(t, result)
}

func TestNewConfirmFuncEmpty(t *testing.T) {
	in := strings.NewReader("\n")
	out := new(bytes.Buffer)
	confirm := NewConfirmFunc(in, out)

	result, err := confirm("Do it?")

	require.NoError(t, err)
	assert.False(t, result)
}

func TestNewConfirmFuncEOF(t *testing.T) {
	in := strings.NewReader("")
	out := new(bytes.Buffer)
	confirm := NewConfirmFunc(in, out)

	result, err := confirm("Do it?")

	require.NoError(t, err)
	assert.False(t, result)
}

func TestAlwaysYes(t *testing.T) {
	confirm := AlwaysYes()

	result, err := confirm("anything")

	require.NoError(t, err)
	assert.True(t, result)
}
