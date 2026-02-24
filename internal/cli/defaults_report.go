package cli

import (
	"os"
	"time"

	"github.com/Flyrell/hourgit/internal/project"
	"github.com/spf13/cobra"
)

var defaultsReportCmd = LeafCommand{
	Use:   "report",
	Short: "Show expanded default working hours for a given month",
	StrFlags: []StringFlag{
		{Name: "month", Usage: "month number 1-12 (default: current)"},
		{Name: "year", Usage: "year (default: current)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		monthFlag, _ := cmd.Flags().GetString("month")
		yearFlag, _ := cmd.Flags().GetString("year")

		return runDefaultsReport(cmd, homeDir, monthFlag, yearFlag, time.Now())
	},
}.Build()

func runDefaultsReport(cmd *cobra.Command, homeDir, monthFlag, yearFlag string, now time.Time) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	defaults := project.GetDefaults(cfg)

	return printScheduleReport(cmd, defaults, "Default working hours", monthFlag, yearFlag, now)
}
