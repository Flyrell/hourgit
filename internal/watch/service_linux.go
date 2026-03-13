//go:build linux

package watch

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// NewServiceManager returns the Linux (systemd) service manager.
func NewServiceManager(homeDir string) (ServiceManager, error) {
	return newSystemdManager(homeDir), nil
}

const systemdServiceName = "hourgit-watch"

type systemdManager struct {
	homeDir     string
	servicePath string
}

func newSystemdManager(homeDir string) *systemdManager {
	home, _ := os.UserHomeDir()
	return &systemdManager{
		homeDir:     homeDir,
		servicePath: filepath.Join(home, ".config", "systemd", "user", systemdServiceName+".service"),
	}
}

// ServiceFileContent generates the systemd service unit content.
func ServiceFileContent(binPath string) string {
	return fmt.Sprintf(`[Unit]
Description=Hourgit File Watcher
After=default.target

[Service]
Type=simple
ExecStart=%s watch
Restart=always
RestartSec=5

[Install]
WantedBy=default.target
`, binPath)
}

func (m *systemdManager) Install(binPath string) error {
	dir := filepath.Dir(m.servicePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	content := ServiceFileContent(binPath)
	if err := os.WriteFile(m.servicePath, []byte(content), 0644); err != nil {
		return err
	}
	return exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func (m *systemdManager) Remove() error {
	if m.IsRunning() {
		_ = m.Stop()
	}
	_ = exec.Command("systemctl", "--user", "disable", systemdServiceName).Run()
	err := os.Remove(m.servicePath)
	if os.IsNotExist(err) {
		err = nil
	}
	if err != nil {
		return err
	}
	return exec.Command("systemctl", "--user", "daemon-reload").Run()
}

func (m *systemdManager) Start() error {
	if err := exec.Command("systemctl", "--user", "enable", systemdServiceName).Run(); err != nil {
		return err
	}
	return exec.Command("systemctl", "--user", "start", systemdServiceName).Run()
}

func (m *systemdManager) Stop() error {
	return exec.Command("systemctl", "--user", "stop", systemdServiceName).Run()
}

func (m *systemdManager) IsInstalled() bool {
	_, err := os.Stat(m.servicePath)
	return err == nil
}

func (m *systemdManager) IsRunning() bool {
	out, err := exec.Command("systemctl", "--user", "is-active", systemdServiceName).Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "active"
}
