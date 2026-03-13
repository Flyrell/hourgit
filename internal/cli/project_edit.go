package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/watch"
	"github.com/spf13/cobra"
)

var projectEditCmd = LeafCommand{
	Use:   "edit [PROJECT]",
	Short: "Edit project name or tracking mode",
	Args:  cobra.MaximumNArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Shorthand: "y", Usage: "skip confirmation prompts"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Shorthand: "p", Usage: "project name or ID"},
		{Name: "name", Shorthand: "n", Usage: "new project name"},
		{Name: "mode", Shorthand: "m", Usage: "tracking mode: standard or precise"},
		{Name: "idle-threshold", Shorthand: "t", Usage: "idle threshold in minutes (precise mode only)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		repoDir, _ := os.Getwd()

		binPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}
		binPath, err = filepath.EvalSymlinks(binPath)
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		nameFlag, _ := cmd.Flags().GetString("name")
		modeFlag, _ := cmd.Flags().GetString("mode")
		idleThresholdFlag, _ := cmd.Flags().GetString("idle-threshold")
		yes, _ := cmd.Flags().GetBool("yes")

		var idleThreshold int
		if idleThresholdFlag != "" {
			v, err := strconv.Atoi(idleThresholdFlag)
			if err != nil {
				return fmt.Errorf("invalid --idle-threshold value %q: must be a number", idleThresholdFlag)
			}
			if v <= 0 {
				return fmt.Errorf("invalid --idle-threshold value %q: must be greater than 0", idleThresholdFlag)
			}
			idleThreshold = v
		}

		// Resolve project identifier: positional arg > --project flag > repo config
		var identifier string
		if len(args) > 0 {
			identifier = args[0]
		} else if projectFlag != "" {
			identifier = projectFlag
		}

		pk := PromptKit{
			PromptWithDefault: NewPromptWithDefaultFunc(),
			Select:            NewSelectFunc(),
			Confirm:           ResolveConfirmFunc(yes),
		}

		return runProjectEdit(cmd, homeDir, repoDir, identifier, nameFlag, modeFlag, idleThreshold, binPath, pk)
	},
}.Build()

func runProjectEdit(cmd *cobra.Command, homeDir, repoDir, identifier, nameFlag, modeFlag string, idleThreshold int, binPath string, pk PromptKit) error {
	if err := validateMode(modeFlag); err != nil {
		return err
	}

	// Resolve project
	entry, err := resolveEditProject(homeDir, repoDir, identifier)
	if err != nil {
		return err
	}

	newName := nameFlag
	newMode := modeFlag
	newIdleThreshold := idleThreshold

	// Interactive mode: prompt for values if no flags provided
	if nameFlag == "" && modeFlag == "" && idleThreshold == 0 {
		newName, newMode, newIdleThreshold, err = promptProjectEdit(entry, pk)
		if err != nil {
			return err
		}
	}

	// Determine what changed
	nameChanged := newName != "" && newName != entry.Name
	currentMode := "standard"
	if entry.Precise {
		currentMode = "precise"
	}
	modeChanged := newMode != "" && newMode != currentMode

	// Determine effective mode after changes
	effectiveMode := currentMode
	if modeChanged {
		effectiveMode = newMode
	}

	// Idle threshold is only valid for precise mode
	if newIdleThreshold > 0 && effectiveMode != "precise" {
		return fmt.Errorf("--idle-threshold is only valid for precise mode")
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}
	currentThreshold := project.GetIdleThreshold(cfg, entry.ID)
	thresholdChanged := newIdleThreshold > 0 && newIdleThreshold != currentThreshold

	if !nameChanged && !modeChanged && !thresholdChanged {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("no changes"))
		return nil
	}

	// Apply name change
	if nameChanged {
		oldName := entry.Name
		entry, err = project.RenameProject(homeDir, entry.ID, newName)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("name: %s → %s", Silent(oldName), Primary(entry.Name))))
	}

	// Apply mode change
	if modeChanged {
		if newMode == "precise" {
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
		} else {
			if err := project.SetPreciseMode(homeDir, entry.ID, false); err != nil {
				return err
			}
			if err := watch.EnsureWatcherService(homeDir, binPath); err != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "%s\n",
					Warning(fmt.Sprintf("warning: could not configure watcher service: %s", err)))
			}
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("mode: %s → %s", Silent(currentMode), Primary(newMode))))
	}

	// Apply idle threshold change
	if thresholdChanged {
		if err := project.SetIdleThreshold(homeDir, entry.ID, newIdleThreshold); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("idle threshold: %s → %s",
			Silent(fmt.Sprintf("%dm", currentThreshold)), Primary(fmt.Sprintf("%dm", newIdleThreshold)))))
	}

	return nil
}

func resolveEditProject(homeDir, repoDir, identifier string) (*project.ProjectEntry, error) {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	if identifier != "" {
		entry := project.ResolveProject(cfg, identifier)
		if entry == nil {
			return nil, fmt.Errorf("project '%s' not found", identifier)
		}
		return entry, nil
	}

	// Fall back to repo config
	if repoDir != "" {
		repoCfg, err := project.ReadRepoConfig(repoDir)
		if err != nil {
			return nil, err
		}
		if repoCfg != nil {
			entry := project.FindProjectByID(cfg, repoCfg.ProjectID)
			if entry != nil {
				return entry, nil
			}
		}
	}

	return nil, fmt.Errorf("no project specified (use positional arg, --project flag, or run from an assigned repo)")
}

func promptProjectEdit(entry *project.ProjectEntry, pk PromptKit) (name, mode string, idleThreshold int, err error) {
	name, err = pk.PromptWithDefault("Project name", entry.Name)
	if err != nil {
		return "", "", 0, err
	}

	currentMode := 0
	if entry.Precise {
		currentMode = 1
	}
	modes := []string{"standard", "precise"}
	// Pre-select current mode by putting it first
	if currentMode == 1 {
		modes = []string{"precise", "standard"}
	}
	idx, err := pk.Select("Tracking mode", modes)
	if err != nil {
		return "", "", 0, err
	}
	mode = modes[idx]

	// Prompt for idle threshold if mode is/becomes precise
	if mode == "precise" {
		currentThreshold := entry.IdleThresholdMinutes
		if currentThreshold <= 0 {
			currentThreshold = project.DefaultIdleThresholdMinutes
		}
		thresholdStr, err := pk.PromptWithDefault("Idle threshold (minutes)", strconv.Itoa(currentThreshold))
		if err != nil {
			return "", "", 0, err
		}
		v, err := strconv.Atoi(thresholdStr)
		if err != nil || v <= 0 {
			return "", "", 0, fmt.Errorf("invalid idle threshold %q: must be a positive number", thresholdStr)
		}
		idleThreshold = v
	}

	return name, mode, idleThreshold, nil
}
