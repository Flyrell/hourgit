package cli

import (
	"fmt"

	"github.com/Flyrell/hour-git/internal/project"
)

// ResolveProjectContext finds the active project using the --project flag or
// the current repo's .git/.hourgit config.
func ResolveProjectContext(homeDir, repoDir, projectFlag string) (*project.ProjectEntry, error) {
	cfg, err := project.ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	if projectFlag != "" {
		entry := project.ResolveProject(cfg, projectFlag)
		if entry == nil {
			return nil, fmt.Errorf("project '%s' not found", projectFlag)
		}
		return entry, nil
	}

	if repoDir != "" {
		repoCfg, err := project.ReadRepoConfig(repoDir)
		if err != nil {
			return nil, err
		}
		if repoCfg != nil {
			entry := project.FindProjectByID(cfg, repoCfg.ProjectID)
			if entry != nil {
				return entry, nil
			}
			entry = project.FindProject(cfg, repoCfg.Project)
			if entry != nil {
				return entry, nil
			}
			return nil, fmt.Errorf("project '%s' from repo config not found in registry", repoCfg.Project)
		}
	}

	return nil, fmt.Errorf("no project found (use --project or run from inside an assigned repo)")
}
