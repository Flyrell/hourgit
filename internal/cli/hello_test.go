package cli

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHelloDefault(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"hello"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "Hello, world!\n", buf.String())
}

func TestHelloWithName(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"hello", "Gopher"})
	err := rootCmd.Execute()

	assert.NoError(t, err)
	assert.Equal(t, "Hello, Gopher!\n", buf.String())
}
