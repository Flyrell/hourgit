package watch

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// PIDPath returns the path to the PID file.
func PIDPath(homeDir string) string {
	return filepath.Join(homeDir, ".hourgit", "watch.pid")
}

// WritePID writes the current process PID to the PID file.
func WritePID(homeDir string) error {
	path := PIDPath(homeDir)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(os.Getpid())), 0644)
}

// ReadPID reads the PID from the PID file. Returns 0 if the file doesn't exist.
func ReadPID(homeDir string) (int, error) {
	data, err := os.ReadFile(PIDPath(homeDir))
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file: %w", err)
	}
	return pid, nil
}

// IsProcessAlive checks if a process with the given PID is running.
func IsProcessAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks if the process exists without actually sending a signal.
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// RemovePID removes the PID file.
func RemovePID(homeDir string) error {
	err := os.Remove(PIDPath(homeDir))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

// IsDaemonRunning checks if the daemon is running by reading the PID file
// and verifying the process is alive.
func IsDaemonRunning(homeDir string) (bool, int, error) {
	pid, err := ReadPID(homeDir)
	if err != nil {
		return false, 0, err
	}
	if pid == 0 {
		return false, 0, nil
	}
	return IsProcessAlive(pid), pid, nil
}
