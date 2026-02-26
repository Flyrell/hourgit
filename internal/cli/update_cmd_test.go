package cli

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateCmdTest(t *testing.T) (string, updateDeps) {
	t.Helper()
	return setupUpdateTest(t)
}

func execUpdateCmd(t *testing.T, version string, deps updateDeps) (string, error) {
	t.Helper()
	old := appVersion
	appVersion = version
	t.Cleanup(func() { appVersion = old })

	buf := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(buf)
	err := runUpdate(cmd, deps)
	return buf.String(), err
}

func TestUpdateCommandDevBuild(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	output, err := execUpdateCmd(t, "dev", deps)

	assert.NoError(t, err)
	assert.Contains(t, output, "Cannot update a dev build")
	assert.False(t, fetchCalled)
}

func TestUpdateCommandAlreadyUpToDate(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "v1.0.0", nil }

	output, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)
	assert.Contains(t, output, "up to date")
	assert.Contains(t, output, "v1.0.0")
}

func TestUpdateCommandNewerVersionAvailable(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "v2.0.0", nil }

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil // skip install
	}

	output, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)
	assert.Contains(t, output, "Checking for updates")
	assert.Contains(t, output, "new version")
	assert.True(t, confirmCalled)
}

func TestUpdateCommandFetchError(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "", fmt.Errorf("network timeout") }

	_, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to check for updates")
	assert.Contains(t, err.Error(), "network timeout")
}

func TestUpdateCommandUpdatesCacheAfterFetch(t *testing.T) {
	homeDir, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "v1.0.0", nil }

	_, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)

	cfg, cfgErr := project.ReadConfig(homeDir)
	require.NoError(t, cfgErr)
	assert.Equal(t, "v1.0.0", cfg.LatestVersion)
	assert.NotNil(t, cfg.LastUpdateCheck)
}

func TestUpdateCommandInstallFlow(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "v2.0.0", nil }
	deps.confirm = func(_ string) (bool, error) { return true, nil }

	installCalled := false
	deps.runInstall = func() error {
		installCalled = true
		return nil
	}

	restartCalled := false
	deps.restartSelf = func() error {
		restartCalled = true
		return nil
	}

	output, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)
	assert.True(t, installCalled)
	assert.True(t, restartCalled)
	assert.Contains(t, output, "Installing update")
	assert.Contains(t, output, "Restarting")
}

func TestUpdateCommandHomeDirError(t *testing.T) {
	_, deps := setupUpdateCmdTest(t)
	deps.fetchVersion = func() (string, error) { return "v1.0.0", nil }
	deps.homeDir = func() (string, error) { return "", fmt.Errorf("no home") }

	output, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)
	assert.Contains(t, output, "up to date")
}

func TestUpdateCommandBypassesCacheTTL(t *testing.T) {
	homeDir, deps := setupUpdateCmdTest(t)

	// Write a fresh cache saying we're up to date
	now := deps.now()
	require.NoError(t, project.WriteConfig(homeDir, &project.Config{
		LastUpdateCheck: &now,
		LatestVersion:   "v1.0.0",
	}))

	// fetchVersion returns a newer version — should be called despite fresh cache
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v2.0.0", nil
	}
	deps.confirm = func(_ string) (bool, error) { return false, nil }

	output, err := execUpdateCmd(t, "v1.0.0", deps)

	assert.NoError(t, err)
	assert.True(t, fetchCalled, "update command should always fetch, bypassing cache TTL")
	assert.Contains(t, output, "new version")
}
