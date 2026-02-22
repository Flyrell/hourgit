package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/spf13/cobra"
)

const hookMarker = "# Installed by hourgit"

const hookContent = `#!/bin/sh
# Installed by hourgit
# TODO: hourgit log --type checkout <branch>
echo "hourgit: post-checkout hook triggered"
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize hourgit in a git repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		projectName, _ := cmd.Flags().GetString("project")
		force, _ := cmd.Flags().GetBool("force")
		merge, _ := cmd.Flags().GetBool("merge")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		return runInit(cmd, dir, homeDir, projectName, force, merge)
	},
}

func init() {
	initCmd.Flags().String("project", "", "assign repository to a project by name or ID (creates if needed)")
	initCmd.Flags().Bool("force", false, "overwrite existing post-checkout hook")
	initCmd.Flags().Bool("merge", false, "append to existing post-checkout hook")
}

func runInit(cmd *cobra.Command, dir, homeDir, projectName string, force, merge bool) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	hookPath := filepath.Join(hooksDir, "post-checkout")

	if existing, err := os.ReadFile(hookPath); err == nil {
		content := string(existing)

		if strings.Contains(content, hookMarker) {
			return fmt.Errorf("hourgit is already initialized")
		}

		if !force && !merge {
			return fmt.Errorf("post-checkout hook already exists (use --force to overwrite or --merge to append)")
		}

		if merge {
			merged := content + "\n" + hookContent
			if err := os.WriteFile(hookPath, []byte(merged), 0755); err != nil {
				return err
			}
		} else {
			if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
				return err
			}
		}
	} else {
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
			return err
		}
	}

	if projectName != "" {
		// Check if repo already has a different project
		cfg, err := project.ReadRepoConfig(dir)
		if err != nil {
			return err
		}

		// Resolve identifier to check for match with existing assignment
		reg, err := project.ReadRegistry(homeDir)
		if err != nil {
			return err
		}
		resolved := project.ResolveProject(reg, projectName)
		resolvedName := projectName
		if resolved != nil {
			resolvedName = resolved.Name
		}

		if cfg != nil && cfg.Project != "" && cfg.Project != resolvedName {
			return fmt.Errorf("repository is already assigned to project '%s' (use 'project set --force' to reassign)", cfg.Project)
		}

		entry, created, err := project.RegisterProject(homeDir, dir, projectName)
		if err != nil {
			return err
		}
		if created {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "project '%s' created (%s)\n", entry.Name, entry.ID)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "repository assigned to project '%s'\n", entry.Name)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "hourgit initialized successfully")
	return nil
}
