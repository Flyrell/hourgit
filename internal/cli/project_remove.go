package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var projectRemoveCmd = LeafCommand{
	Use:   "remove PROJECT",
	Short: "Remove a project and clean up its repository assignments",
	Args:  cobra.ExactArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompt"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		yes, _ := cmd.Flags().GetBool("yes")
		confirm := ResolveConfirmFunc(yes)

		return runProjectRemove(cmd, homeDir, args[0], confirm)
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
			return fmt.Errorf("aborted")
		}
	}

	// Best-effort cleanup: remove repo configs and hooks. Errors are intentionally
	// ignored because the repos may have been moved/deleted since assignment, and
	// failing to clean up a single repo should not block project removal.
	for _, repoDir := range entry.Repos {
		_ = project.RemoveRepoConfig(repoDir)
		_ = project.RemoveHookFromRepo(repoDir)
	}

	// Remove project from registry
	_, err = project.RemoveProject(homeDir, identifier)
	if err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("project '%s' removed", Primary(entry.Name))))
	return nil
}
