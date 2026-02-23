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

var initCmd = LeafCommand{
	Use:   "init",
	Short: "Initialize hourgit in a git repository",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "assign repository to a project by name or ID (creates if needed)"},
	},
	BoolFlags: []BoolFlag{
		{Name: "force", Usage: "overwrite existing post-checkout hook"},
		{Name: "merge", Usage: "append to existing post-checkout hook"},
		{Name: "yes", Usage: "skip confirmation prompt"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}

		projectName, _ := cmd.Flags().GetString("project")
		force, _ := cmd.Flags().GetBool("force")
		merge, _ := cmd.Flags().GetBool("merge")
		yes, _ := cmd.Flags().GetBool("yes")

		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		var confirm ConfirmFunc
		if yes {
			confirm = AlwaysYes()
		} else {
			confirm = NewConfirmFunc(cmd.InOrStdin(), cmd.OutOrStdout())
		}

		return runInit(cmd, dir, homeDir, projectName, force, merge, confirm)
	},
}.Build()

func runInit(cmd *cobra.Command, dir, homeDir, projectName string, force, merge bool, confirm ConfirmFunc) error {
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
			return fmt.Errorf("repository is already assigned to project '%s' (use 'project assign --force' to reassign)", cfg.Project)
		}

		// If project doesn't exist, prompt to create
		var entry *project.ProjectEntry
		if resolved == nil {
			prompt := fmt.Sprintf("Project '%s' does not exist. Create it?", projectName)
			confirmed, err := confirm(prompt)
			if err != nil {
				return err
			}
			if !confirmed {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text("project assignment skipped"))
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("hourgit initialized successfully"))
				return nil
			}

			entry, err = project.CreateProject(homeDir, projectName)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(entry.Name), Silent(entry.ID))))
		} else {
			entry = resolved
		}

		if err := project.AssignProject(homeDir, dir, entry); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository assigned to project '%s'", Primary(entry.Name))))
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("hourgit initialized successfully"))
	return nil
}
