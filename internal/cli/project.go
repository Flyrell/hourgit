package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage project assignments",
}

var projectSetCmd = &cobra.Command{
	Use:   "set PROJECT",
	Short: "Assign repository to a project",
	Args:  cobra.ExactArgs(1),
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

		return runProjectSet(cmd, dir, homeDir, args[0], force)
	},
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all projects and their repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runProjectList(cmd, homeDir)
	},
}

func init() {
	projectSetCmd.Flags().Bool("force", false, "reassign repository to a different project")
	projectCmd.AddCommand(projectSetCmd)
	projectCmd.AddCommand(projectListCmd)
}

func runProjectList(cmd *cobra.Command, homeDir string) error {
	reg, err := project.ReadRegistry(homeDir)
	if err != nil {
		return err
	}

	if len(reg.Projects) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), Silent("No projects found."))
		return nil
	}

	for i, p := range reg.Projects {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", Silent(p.ID), Primary(p.Name))
		if len(p.Repos) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), Silent("└── (no repositories assigned)"))
		} else {
			for j, r := range p.Repos {
				if j < len(p.Repos)-1 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("├── %s", r)))
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("└── %s", r)))
				}
			}
		}
		if i < len(reg.Projects)-1 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}

func runProjectSet(cmd *cobra.Command, repoDir, homeDir, projectName string, force bool) error {
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

	entry, created, err := project.RegisterProject(homeDir, repoDir, projectName)
	if err != nil {
		return err
	}

	if created {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository assigned to project '%s'", Primary(entry.Name))))
	return nil
}
