package entry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"github.com/Flyrell/hourgit/internal/project"
)

var validIDPattern = regexp.MustCompile(`^[0-9a-f]{7}$`)

// validateID checks that an entry ID matches the expected 7-char hex format.
func validateID(id string) error {
	if !validIDPattern.MatchString(id) {
		return fmt.Errorf("invalid entry ID %q", id)
	}
	return nil
}

// EntryPath returns the filesystem path for a single entry file.
func EntryPath(homeDir, slug, id string) (string, error) {
	if err := validateID(id); err != nil {
		return "", err
	}
	return filepath.Join(project.LogDir(homeDir, slug), id), nil
}

// writeTypedEntry marshals data with the given type and writes it to the project's log directory.
func writeTypedEntry(homeDir, slug, id string, data any) error {
	dir := project.LogDir(homeDir, slug)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	path, err := EntryPath(homeDir, slug, id)
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// fileData holds the raw bytes and filename from a directory scan.
type fileData struct {
	name string
	data []byte
}

// readAllFiles reads all non-directory files from a project's log directory.
func readAllFiles(homeDir, slug string) ([]fileData, error) {
	dir := project.LogDir(homeDir, slug)
	files, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var result []fileData
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, f.Name()))
		if err != nil {
			return nil, err
		}
		result = append(result, fileData{name: f.Name(), data: data})
	}
	return result, nil
}

// matchesType checks if JSON data has a "type" field matching expectedType.
// For log entries, missing or empty type also matches (legacy compatibility).
func matchesType(data []byte, expectedType string) bool {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return false
	}
	t, ok := raw["type"]
	if !ok {
		return expectedType == TypeLog
	}
	var typ string
	if err := json.Unmarshal(t, &typ); err != nil {
		return false
	}
	if typ == "" {
		return expectedType == TypeLog
	}
	return typ == expectedType
}

// WriteEntry writes a single entry file to the project's log directory.
func WriteEntry(homeDir, slug string, e Entry) error {
	e.Type = TypeLog
	return writeTypedEntry(homeDir, slug, e.ID, e)
}

// ReadEntry reads a single entry by hash from a project's log directory.
func ReadEntry(homeDir, slug, id string) (Entry, error) {
	path, err := EntryPath(homeDir, slug, id)
	if err != nil {
		return Entry{}, err
	}
	data, err := os.ReadFile(path)
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
	files, err := readAllFiles(homeDir, slug)
	if err != nil {
		return nil, err
	}

	var entries []Entry
	for _, f := range files {
		if !matchesType(f.data, TypeLog) {
			continue
		}
		var e Entry
		if err := json.Unmarshal(f.data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// IsCheckoutEntry checks if the file at the given path exists and is a checkout entry.
func IsCheckoutEntry(homeDir, slug, id string) bool {
	path, err := EntryPath(homeDir, slug, id)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return matchesType(data, TypeCheckout)
}

// DeleteEntry removes an entry file by hash.
func DeleteEntry(homeDir, slug, id string) error {
	path, err := EntryPath(homeDir, slug, id)
	if err != nil {
		return err
	}
	err = os.Remove(path)
	if os.IsNotExist(err) {
		return fmt.Errorf("entry '%s' not found", id)
	}
	return err
}

// WriteCheckoutEntry writes a single checkout entry file to the project's log directory.
func WriteCheckoutEntry(homeDir, slug string, e CheckoutEntry) error {
	e.Type = TypeCheckout
	return writeTypedEntry(homeDir, slug, e.ID, e)
}

// ReadCheckoutEntry reads a single checkout entry by ID.
func ReadCheckoutEntry(homeDir, slug, id string) (CheckoutEntry, error) {
	path, err := EntryPath(homeDir, slug, id)
	if err != nil {
		return CheckoutEntry{}, err
	}
	data, err := os.ReadFile(path)
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
	return writeTypedEntry(homeDir, slug, e.ID, e)
}

// ReadAllSubmitEntries reads all submit marker entries from a project's log directory.
func ReadAllSubmitEntries(homeDir, slug string) ([]SubmitEntry, error) {
	files, err := readAllFiles(homeDir, slug)
	if err != nil {
		return nil, err
	}

	var entries []SubmitEntry
	for _, f := range files {
		if !matchesType(f.data, TypeSubmit) {
			continue
		}
		var e SubmitEntry
		if err := json.Unmarshal(f.data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// ReadAllCheckoutEntries reads all checkout entries from a project's log directory.
func ReadAllCheckoutEntries(homeDir, slug string) ([]CheckoutEntry, error) {
	files, err := readAllFiles(homeDir, slug)
	if err != nil {
		return nil, err
	}

	var entries []CheckoutEntry
	for _, f := range files {
		if !matchesType(f.data, TypeCheckout) {
			continue
		}
		var e CheckoutEntry
		if err := json.Unmarshal(f.data, &e); err != nil {
			continue
		}
		entries = append(entries, e)
	}
	return entries, nil
}
