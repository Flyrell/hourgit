package cli

import (
	"fmt"
	"os"

	"github.com/Flyrell/hour-git/internal/project"
	"github.com/spf13/cobra"
)

var projectListCmd = LeafCommand{
	Use:   "list",
	Short: "List all projects and their repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		return runProjectList(cmd, homeDir)
	},
}.Build()

func runProjectList(cmd *cobra.Command, homeDir string) error {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return err
	}

	if len(cfg.Projects) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), Silent("No projects found."))
		return nil
	}

	for i, p := range cfg.Projects {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s  %s\n", Silent(p.ID), Primary(p.Name))
		if len(p.Repos) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), Silent("└── (no repositories assigned)"))
		} else {
			for j, r := range p.Repos {
				if j < len(p.Repos)-1 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("├── %s", r)))
				} else {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n", Text(fmt.Sprintf("└── %s", r)))
				}
			}
		}
		if i < len(cfg.Projects)-1 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	return nil
}
