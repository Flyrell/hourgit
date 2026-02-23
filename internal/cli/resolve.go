package cli

import (
	"fmt"

	"github.com/Flyrell/hour-git/internal/project"
)

// ResolveProjectContext finds the active project using the --project flag or
// the current repo's .git/.hourgit config.
func ResolveProjectContext(homeDir, repoDir, projectFlag string) (*project.ProjectEntry, error) {
	reg, err := project.ReadRegistry(homeDir)
	if err != nil {
		return nil, err
	}

	if projectFlag != "" {
		entry := project.ResolveProject(reg, projectFlag)
		if entry == nil {
			return nil, fmt.Errorf("project '%s' not found", projectFlag)
		}
		return entry, nil
	}

	if repoDir != "" {
		cfg, err := project.ReadRepoConfig(repoDir)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			entry := project.FindProjectByID(reg, cfg.ProjectID)
			if entry != nil {
				return entry, nil
			}
			entry = project.FindProject(reg, cfg.Project)
			if entry != nil {
				return entry, nil
			}
			return nil, fmt.Errorf("project '%s' from repo config not found in registry", cfg.Project)
		}
	}

	return nil, fmt.Errorf("no project found (use --project or run from inside an assigned repo)")
}
