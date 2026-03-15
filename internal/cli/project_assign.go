package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var projectAssignCmd = LeafCommand{
	Use:   "assign [PROJECT]",
	Short: "Assign repository to a project",
	Args:  cobra.MaximumNArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "force", Shorthand: "f", Usage: "reassign repository to a different project"},
		{Name: "yes", Shorthand: "y", Usage: "skip confirmation prompt"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Shorthand: "p", Usage: "project name or ID"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		force, _ := cmd.Flags().GetBool("force")
		yes, _ := cmd.Flags().GetBool("yes")
		projectFlag, _ := cmd.Flags().GetString("project")

		confirm := ResolveConfirmFunc(yes)

		// Resolve project name: positional arg > --project flag > repo config
		var projectName string
		if len(args) > 0 {
			projectName = args[0]
		} else if projectFlag != "" {
			projectName = projectFlag
		} else {
			// Fall back to repo config
			entry, err := resolveProjectFromRepo(homeDir, dir)
			if err != nil {
				return err
			}
			projectName = entry.Name
		}

		return runProjectAssign(cmd, dir, homeDir, projectName, force, confirm)
	},
}.Build()

func runProjectAssign(cmd *cobra.Command, repoDir, homeDir, projectName string, force bool, confirm ConfirmFunc) error {
	// Check hourgit is initialized
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	hookData, err := os.ReadFile(hookPath)
	if err != nil || !strings.Contains(string(hookData), project.HookMarker) {
		return fmt.Errorf("hourgit is not initialized (run 'hourgit init' first)")
	}

	// Check existing repo config
	cfg, err := project.ReadRepoConfig(repoDir)
	if err != nil {
		return err
	}

	// Resolve project (may prompt to create)
	result, err := project.ResolveOrCreate(homeDir, projectName, func(name string) (bool, error) {
		return confirm(fmt.Sprintf("Project '%s' does not exist. Create it?", name))
	})
	if err != nil {
		return err
	}
	if result == nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
		return nil
	}

	// Check existing assignment
	if cfg != nil && cfg.Project != "" {
		if cfg.Project == result.Entry.Name {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository is already assigned to project '%s'", Primary(cfg.Project))))
			return nil
		}

		if !force {
			return fmt.Errorf("repository is already assigned to project '%s' (use --force to reassign)", cfg.Project)
		}

		// Remove repo from old project
		appCfg, err := project.ReadConfig(homeDir)
		if err != nil {
			return err
		}
		oldEntry := project.FindProject(appCfg, cfg.Project)
		if oldEntry != nil {
			project.RemoveRepoFromProject(oldEntry, repoDir)
			if err := project.WriteConfig(homeDir, appCfg); err != nil {
				return err
			}
		}
	}

	if result.Created {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(result.Entry.Name), Silent(result.Entry.ID))))
	}

	if err := project.AssignProject(homeDir, repoDir, result.Entry); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository assigned to project '%s'", Primary(result.Entry.Name))))
	return nil
}
