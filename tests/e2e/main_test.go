package e2e

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

var binaryPath string

// skipE2E is set to true when prerequisites (git) are missing.
var skipE2E bool

func TestMain(m *testing.M) {
	// Check prerequisites — git must be available for e2e tests
	if _, err := exec.LookPath("git"); err != nil {
		fmt.Println("e2e: skipping — git not found in PATH")
		skipE2E = true
		os.Exit(m.Run())
	}

	tmp, err := os.MkdirTemp("", "hourgit-e2e-bin-*")
	if err != nil {
		panic("failed to create temp dir for binary: " + err.Error())
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	binaryPath = filepath.Join(tmp, "hourgit")

	// Build the binary from the project root
	build := exec.Command("go", "build", "-ldflags", "-X main.version=e2e-test", "-o", binaryPath, "./cmd/hourgit")
	build.Dir = filepath.Join("..", "..")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		panic("failed to build hourgit binary: " + err.Error())
	}

	os.Exit(m.Run())
}
