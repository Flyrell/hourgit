package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/spf13/cobra"
)

var projectAssignCmd = LeafCommand{
	Use:   "assign PROJECT",
	Short: "Assign repository to a project",
	Args:  cobra.ExactArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "force", Usage: "reassign repository to a different project"},
		{Name: "yes", Usage: "skip confirmation prompt"},
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

		var confirm ConfirmFunc
		if yes {
			confirm = AlwaysYes()
		} else {
			confirm = NewConfirmFunc()
		}

		return runProjectAssign(cmd, dir, homeDir, args[0], force, confirm)
	},
}.Build()

func runProjectAssign(cmd *cobra.Command, repoDir, homeDir, projectName string, force bool, confirm ConfirmFunc) error {
	// Check hourgit is initialized
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	hookData, err := os.ReadFile(hookPath)
	if err != nil || !strings.Contains(string(hookData), project.HookMarker) {
		return fmt.Errorf("hourgit is not initialized (run 'hour-git init' first)")
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
		return fmt.Errorf("aborted")
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
