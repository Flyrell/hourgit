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
	initCmd.Flags().String("project", "", "assign repository to a project (creates if needed)")
	initCmd.Flags().Bool("force", false, "overwrite existing post-checkout hook")
	initCmd.Flags().Bool("merge", false, "append to existing post-checkout hook")
}

func runInit(cmd *cobra.Command, dir, homeDir, projectName string, force, merge bool) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "error: not a git repository")
		return fmt.Errorf("not a git repository")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	hookPath := filepath.Join(hooksDir, "post-checkout")

	if existing, err := os.ReadFile(hookPath); err == nil {
		content := string(existing)

		if strings.Contains(content, hookMarker) {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "error: hourgit is already initialized")
			return fmt.Errorf("hourgit is already initialized")
		}

		if !force && !merge {
			_, _ = fmt.Fprintln(cmd.ErrOrStderr(), "error: post-checkout hook already exists (use --force to overwrite or --merge to append)")
			return fmt.Errorf("post-checkout hook already exists")
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
		if cfg != nil && cfg.Project != "" && cfg.Project != projectName {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "error: repository is already assigned to project '%s' (use 'project set --force' to reassign)\n", cfg.Project)
			return fmt.Errorf("repository is already assigned to project '%s'", cfg.Project)
		}

		entry, created, err := project.RegisterProject(homeDir, dir, projectName)
		if err != nil {
			return err
		}
		if created {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "project '%s' created\n", entry.Name)
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "repository assigned to project '%s'\n", projectName)
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "hourgit initialized successfully")
	return nil
}
