package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

func hookScript(binPath, version string) string {
	return fmt.Sprintf(`#!/bin/sh
%s (version: %s)

# Only act on branch checkouts (flag=1), skip file checkouts (flag=0)
[ "$3" = "0" ] && exit 0

# Skip if old and new HEAD are the same SHA (e.g. pull, fetch)
[ "$1" = "$2" ] && exit 0

%s sync 2>/dev/null || true
`, project.HookMarker, version, binPath)
}

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

		binPath, err := os.Executable()
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}
		binPath, err = filepath.EvalSymlinks(binPath)
		if err != nil {
			return fmt.Errorf("could not resolve binary path: %w", err)
		}

		confirm := ResolveConfirmFunc(yes)
		selectFn := ResolveSelectFunc(yes)

		return runInit(cmd, dir, homeDir, projectName, force, merge, binPath, confirm, selectFn)
	},
}.Build()

func runInit(cmd *cobra.Command, dir, homeDir, projectName string, force, merge bool, binPath string, confirm ConfirmFunc, selectFn SelectFunc) error {
	gitDir := filepath.Join(dir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository")
	}

	hooksDir := filepath.Join(gitDir, "hooks")
	hookPath := filepath.Join(hooksDir, "post-checkout")
	hook := hookScript(binPath, appVersion)

	if existing, err := os.ReadFile(hookPath); err == nil {
		content := string(existing)

		if strings.Contains(content, project.HookMarker) {
			return fmt.Errorf("hourgit is already initialized")
		}

		if !force && !merge {
			return fmt.Errorf("post-checkout hook already exists (use --force to overwrite or --merge to append)")
		}

		if merge {
			merged := content + "\n" + hook
			if err := os.WriteFile(hookPath, []byte(merged), 0755); err != nil {
				return err
			}
		} else {
			if err := os.WriteFile(hookPath, []byte(hook), 0755); err != nil {
				return err
			}
		}
	} else {
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return err
		}
		if err := os.WriteFile(hookPath, []byte(hook), 0755); err != nil {
			return err
		}
	}

	if projectName != "" {
		// Check if repo already has a different project
		cfg, err := project.ReadRepoConfig(dir)
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text("project assignment skipped"))
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("hourgit initialized successfully"))
			return nil
		}

		if cfg != nil && cfg.Project != "" && cfg.Project != result.Entry.Name {
			return fmt.Errorf("repository is already assigned to project '%s' (use 'project assign --force' to reassign)", cfg.Project)
		}

		if result.Created {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' created (%s)", Primary(result.Entry.Name), Silent(result.Entry.ID))))
		}

		if err := project.AssignProject(homeDir, dir, result.Entry); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("repository assigned to project '%s'", Primary(result.Entry.Name))))
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("hourgit initialized successfully"))

	// Offer to install shell completions
	shell := detectShell()
	if shell != "" && !isCompletionInstalled(shell, homeDir) {
		choice, err := selectFn("Shell completions", []string{
			fmt.Sprintf("Install automatically (for %s)", shell),
			"Skip (run 'hourgit completion install' later)",
		})
		if err != nil {
			return err
		}
		if choice == 0 {
			if err := installCompletion(shell, homeDir); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("shell completions installed for %s", Primary(shell))))
		}
	}

	return nil
}
