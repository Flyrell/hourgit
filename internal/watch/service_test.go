package watch

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockServiceManager implements ServiceManager for testing.
type mockServiceManager struct {
	installed bool
	running   bool
	binPath   string
}

func (m *mockServiceManager) Install(binPath string) error {
	m.installed = true
	m.binPath = binPath
	return nil
}

func (m *mockServiceManager) Remove() error {
	m.installed = false
	m.running = false
	return nil
}

func (m *mockServiceManager) Start() error {
	m.running = true
	return nil
}

func (m *mockServiceManager) Stop() error {
	m.running = false
	return nil
}

func (m *mockServiceManager) IsInstalled() bool { return m.installed }
func (m *mockServiceManager) IsRunning() bool    { return m.running }

func TestMockServiceManagerLifecycle(t *testing.T) {
	sm := &mockServiceManager{}

	assert.False(t, sm.IsInstalled())
	assert.False(t, sm.IsRunning())

	require.NoError(t, sm.Install("/usr/local/bin/hourgit"))
	assert.True(t, sm.IsInstalled())
	assert.Equal(t, "/usr/local/bin/hourgit", sm.binPath)

	require.NoError(t, sm.Start())
	assert.True(t, sm.IsRunning())

	require.NoError(t, sm.Stop())
	assert.False(t, sm.IsRunning())

	require.NoError(t, sm.Remove())
	assert.False(t, sm.IsInstalled())
	assert.False(t, sm.IsRunning())
}

func TestNewServiceManager(t *testing.T) {
	home := t.TempDir()
	sm, err := NewServiceManager(home)

	switch runtime.GOOS {
	case "darwin", "linux", "windows":
		require.NoError(t, err)
		assert.NotNil(t, sm)
	default:
		assert.Error(t, err)
	}
}
