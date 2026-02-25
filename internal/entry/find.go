package entry

import (
	"errors"
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
		// Directory not existing means no projects exist yet — treat as not found.
		// Other errors (permissions, IO) are propagated.
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("entry '%s' not found", id)
		}
		return nil, fmt.Errorf("reading hourgit directory: %w", err)
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		slug := d.Name()
		e, err := ReadEntry(homeDir, slug, id)
		if err != nil {
			// ReadEntry returns "not found" or type-mismatch errors — skip to next project.
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

// FoundAnyEntry pairs an entry ID, type, slug, and human-readable detail.
type FoundAnyEntry struct {
	ID     string
	Type   string // "log" or "checkout"
	Slug   string
	Detail string // human-readable summary
}

// FindAnyEntryAcrossProjects scans all project directories under ~/.hourgit/
// looking for a log or checkout entry with the given ID. Returns the first match.
func FindAnyEntryAcrossProjects(homeDir, id string) (*FoundAnyEntry, error) {
	hourgitDir := filepath.Join(homeDir, ".hourgit")
	dirs, err := os.ReadDir(hourgitDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("entry '%s' not found", id)
		}
		return nil, fmt.Errorf("reading hourgit directory: %w", err)
	}

	for _, d := range dirs {
		if !d.IsDir() {
			continue
		}
		slug := d.Name()

		// Try as log entry
		e, err := ReadEntry(homeDir, slug, id)
		if err == nil {
			detail := fmt.Sprintf("%s — %s", FormatMinutes(e.Minutes), e.Message)
			if e.Task != "" {
				detail = fmt.Sprintf("[%s] %s", e.Task, detail)
			}
			return &FoundAnyEntry{ID: e.ID, Type: TypeLog, Slug: slug, Detail: detail}, nil
		}

		// Try as checkout entry
		ce, err := ReadCheckoutEntry(homeDir, slug, id)
		if err == nil {
			detail := fmt.Sprintf("%s → %s at %s",
				ce.Previous, ce.Next, ce.Timestamp.Format("2006-01-02 15:04"))
			return &FoundAnyEntry{ID: ce.ID, Type: TypeCheckout, Slug: slug, Detail: detail}, nil
		}
	}

	return nil, fmt.Errorf("entry '%s' not found", id)
}
