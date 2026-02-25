package cli

import (
	"fmt"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/spf13/cobra"
)

var checkoutCmd = LeafCommand{
	Use:   "checkout",
	Short: "Record a branch checkout (used by the post-checkout hook)",
	StrFlags: []StringFlag{
		{Name: "prev", Usage: "previous branch name"},
		{Name: "next", Usage: "next branch name"},
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		prevFlag, _ := cmd.Flags().GetString("prev")
		nextFlag, _ := cmd.Flags().GetString("next")

		return runCheckout(cmd, homeDir, repoDir, projectFlag, prevFlag, nextFlag, time.Now)
	},
}.Build()

func runCheckout(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag, prev, next string,
	nowFn func() time.Time,
) error {
	if prev == "" {
		return fmt.Errorf("--prev is required")
	}
	if next == "" {
		return fmt.Errorf("--next is required")
	}
	if prev == next {
		return nil // silent no-op, known benign case from hook
	}

	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	e := entry.CheckoutEntry{
		ID:        hashutil.GenerateID("checkout"),
		Timestamp: nowFn().UTC(),
		Previous:  prev,
		Next:      next,
	}

	if err := entry.WriteCheckoutEntry(homeDir, proj.Slug, e); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "checkout %s → %s for project '%s' (%s)\n",
		Primary(prev),
		Primary(next),
		Primary(proj.Name),
		Silent(e.ID),
	)

	return nil
}
