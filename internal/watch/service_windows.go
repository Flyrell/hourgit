//go:build windows

package watch

import (
	"os/exec"
	"strings"
)

// NewServiceManager returns the Windows (schtasks) service manager.
func NewServiceManager(homeDir string) (ServiceManager, error) {
	return newSchtasksManager(homeDir), nil
}

const schtaskName = "HourgitWatch"

type schtasksManager struct {
	homeDir string
}

func newSchtasksManager(homeDir string) *schtasksManager {
	return &schtasksManager{homeDir: homeDir}
}

func (m *schtasksManager) Install(binPath string) error {
	return exec.Command("schtasks", "/create",
		"/tn", schtaskName,
		"/tr", binPath+" watch",
		"/sc", "onlogon",
		"/rl", "limited",
		"/f",
	).Run()
}

func (m *schtasksManager) Remove() error {
	if m.IsRunning() {
		_ = m.Stop()
	}
	return exec.Command("schtasks", "/delete", "/tn", schtaskName, "/f").Run()
}

func (m *schtasksManager) Start() error {
	return exec.Command("schtasks", "/run", "/tn", schtaskName).Run()
}

func (m *schtasksManager) Stop() error {
	return exec.Command("schtasks", "/end", "/tn", schtaskName).Run()
}

func (m *schtasksManager) IsInstalled() bool {
	err := exec.Command("schtasks", "/query", "/tn", schtaskName).Run()
	return err == nil
}

func (m *schtasksManager) IsRunning() bool {
	out, err := exec.Command("schtasks", "/query", "/tn", schtaskName, "/fo", "list", "/v").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Running")
}
