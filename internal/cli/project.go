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
	Use:   "set PROJECT_NAME",
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

func init() {
	projectSetCmd.Flags().Bool("force", false, "reassign repository to a different project")
	projectCmd.AddCommand(projectSetCmd)
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

	if cfg != nil && cfg.Project != "" {
		if cfg.Project == projectName {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "repository is already assigned to project '%s'\n", projectName)
			return nil
		}

		if !force {
			return fmt.Errorf("repository is already assigned to project '%s' (use --force to reassign)", cfg.Project)
		}

		// Remove repo from old project
		reg, err := project.ReadRegistry(homeDir)
		if err != nil {
			return err
		}
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
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "project '%s' created\n", entry.Name)
	}
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "repository assigned to project '%s'\n", projectName)
	return nil
}
