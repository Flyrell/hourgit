package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/spf13/cobra"
)

var projectAddCmd = LeafCommand{
	Use:   "add PROJECT",
	Short: "Create a new project",
	Args:  cobra.ExactArgs(1),
	StrFlags: []StringFlag{
		{Name: "mode", Usage: "tracking mode: standard or precise (default: standard)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		modeFlag, _ := cmd.Flags().GetString("mode")

		binPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}
		binPath, err = filepath.EvalSymlinks(binPath)
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}

		return runProjectAdd(cmd, homeDir, args[0], modeFlag, binPath)
	},
}.Build()

func runProjectAdd(cmd *cobra.Command, homeDir, name, mode, binPath string) error {
	if err := validateMode(mode); err != nil {
		return err
	}

	entry, err := project.CreateProject(homeDir, name)
	if err != nil {
		return err
	}

	if mode == "precise" {
		if err := project.SetPreciseMode(homeDir, entry.ID, true); err != nil {
			return err
		}
		if err := project.SetIdleThreshold(homeDir, entry.ID, project.DefaultIdleThresholdMinutes); err != nil {
			return err
		}
		if err := watch.EnsureWatcherService(homeDir, binPath); err != nil {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n",
				Warning(fmt.Sprintf("warning: could not configure watcher service: %s", err)))
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
	return nil
}
