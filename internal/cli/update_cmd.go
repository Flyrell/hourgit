package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var updateCmd = LeafCommand{
	Use:   "update",
	Short: "Check for and install updates",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate(cmd, defaultUpdateDeps())
	},
}.Build()

func runUpdate(cmd *cobra.Command, deps updateDeps) error {
	w := cmd.OutOrStdout()

	if appVersion == "dev" {
		_, _ = fmt.Fprintf(w, "%s\n", Text("Cannot update a dev build"))
		return nil
	}

	_, _ = fmt.Fprintf(w, "%s\n", Text("Checking for updates..."))

	latest, err := deps.fetchVersion()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Update cache so auto-check doesn't re-fetch unnecessarily
	homeDir, homeErr := deps.homeDir()
	if homeErr == nil {
		cfg, cfgErr := deps.readConfig(homeDir)
		if cfgErr == nil {
			now := deps.now()
			cfg.LastUpdateCheck = &now
			cfg.LatestVersion = latest
			_ = deps.writeConfig(homeDir, cfg)

			// If newer version available, reuse promptUpdate flow
			if compareVersions(appVersion, latest) < 0 {
				promptUpdate(cmd, deps, homeDir, cfg)
				return nil
			}
		}
	}

	if compareVersions(appVersion, latest) >= 0 {
		_, _ = fmt.Fprintf(w, "%s\n", Text(fmt.Sprintf("hourgit is up to date (%s)", appVersion)))
		return nil
	}

	return nil
}
