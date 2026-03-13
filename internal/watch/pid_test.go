package watch

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPIDWriteReadRemove(t *testing.T) {
	home := t.TempDir()

	// Write PID
	require.NoError(t, WritePID(home))

	// Read PID
	pid, err := ReadPID(home)
	require.NoError(t, err)
	assert.Equal(t, os.Getpid(), pid)

	// Remove PID
	require.NoError(t, RemovePID(home))

	// Read after remove
	pid, err = ReadPID(home)
	require.NoError(t, err)
	assert.Equal(t, 0, pid)
}

func TestReadPIDMissing(t *testing.T) {
	home := t.TempDir()

	pid, err := ReadPID(home)
	require.NoError(t, err)
	assert.Equal(t, 0, pid)
}

func TestRemovePIDMissing(t *testing.T) {
	home := t.TempDir()
	assert.NoError(t, RemovePID(home))
}

func TestIsProcessAlive(t *testing.T) {
	// Current process should be alive
	assert.True(t, IsProcessAlive(os.Getpid()))

	// PID 0 should not be alive
	assert.False(t, IsProcessAlive(0))

	// Negative PID
	assert.False(t, IsProcessAlive(-1))
}

func TestIsDaemonRunning(t *testing.T) {
	home := t.TempDir()

	// No PID file
	running, pid, err := IsDaemonRunning(home)
	require.NoError(t, err)
	assert.False(t, running)
	assert.Equal(t, 0, pid)

	// Write our own PID (which is alive)
	require.NoError(t, WritePID(home))
	running, pid, err = IsDaemonRunning(home)
	require.NoError(t, err)
	assert.True(t, running)
	assert.Equal(t, os.Getpid(), pid)
}
