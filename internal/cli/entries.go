package cli

import (
	"github.com/Flyrell/hourgit/internal/entry"
)

// ProjectEntries holds all entry types for a project.
type ProjectEntries struct {
	Checkouts      []entry.CheckoutEntry
	Logs           []entry.Entry
	Commits        []entry.CommitEntry
	ActivityStops  []entry.ActivityStopEntry
	ActivityStarts []entry.ActivityStartEntry
}

// LoadProjectEntries reads all 5 entry types for a project in one call.
func LoadProjectEntries(homeDir, slug string) (ProjectEntries, error) {
	checkouts, err := entry.ReadAllCheckoutEntries(homeDir, slug)
	if err != nil {
		return ProjectEntries{}, err
	}

	logs, err := entry.ReadAllEntries(homeDir, slug)
	if err != nil {
		return ProjectEntries{}, err
	}

	commits, err := entry.ReadAllCommitEntries(homeDir, slug)
	if err != nil {
		return ProjectEntries{}, err
	}

	activityStops, err := entry.ReadAllActivityStopEntries(homeDir, slug)
	if err != nil {
		return ProjectEntries{}, err
	}

	activityStarts, err := entry.ReadAllActivityStartEntries(homeDir, slug)
	if err != nil {
		return ProjectEntries{}, err
	}

	return ProjectEntries{
		Checkouts:      checkouts,
		Logs:           logs,
		Commits:        commits,
		ActivityStops:  activityStops,
		ActivityStarts: activityStarts,
	}, nil
}
