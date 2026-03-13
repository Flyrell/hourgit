//go:build darwin

package watch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewServiceManager returns the macOS (launchd) service manager.
func NewServiceManager(homeDir string) (ServiceManager, error) {
	return newLaunchdManager(homeDir), nil
}

const (
	launchdLabel = "com.hourgit.watch"
)

type launchdManager struct {
	homeDir   string
	plistPath string
}

func newLaunchdManager(homeDir string) *launchdManager {
	home, _ := os.UserHomeDir()
	return &launchdManager{
		homeDir:   homeDir,
		plistPath: filepath.Join(home, "Library", "LaunchAgents", launchdLabel+".plist"),
	}
}

// PlistContent generates the launchd plist XML for the watcher daemon.
func PlistContent(binPath string) string {
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
		<string>watch</string>
	</array>
	<key>KeepAlive</key>
	<true/>
	<key>RunAtLoad</key>
	<true/>
	<key>StandardOutPath</key>
	<string>/tmp/hourgit-watch.log</string>
	<key>StandardErrorPath</key>
	<string>/tmp/hourgit-watch.log</string>
</dict>
</plist>
`, launchdLabel, binPath)
}

func (m *launchdManager) Install(binPath string) error {
	dir := filepath.Dir(m.plistPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := PlistContent(binPath)
	return os.WriteFile(m.plistPath, []byte(content), 0644)
}

func (m *launchdManager) Remove() error {
	// Unload first if loaded
	if m.IsRunning() {
		_ = m.Stop()
	}
	err := os.Remove(m.plistPath)
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (m *launchdManager) Start() error {
	return exec.Command("launchctl", "load", m.plistPath).Run()
}

func (m *launchdManager) Stop() error {
	return exec.Command("launchctl", "unload", m.plistPath).Run()
}

func (m *launchdManager) IsInstalled() bool {
	_, err := os.Stat(m.plistPath)
	return err == nil
}

func (m *launchdManager) IsRunning() bool {
	out, err := exec.Command("launchctl", "list").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), launchdLabel)
}
