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
			confirm = NewConfirmFunc(cmd.InOrStdin(), cmd.OutOrStdout())
		}

		return runProjectAssign(cmd, dir, homeDir, args[0], force, confirm)
	},
}.Build()

func runProjectAssign(cmd *cobra.Command, repoDir, homeDir, projectName string, force bool, confirm ConfirmFunc) error {
	// Check hourgit is initialized
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	hookData, err := os.ReadFile(hookPath)
	if err != nil || !strings.Contains(string(hookData), hookMarker) {
		return fmt.Errorf("hourgit is not initialized (run 'hour-git init' first)")
	}

	// Check existing repo config
	cfg, err := project.ReadRepoConfig(repoDir)
	if err != nil {
		return err
	}

	// Resolve identifier (could be ID or name)
	reg, err := project.ReadRegistry(homeDir)
	if err != nil {
		return err
	}
	resolved := project.ResolveProject(reg, projectName)
	resolvedName := projectName
	if resolved != nil {
		resolvedName = resolved.Name
	}

	if cfg != nil && cfg.Project != "" {
		if cfg.Project == resolvedName {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository is already assigned to project '%s'", Primary(cfg.Project))))
			return nil
		}

		if !force {
			return fmt.Errorf("repository is already assigned to project '%s' (use --force to reassign)", cfg.Project)
		}

		// Remove repo from old project
		oldEntry := project.FindProject(reg, cfg.Project)
		if oldEntry != nil {
			project.RemoveRepoFromProject(oldEntry, repoDir)
			if err := project.WriteRegistry(homeDir, reg); err != nil {
				return err
			}
		}
	}

	// If project doesn't exist, prompt to create it
	var entry *project.ProjectEntry
	if resolved == nil {
		prompt := fmt.Sprintf("Project '%s' does not exist. Create it?", projectName)
		confirmed, err := confirm(prompt)
		if err != nil {
			return err
		}
		if !confirmed {
			return fmt.Errorf("aborted")
		}

		entry, err = project.CreateProject(homeDir, projectName)
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
	} else {
		entry = resolved
	}

	if err := project.AssignProject(homeDir, repoDir, entry); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository assigned to project '%s'", Primary(entry.Name))))
	return nil
}
