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
	e.Type = TypeLog

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
	if e.Type != "" && e.Type != TypeLog {
		return Entry{}, fmt.Errorf("entry '%s' not found", id)
	}
	return e, nil
}

// ReadAllEntries reads all log entry files from a project's log directory.
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

		// Skip files that aren't valid JSON or aren't log entries.
		// Corrupted or partial files shouldn't block reading valid entries.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		if t, ok := raw["type"]; ok {
			var typ string
			if err := json.Unmarshal(t, &typ); err == nil && typ != "" && typ != TypeLog {
				continue
			}
		}

		var e Entry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// IsCheckoutEntry checks if the file at the given path exists and is a checkout entry.
func IsCheckoutEntry(homeDir, slug, id string) bool {
	data, err := os.ReadFile(EntryPath(homeDir, slug, id))
	if err != nil {
		return false
	}

	var raw struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	return raw.Type == TypeCheckout
}

// DeleteEntry removes an entry file by hash.
func DeleteEntry(homeDir, slug, id string) error {
	err := os.Remove(EntryPath(homeDir, slug, id))
	if os.IsNotExist(err) {
		return fmt.Errorf("entry '%s' not found", id)
	}
	return err
}

// WriteCheckoutEntry writes a single checkout entry file to the project's log directory.
func WriteCheckoutEntry(homeDir, slug string, e CheckoutEntry) error {
	e.Type = TypeCheckout

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

// ReadCheckoutEntry reads a single checkout entry by ID.
func ReadCheckoutEntry(homeDir, slug, id string) (CheckoutEntry, error) {
	data, err := os.ReadFile(EntryPath(homeDir, slug, id))
	if err != nil {
		return CheckoutEntry{}, fmt.Errorf("checkout entry '%s' not found", id)
	}

	var e CheckoutEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return CheckoutEntry{}, err
	}
	if e.Type != TypeCheckout {
		return CheckoutEntry{}, fmt.Errorf("checkout entry '%s' not found", id)
	}
	return e, nil
}

// WriteSubmitEntry writes a submit marker entry to the project's log directory.
func WriteSubmitEntry(homeDir, slug string, e SubmitEntry) error {
	e.Type = TypeSubmit

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

// ReadAllSubmitEntries reads all submit marker entries from a project's log directory.
func ReadAllSubmitEntries(homeDir, slug string) ([]SubmitEntry, error) {
	dir := project.LogDir(homeDir, slug)
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []SubmitEntry
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}

		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		t, ok := raw["type"]
		if !ok {
			continue
		}
		var typ string
		if err := json.Unmarshal(t, &typ); err != nil || typ != TypeSubmit {
			continue
		}

		var e SubmitEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ReadAllCheckoutEntries reads all checkout entries from a project's log directory.
func ReadAllCheckoutEntries(homeDir, slug string) ([]CheckoutEntry, error) {
	dir := project.LogDir(homeDir, slug)
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var entries []CheckoutEntry
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}

		// Skip files that aren't valid JSON or aren't checkout entries.
		// Corrupted or partial files shouldn't block reading valid entries.
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			continue
		}
		t, ok := raw["type"]
		if !ok {
			continue
		}
		var typ string
		if err := json.Unmarshal(t, &typ); err != nil || typ != TypeCheckout {
			continue
		}

		var e CheckoutEntry
		if err := json.Unmarshal(data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}
