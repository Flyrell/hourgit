package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var removeCmd = LeafCommand{
	Use:   "remove <hash>",
	Short: "Remove a log or checkout entry",
	Args:  cobra.ExactArgs(1),
	BoolFlags: []BoolFlag{
		{Name: "yes", Usage: "skip confirmation prompt"},
	},
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		repoDir, _ := os.Getwd()
		projectFlag, _ := cmd.Flags().GetString("project")
		yesFlag, _ := cmd.Flags().GetBool("yes")

		var confirm ConfirmFunc
		if yesFlag {
			confirm = AlwaysYes()
		} else {
			confirm = NewConfirmFunc()
		}

		return runRemove(cmd, homeDir, repoDir, projectFlag, args[0], confirm)
	},
}.Build()

func runRemove(cmd *cobra.Command, homeDir, repoDir, projectFlag, hash string, confirm ConfirmFunc) error {
	slug, entryType, detail, err := locateAnyEntry(homeDir, repoDir, projectFlag, hash)
	if err != nil {
		return err
	}

	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintf(w, "  type:   %s\n", Primary(entryType))
	_, _ = fmt.Fprintf(w, "  detail: %s\n", Primary(detail))

	if confirm != nil {
		ok, err := confirm("Remove this entry?")
		if err != nil {
			return err
		}
		if !ok {
			_, _ = fmt.Fprintln(w, "cancelled")
			return nil
		}
	}

	if err := entry.DeleteEntry(homeDir, slug, hash); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(w, "removed entry %s\n", Silent(hash))
	return nil
}

// locateAnyEntry finds any entry (log or checkout) by hash, trying project flag,
// repo context, then scanning all projects.
func locateAnyEntry(homeDir, repoDir, projectFlag, hash string) (slug, entryType, detail string, err error) {
	// Try project flag first
	if projectFlag != "" {
		cfg, err := project.ReadConfig(homeDir)
		if err != nil {
			return "", "", "", err
		}
		proj := project.ResolveProject(cfg, projectFlag)
		if proj == nil {
			return "", "", "", fmt.Errorf("project '%s' not found", projectFlag)
		}
		return locateAnyEntryInProject(homeDir, proj.Slug, hash)
	}

	// Try repo context
	if repoDir != "" {
		proj, err := ResolveProjectContext(homeDir, repoDir, "")
		if err == nil {
			s, t, d, err := locateAnyEntryInProject(homeDir, proj.Slug, hash)
			if err == nil {
				return s, t, d, nil
			}
		}
	}

	// Scan all projects
	found, err := entry.FindAnyEntryAcrossProjects(homeDir, hash)
	if err != nil {
		return "", "", "", err
	}
	return found.Slug, found.Type, found.Detail, nil
}

// locateAnyEntryInProject tries to find a log or checkout entry in a specific project.
func locateAnyEntryInProject(homeDir, slug, hash string) (string, string, string, error) {
	// Try as log entry
	e, err := entry.ReadEntry(homeDir, slug, hash)
	if err == nil {
		detail := fmt.Sprintf("%s — %s", entry.FormatMinutes(e.Minutes), e.Message)
		if e.Task != "" {
			detail = fmt.Sprintf("[%s] %s", e.Task, detail)
		}
		return slug, entry.TypeLog, detail, nil
	}

	// Try as checkout entry
	ce, err := entry.ReadCheckoutEntry(homeDir, slug, hash)
	if err == nil {
		detail := fmt.Sprintf("%s → %s at %s",
			ce.Previous, ce.Next, ce.Timestamp.Format("2006-01-02 15:04"))
		return slug, entry.TypeCheckout, detail, nil
	}

	return "", "", "", fmt.Errorf("entry '%s' not found", hash)
}
