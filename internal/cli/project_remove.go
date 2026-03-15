package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var projectRemoveCmd = LeafCommand{
	Use:   "remove [PROJECT]",
	Short: "Remove a project and clean up its repository assignments",
	Args:  cobra.MaximumNArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Shorthand: "y", Usage: "skip confirmation prompt"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Shorthand: "p", Usage: "project name or ID"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		repoDir, _ := os.Getwd()

		projectFlag, _ := cmd.Flags().GetString("project")
		yes, _ := cmd.Flags().GetBool("yes")
		confirm := ResolveConfirmFunc(yes)

		// Resolve project identifier: positional arg > --project flag > repo config
		var identifier string
		if len(args) > 0 {
			identifier = args[0]
		} else if projectFlag != "" {
			identifier = projectFlag
		}

		if identifier == "" {
			// Fall back to repo config
			entry, err := resolveProjectFromRepo(homeDir, repoDir)
			if err != nil {
				return err
			}
			identifier = entry.Name
		}

		return runProjectRemove(cmd, homeDir, identifier, confirm)
	},
}.Build()

func runProjectRemove(cmd *cobra.Command, homeDir, identifier string, confirm ConfirmFunc) error {
	// Look up the project first to check repos
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	entry := project.ResolveProject(cfg, identifier)
	if entry == nil {
		return fmt.Errorf("project '%s' not found", identifier)
	}

	// If project has repos, prompt for confirmation
	if len(entry.Repos) > 0 {
		repoList := strings.Join(entry.Repos, "\n  ")
		prompt := fmt.Sprintf("Project '%s' is assigned to %d repo(s):\n  %s\nRemove project and clean up assignments?",
			entry.Name, len(entry.Repos), repoList)

		confirmed, err := confirm(prompt)
		if err != nil {
			return err
		}
		if !confirmed {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "cancelled")
			return nil
		}
	}

	// Best-effort cleanup: remove repo configs and hooks. Errors are intentionally
	// ignored because the repos may have been moved/deleted since assignment, and
	// failing to clean up a single repo should not block project removal.
	for _, repoDir := range entry.Repos {
		_ = project.RemoveRepoConfig(repoDir)
		_ = project.RemoveHookFromRepo(repoDir)
	}

	// Best-effort cleanup: delete the project's time entry directory
	_ = os.RemoveAll(project.LogDir(homeDir, entry.Slug))

	// Remove project from registry
	_, err = project.RemoveProject(homeDir, identifier)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' removed", Primary(entry.Name))))
	return nil
}

// resolveProjectFromRepo resolves a project from the current repo's config.
func resolveProjectFromRepo(homeDir, repoDir string) (*project.ProjectEntry, error) {
	if repoDir == "" {
		return nil, fmt.Errorf("no project specified (use positional arg, --project flag, or run from an assigned repo)")
	}

	repoCfg, err := project.ReadRepoConfig(repoDir)
	if err != nil {
		return nil, err
	}
	if repoCfg == nil {
		return nil, fmt.Errorf("no project specified (use positional arg, --project flag, or run from an assigned repo)")
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	entry := project.FindProjectByID(cfg, repoCfg.ProjectID)
	if entry != nil {
		return entry, nil
	}
	entry = project.FindProject(cfg, repoCfg.Project)
	if entry != nil {
		return entry, nil
	}

	return nil, fmt.Errorf("project '%s' from repo config not found in registry", repoCfg.Project)
}
