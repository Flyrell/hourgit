package cli

import (
	"fmt"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var configReportCmd = LeafCommand{
	Use:   "report",
	Short: "Show expanded working hours for a given month",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")
		monthFlag, _ := cmd.Flags().GetString("month")
		yearFlag, _ := cmd.Flags().GetString("year")

		return runConfigReport(cmd, homeDir, repoDir, projectFlag, monthFlag, yearFlag, time.Now())
	},
}.Build()

func runConfigReport(cmd *cobra.Command, homeDir, repoDir, projectFlag, monthFlag, yearFlag string, now time.Time) error {
	entry, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	entries := project.GetSchedules(cfg, entry.ID)
	label := fmt.Sprintf("Working hours for '%s'", Primary(entry.Name))

	return printScheduleReport(cmd, entries, label, monthFlag, yearFlag, now)
}
