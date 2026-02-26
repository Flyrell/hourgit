package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"
)

const updateCheckTTL = 8 * time.Hour

// updateDeps bundles all side-effects for testability.
type updateDeps struct {
	now          func() time.Time
	isTTY        func() bool
	fetchVersion func() (string, error)
	readConfig   func(homeDir string) (*project.Config, error)
	writeConfig  func(homeDir string, cfg *project.Config) error
	confirm      func(prompt string) (bool, error)
	runInstall   func() error
	restartSelf  func() error
	homeDir      func() (string, error)
}

func defaultUpdateDeps() updateDeps {
	return updateDeps{
		now:          time.Now,
		isTTY:        func() bool { return isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd()) },
		fetchVersion: fetchLatestVersion,
		readConfig:   project.ReadConfig,
		writeConfig:  project.WriteConfig,
		confirm:      NewConfirmFunc(),
		runInstall:   runSelfInstall,
		restartSelf:  restartProcess,
		homeDir:      os.UserHomeDir,
	}
}

func checkForUpdate(cmd *cobra.Command, deps updateDeps) {
	// Guard: skip if --skip-updates
	skipUpdates, _ := cmd.Flags().GetBool("skip-updates")
	if skipUpdates {
		return
	}

	// Guard: skip if dev build
	if appVersion == "dev" {
		return
	}

	// Guard: skip if not a TTY
	if !deps.isTTY() {
		return
	}

	// Resolve home directory for global config
	homeDir, err := deps.homeDir()
	if err != nil {
		return
	}

	// Read global config (includes update cache)
	cfg, err := deps.readConfig(homeDir)
	if err != nil {
		return
	}

	// Check if cache is fresh
	now := deps.now()
	if cfg.LastUpdateCheck != nil && now.Sub(*cfg.LastUpdateCheck) < updateCheckTTL {
		// Cache is fresh — check if the cached version is newer
		if cfg.LatestVersion == "" || compareVersions(appVersion, cfg.LatestVersion) >= 0 {
			return
		}
		// Cached version is newer, show prompt
		promptUpdate(cmd, deps, homeDir, cfg)
		return
	}

	// Cache is stale or missing — fetch latest version
	latest, err := deps.fetchVersion()
	if err != nil {
		return
	}

	// Update cache
	cfg.LastUpdateCheck = &now
	cfg.LatestVersion = latest
	_ = deps.writeConfig(homeDir, cfg)

	// Compare versions
	if compareVersions(appVersion, latest) >= 0 {
		return
	}

	promptUpdate(cmd, deps, homeDir, cfg)
}

func promptUpdate(cmd *cobra.Command, deps updateDeps, homeDir string, cfg *project.Config) {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "\n%s\n", Warning(fmt.Sprintf("A new version of hourgit is available: %s → %s", appVersion, Primary(cfg.LatestVersion))))

	install, err := deps.confirm("Install update now?")
	if err != nil || !install {
		return
	}

	_, _ = fmt.Fprintf(w, "%s\n", Text("Installing update..."))

	if err := deps.runInstall(); err != nil {
		_, _ = fmt.Fprintf(w, "%s\n\n", Error(fmt.Sprintf("Update failed: %s", err)))
		return
	}

	_, _ = fmt.Fprintf(w, "%s\n\n", Text("Update installed. Restarting..."))

	// Clear cached version so the restarted process doesn't re-prompt
	now := deps.now()
	cfg.LastUpdateCheck = &now
	cfg.LatestVersion = ""
	_ = deps.writeConfig(homeDir, cfg)

	if err := deps.restartSelf(); err != nil {
		_, _ = fmt.Fprintf(w, "%s\n\n", Error(fmt.Sprintf("Restart failed: %s", err)))
	}
}

// compareVersions compares two semver strings. Returns -1 if a < b, 0 if equal, 1 if a > b.
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")

	aParts := strings.SplitN(a, ".", 3)
	bParts := strings.SplitN(b, ".", 3)

	for i := 0; i < 3; i++ {
		av, bv := 0, 0
		if i < len(aParts) {
			av, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bv, _ = strconv.Atoi(bParts[i])
		}
		if av < bv {
			return -1
		}
		if av > bv {
			return 1
		}
	}
	return 0
}

// fetchLatestVersion queries GitHub API for the latest release tag.
func fetchLatestVersion() (string, error) {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/Flyrell/hourgit/releases/latest")
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(body, &release); err != nil {
		return "", err
	}

	if release.TagName == "" {
		return "", fmt.Errorf("empty tag_name in GitHub response")
	}

	return release.TagName, nil
}

// runSelfInstall runs the install script to update the binary.
func runSelfInstall() error {
	cmd := exec.Command("bash", "-c", "curl -fsSL https://hourgit.com/install.sh | bash")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// restartProcess replaces the current process with a fresh invocation.
func restartProcess() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return err
	}
	return syscall.Exec(exe, os.Args, os.Environ())
}
