package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const completionMarker = "hourgit completion"

// shellConfig maps shell names to their config file paths (relative to home directory).
var shellConfigs = map[string]string{
	"bash":       ".bashrc",
	"zsh":        ".zshrc",
	"fish":       ".config/fish/config.fish",
	"powershell": ".config/powershell/Microsoft.PowerShell_profile.ps1",
}

// shellEvalLines maps shell names to the line that should be appended to the config file.
var shellEvalLines = map[string]string{
	"bash":       `eval "$(hourgit completion generate bash)"`,
	"zsh":        `eval "$(hourgit completion generate zsh)"`,
	"fish":       `hourgit completion generate fish | source`,
	"powershell": `hourgit completion generate powershell | Out-String | Invoke-Expression`,
}

// detectShell reads the SHELL environment variable and returns a shell name.
func detectShell() string {
	shell := os.Getenv("SHELL")
	base := filepath.Base(shell)
	switch base {
	case "bash":
		return "bash"
	case "zsh":
		return "zsh"
	case "fish":
		return "fish"
	default:
		return ""
	}
}

// isCompletionInstalled checks if the completion eval line is already present in the shell config.
func isCompletionInstalled(shell, homeDir string) bool {
	configPath, ok := shellConfigs[shell]
	if !ok {
		return false
	}
	data, err := os.ReadFile(filepath.Join(homeDir, configPath))
	if err != nil {
		return false
	}
	return strings.Contains(string(data), completionMarker)
}

// installCompletion appends the shell completion eval line to the shell config file.
// Returns nil on success, including when completion is already installed.
func installCompletion(shell, homeDir string) error {
	configRelPath, ok := shellConfigs[shell]
	if !ok {
		return fmt.Errorf("unsupported shell for completion install: %s", shell)
	}

	configPath := filepath.Join(homeDir, configRelPath)

	// Check if already installed
	if data, err := os.ReadFile(configPath); err == nil {
		if strings.Contains(string(data), completionMarker) {
			return nil
		}
	}

	evalLine, ok := shellEvalLines[shell]
	if !ok {
		return fmt.Errorf("unsupported shell for completion install: %s", shell)
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(configPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}

	_, writeErr := fmt.Fprintf(f, "\n# hourgit shell completion\n%s\n", evalLine)
	if closeErr := f.Close(); closeErr != nil {
		return closeErr
	}
	return writeErr
}
