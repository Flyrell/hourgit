package entry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Flyrell/hourgit/internal/project"
)

// EntryPath returns the filesystem path for a single entry file.
func EntryPath(homeDir, slug, id string) string {
	return filepath.Join(project.LogDir(homeDir, slug), id)
}

// WriteEntry writes a single entry file to the project's log directory.
func WriteEntry(homeDir, slug string, e Entry) error {
	dir := project.LogDir(homeDir, slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(e, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(EntryPath(homeDir, slug, e.ID), data, 0644)
}

// ReadEntry reads a single entry by hash from a project's log directory.
func ReadEntry(homeDir, slug, id string) (Entry, error) {
	data, err := os.ReadFile(EntryPath(homeDir, slug, id))
	if err != nil {
		return Entry{}, fmt.Errorf("entry '%s' not found", id)
	}

	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return Entry{}, err
	}
	return e, nil
}

// ReadAllEntries reads all entry files from a project's log directory.
func ReadAllEntries(homeDir, slug string) ([]Entry, error) {
	dir := project.LogDir(homeDir, slug)
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			continue // skip corrupt files
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// DeleteEntry removes an entry file by hash.
func DeleteEntry(homeDir, slug, id string) error {
	err := os.Remove(EntryPath(homeDir, slug, id))
	if os.IsNotExist(err) {
		return fmt.Errorf("entry '%s' not found", id)
	}
	return err
}
