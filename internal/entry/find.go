package entry

import (
	"fmt"
	"os"
	"path/filepath"
)

// FoundEntry pairs an entry with the project slug it was found in.
type FoundEntry struct {
	Entry Entry
	Slug  string
}

// FindEntryAcrossProjects scans all project directories under ~/.hourgit/
// looking for a log entry with the given ID. Returns the first match.
func FindEntryAcrossProjects(homeDir, id string) (*FoundEntry, error) {
	hourgitDir := filepath.Join(homeDir, ".hourgit")
	dirs, err := os.ReadDir(hourgitDir)
	if err != nil {
		return nil, fmt.Errorf("entry '%s' not found", id)
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		slug := d.Name()
		e, err := ReadEntry(homeDir, slug, id)
		if err != nil {
			continue
		}
		return &FoundEntry{Entry: e, Slug: slug}, nil
	}

	// Check if it exists as a checkout entry
	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		if IsCheckoutEntry(homeDir, d.Name(), id) {
			return nil, fmt.Errorf("entry '%s' is a checkout entry and cannot be edited", id)
		}
	}

	return nil, fmt.Errorf("entry '%s' not found", id)
}
