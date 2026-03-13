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
		return runProjectAdd(cmd, homeDir, args[0], modeFlag)
	},
}.Build()

func runProjectAdd(cmd *cobra.Command, homeDir, name, mode string) error {
	if mode != "" && mode != "standard" && mode != "precise" {
		return fmt.Errorf("invalid --mode value %q (supported: standard, precise)", mode)
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
		binPath, _ := os.Executable()
		binPath, _ = filepath.EvalSymlinks(binPath)
		_ = watch.EnsureWatcherService(homeDir, binPath)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
	return nil
}
