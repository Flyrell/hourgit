package cli

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupUpdateTest(t *testing.T) (homeDir string, deps updateDeps) {
	t.Helper()
	home := t.TempDir()

	now := time.Date(2026, 2, 26, 12, 0, 0, 0, time.UTC)

	deps = updateDeps{
		now:          func() time.Time { return now },
		isTTY:        func() bool { return true },
		fetchVersion: func() (string, error) { return "v1.0.0", nil },
		readConfig:   project.ReadConfig,
		writeConfig:  project.WriteConfig,
		confirm:      func(_ string) (bool, error) { return false, nil },
		runInstall:   func() error { return nil },
		restartSelf:  func() error { return nil },
		homeDir:      func() (string, error) { return home, nil },
	}

	return home, deps
}

func execUpdateCheck(t *testing.T, version string, deps updateDeps, extraArgs ...string) string {
	t.Helper()
	old := appVersion
	appVersion = version
	t.Cleanup(func() { appVersion = old })

	buf := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetArgs(append([]string{"version"}, extraArgs...))
	checkForUpdate(cmd, deps)
	return buf.String()
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "v1.0.0", 0},
		{"1.0.0", "v1.0.0", 0},
		{"0.9.0", "1.0.0", -1},
		{"1.0.0", "0.9.0", 1},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"0.1.0", "0.0.9", 1},
		{"v0.1.0", "0.2.0", -1},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, compareVersions(tt.a, tt.b))
		})
	}
}

func TestUpdateSkipsDevVersion(t *testing.T) {
	_, deps := setupUpdateTest(t)
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	output := execUpdateCheck(t, "dev", deps)

	assert.Empty(t, output)
	assert.False(t, fetchCalled)
}

func TestUpdateSkipsSkipUpdatesFlag(t *testing.T) {
	_, deps := setupUpdateTest(t)
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	old := appVersion
	appVersion = "0.1.0"
	t.Cleanup(func() { appVersion = old })

	buf := new(bytes.Buffer)
	cmd := newRootCmd()
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"version", "--skip-updates"})
	_ = cmd.ParseFlags([]string{"--skip-updates"})
	checkForUpdate(cmd, deps)

	assert.Empty(t, buf.String())
	assert.False(t, fetchCalled)
}

func TestUpdateSkipsNonTTY(t *testing.T) {
	_, deps := setupUpdateTest(t)
	deps.isTTY = func() bool { return false }
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.Empty(t, output)
	assert.False(t, fetchCalled)
}

func TestUpdateFreshCacheSkipsFetch(t *testing.T) {
	homeDir, deps := setupUpdateTest(t)

	now := deps.now()
	recent := now.Add(-1 * time.Hour)
	require.NoError(t, project.WriteConfig(homeDir, &project.Config{
		LastUpdateCheck: &recent,
		LatestVersion:   "0.1.0", // same as current
	}))

	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.Empty(t, output)
	assert.False(t, fetchCalled)
}

func TestUpdateFreshCacheNewerVersionPromptsUpdate(t *testing.T) {
	homeDir, deps := setupUpdateTest(t)

	now := deps.now()
	recent := now.Add(-1 * time.Hour)
	require.NoError(t, project.WriteConfig(homeDir, &project.Config{
		LastUpdateCheck: &recent,
		LatestVersion:   "2.0.0",
	}))

	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil // skip
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.Contains(t, output, "new version")
	assert.True(t, confirmCalled)
}

func TestUpdateStaleCacheFetchesAndPrompts(t *testing.T) {
	homeDir, deps := setupUpdateTest(t)

	now := deps.now()
	stale := now.Add(-9 * time.Hour)
	require.NoError(t, project.WriteConfig(homeDir, &project.Config{
		LastUpdateCheck: &stale,
		LatestVersion:   "0.1.0",
	}))

	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v2.0.0", nil
	}
	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.True(t, fetchCalled)
	assert.True(t, confirmCalled)
	assert.Contains(t, output, "new version")

	// Verify cache was updated in global config
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", cfg.LatestVersion)
	assert.NotNil(t, cfg.LastUpdateCheck)
}

func TestUpdateNoCacheFetchesAndSkipsWhenSameVersion(t *testing.T) {
	_, deps := setupUpdateTest(t)

	// No config file at all — ReadConfig returns fresh config with defaults
	deps.fetchVersion = func() (string, error) { return "v0.1.0", nil }
	confirmCalled := false
	deps.confirm = func(_ string) (bool, error) {
		confirmCalled = true
		return false, nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.Empty(t, output)
	assert.False(t, confirmCalled)
}

func TestUpdateInstallFlowSuccess(t *testing.T) {
	_, deps := setupUpdateTest(t)

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

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.True(t, installCalled)
	assert.True(t, restartCalled)
	assert.Contains(t, output, "Installing update")
	assert.Contains(t, output, "Restarting")
}

func TestUpdateInstallClearsCachedVersion(t *testing.T) {
	homeDir, deps := setupUpdateTest(t)

	deps.fetchVersion = func() (string, error) { return "v2.0.0", nil }
	deps.confirm = func(_ string) (bool, error) { return true, nil }
	deps.runInstall = func() error { return nil }
	deps.restartSelf = func() error { return nil }

	_ = execUpdateCheck(t, "0.1.0", deps)

	// Cache should be cleared after install
	cfg, err := project.ReadConfig(homeDir)
	require.NoError(t, err)
	assert.Empty(t, cfg.LatestVersion)
	assert.NotNil(t, cfg.LastUpdateCheck)
}

func TestUpdateInstallFlowFailure(t *testing.T) {
	_, deps := setupUpdateTest(t)

	deps.fetchVersion = func() (string, error) { return "v2.0.0", nil }
	deps.confirm = func(_ string) (bool, error) { return true, nil }
	deps.runInstall = func() error { return fmt.Errorf("download failed") }

	restartCalled := false
	deps.restartSelf = func() error {
		restartCalled = true
		return nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.False(t, restartCalled)
	assert.Contains(t, output, "Update failed")
}

func TestUpdateHomeDirErrorSkips(t *testing.T) {
	_, deps := setupUpdateTest(t)
	deps.homeDir = func() (string, error) { return "", fmt.Errorf("no home") }
	fetchCalled := false
	deps.fetchVersion = func() (string, error) {
		fetchCalled = true
		return "v1.0.0", nil
	}

	output := execUpdateCheck(t, "0.1.0", deps)

	assert.Empty(t, output)
	assert.False(t, fetchCalled)
}
