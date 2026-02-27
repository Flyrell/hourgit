package cli

import (
	"fmt"

	"github.com/Flyrell/hourgit/internal/project"
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

	// Best-effort cache update
	homeDir, _ := deps.homeDir()
	var cfg *project.Config
	if homeDir != "" {
		cfg, _ = deps.readConfig(homeDir)
		if cfg != nil {
			now := deps.now()
			cfg.LastUpdateCheck = &now
			cfg.LatestVersion = latest
			_ = deps.writeConfig(homeDir, cfg)
		}
	}

	if compareVersions(appVersion, latest) >= 0 {
		_, _ = fmt.Fprintf(w, "%s\n", Text(fmt.Sprintf("hourgit is up to date (%s)", appVersion)))
		return nil
	}

	// Newer version available
	if homeDir != "" && cfg != nil {
		promptUpdate(cmd, deps, homeDir, cfg)
	} else {
		_, _ = fmt.Fprintf(w, "\n%s\n", Warning(fmt.Sprintf(
			"A new version of hourgit is available: %s → %s", appVersion, Primary(latest))))
		install, confirmErr := deps.confirm("Install update now?")
		if confirmErr != nil || !install {
			return nil
		}
		_, _ = fmt.Fprintf(w, "%s\n", Text("Installing update..."))
		if installErr := deps.runInstall(); installErr != nil {
			_, _ = fmt.Fprintf(w, "%s\n\n", Error(fmt.Sprintf("Update failed: %s", installErr)))
			return nil
		}
		_, _ = fmt.Fprintf(w, "%s\n\n", Text("Update installed. Restarting..."))
		if restartErr := deps.restartSelf(); restartErr != nil {
			_, _ = fmt.Fprintf(w, "%s\n\n", Error(fmt.Sprintf("Restart failed: %s", restartErr)))
		}
	}

	return nil
}
