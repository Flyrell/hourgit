package cli

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

// historyItem is a unified representation of a log or checkout entry for display.
type historyItem struct {
	ID        string
	Timestamp time.Time
	Type      string
	Project   string
	Detail    string
}

var historyCmd = LeafCommand{
	Use:   "history",
	Short: "Show a chronological feed of all recorded activity",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "filter by project name or ID"},
		{Name: "limit", Usage: "maximum number of entries to show (0 = all)", Default: "50"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		limitStr, _ := cmd.Flags().GetString("limit")

		limit, err := strconv.Atoi(limitStr)
		if err != nil {
			return fmt.Errorf("invalid --limit value %q: expected a number", limitStr)
		}
		if limit < 0 {
			return fmt.Errorf("--limit must be 0 or positive")
		}

		return runHistory(cmd, homeDir, projectFlag, limit)
	},
}.Build()

func runHistory(cmd *cobra.Command, homeDir, projectFlag string, limit int) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	projects := cfg.Projects
	if projectFlag != "" {
		entry := project.ResolveProject(cfg, projectFlag)
		if entry == nil {
			return fmt.Errorf("project '%s' not found", projectFlag)
		}
		projects = []project.ProjectEntry{*entry}
	}

	var items []historyItem

	for _, proj := range projects {
		logs, err := entry.ReadAllEntries(homeDir, proj.Slug)
		if err != nil {
			return err
		}
		for _, e := range logs {
			detail := entry.FormatMinutes(e.Minutes)
			if e.Task != "" {
				detail += "  [" + e.Task + "]"
			}
			if e.Message != "" {
				if e.Task != "" {
					detail += " " + e.Message
				} else {
					detail += "  " + e.Message
				}
			}
			items = append(items, historyItem{
				ID:        e.ID,
				Timestamp: e.CreatedAt,
				Type:      "log",
				Project:   proj.Name,
				Detail:    detail,
			})
		}

		checkouts, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
		if err != nil {
			return err
		}
		for _, e := range checkouts {
			items = append(items, historyItem{
				ID:        e.ID,
				Timestamp: e.Timestamp,
				Type:      "checkout",
				Project:   proj.Name,
				Detail:    e.Previous + " → " + e.Next,
			})
		}
	}

	if len(items) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no entries found")
		return nil
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Timestamp.After(items[j].Timestamp)
	})

	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}

	w := cmd.OutOrStdout()
	for _, item := range items {
		_, _ = fmt.Fprintf(w, "%s  %s  %s  %s  %s\n",
			Silent(item.ID),
			Text(item.Timestamp.Format("2006-01-02 15:04:05")),
			Info(item.Type),
			Primary(item.Project),
			Text(item.Detail),
		)
	}

	return nil
}
