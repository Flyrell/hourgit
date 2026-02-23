package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hour-git/internal/hashutil"
	"github.com/Flyrell/hour-git/internal/stringutil"
)

const hookMarker = "# Installed by hourgit"

// RepoConfig is the per-repo marker stored in .git/.hourgit.
type RepoConfig struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id,omitempty"`
}

// ProjectEntry represents a single project in the global registry.
type ProjectEntry struct {
	ID    string   `json:"id"`
	Name  string   `json:"name"`
	Slug  string   `json:"slug"`
	Repos []string `json:"repos"`
}

// ProjectRegistry holds all registered projects.
type ProjectRegistry struct {
	Projects []ProjectEntry `json:"projects"`
}

// HourgitDir returns the global hourgit config directory.
func HourgitDir(homeDir string) string {
	return filepath.Join(homeDir, ".hourgit")
}

// RegistryPath returns the path to the global projects.json.
func RegistryPath(homeDir string) string {
	return filepath.Join(HourgitDir(homeDir), "projects.json")
}

// LogDir returns the directory for a project's log entries.
func LogDir(homeDir, slug string) string {
	return filepath.Join(HourgitDir(homeDir), slug)
}

// ReadRegistry reads the global project registry.
// Returns an empty registry if the file does not exist.
func ReadRegistry(homeDir string) (*ProjectRegistry, error) {
	data, err := os.ReadFile(RegistryPath(homeDir))
	if errors.Is(err, os.ErrNotExist) {
		return &ProjectRegistry{}, nil
	}
	if err != nil {
		return nil, err
	}

	var reg ProjectRegistry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, err
	}
	return &reg, nil
}

// WriteRegistry writes the global project registry, creating the directory if needed.
func WriteRegistry(homeDir string, reg *ProjectRegistry) error {
	dir := HourgitDir(homeDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(RegistryPath(homeDir), data, 0644)
}

// FindProject looks up a project by name in the registry.
// Returns nil if not found.
func FindProject(reg *ProjectRegistry, name string) *ProjectEntry {
	for i := range reg.Projects {
		if reg.Projects[i].Name == name {
			return &reg.Projects[i]
		}
	}
	return nil
}

// FindProjectByID looks up a project by ID in the registry.
// Returns nil if not found.
func FindProjectByID(reg *ProjectRegistry, id string) *ProjectEntry {
	for i := range reg.Projects {
		if reg.Projects[i].ID == id {
			return &reg.Projects[i]
		}
	}
	return nil
}

// ResolveProject looks up a project by ID first, then by name.
// Returns nil if not found by either.
func ResolveProject(reg *ProjectRegistry, identifier string) *ProjectEntry {
	if entry := FindProjectByID(reg, identifier); entry != nil {
		return entry
	}
	return FindProject(reg, identifier)
}

// ReadRepoConfig reads the per-repo hourgit config from .git/.hourgit.
// Returns nil if the file does not exist.
func ReadRepoConfig(repoDir string) (*RepoConfig, error) {
	data, err := os.ReadFile(filepath.Join(repoDir, ".git", ".hourgit"))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg RepoConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// WriteRepoConfig writes the per-repo hourgit config to .git/.hourgit.
func WriteRepoConfig(repoDir string, cfg *RepoConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(repoDir, ".git", ".hourgit"), data, 0644)
}

// RemoveRepoFromProject removes repoDir from the project's repos list.
func RemoveRepoFromProject(entry *ProjectEntry, repoDir string) {
	repos := make([]string, 0, len(entry.Repos))
	for _, r := range entry.Repos {
		if r != repoDir {
			repos = append(repos, r)
		}
	}
	entry.Repos = repos
}

// CreateProject creates a new project in the registry.
// Returns an error if a project with the same name already exists.
func CreateProject(homeDir, name string) (*ProjectEntry, error) {
	reg, err := ReadRegistry(homeDir)
	if err != nil {
		return nil, err
	}

	if existing := FindProject(reg, name); existing != nil {
		return nil, fmt.Errorf("project '%s' already exists (%s)", name, existing.ID)
	}

	entry := ProjectEntry{
		ID:    hashutil.GenerateID(name),
		Name:  name,
		Slug:  stringutil.Slugify(name),
		Repos: []string{},
	}
	reg.Projects = append(reg.Projects, entry)

	if err := os.MkdirAll(LogDir(homeDir, entry.Slug), 0755); err != nil {
		return nil, err
	}

	if err := WriteRegistry(homeDir, reg); err != nil {
		return nil, err
	}

	return &entry, nil
}

// AssignProject assigns a repository to an existing project.
// It adds repoDir to the project's repos list (deduplicated) and writes the per-repo config.
func AssignProject(homeDir, repoDir string, entry *ProjectEntry) error {
	reg, err := ReadRegistry(homeDir)
	if err != nil {
		return err
	}

	regEntry := FindProjectByID(reg, entry.ID)
	if regEntry == nil {
		return fmt.Errorf("project '%s' not found in registry", entry.Name)
	}

	// Add repo if not already present
	found := false
	for _, r := range regEntry.Repos {
		if r == repoDir {
			found = true
			break
		}
	}
	if !found {
		regEntry.Repos = append(regEntry.Repos, repoDir)
	}

	if err := WriteRegistry(homeDir, reg); err != nil {
		return err
	}

	return WriteRepoConfig(repoDir, &RepoConfig{Project: regEntry.Name, ProjectID: regEntry.ID})
}

// RemoveProject removes a project from the registry by ID or name.
// Returns the removed entry so the caller can handle cleanup.
func RemoveProject(homeDir, identifier string) (*ProjectEntry, error) {
	reg, err := ReadRegistry(homeDir)
	if err != nil {
		return nil, err
	}

	idx := -1
	for i := range reg.Projects {
		if reg.Projects[i].ID == identifier || reg.Projects[i].Name == identifier {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil, fmt.Errorf("project '%s' not found", identifier)
	}

	removed := reg.Projects[idx]
	reg.Projects = append(reg.Projects[:idx], reg.Projects[idx+1:]...)

	if err := WriteRegistry(homeDir, reg); err != nil {
		return nil, err
	}

	return &removed, nil
}

// RemoveRepoConfig deletes the per-repo hourgit config (.git/.hourgit).
func RemoveRepoConfig(repoDir string) error {
	path := filepath.Join(repoDir, ".git", ".hourgit")
	err := os.Remove(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// RemoveHookFromRepo removes the hourgit section from the post-checkout hook.
// If the hook becomes empty after removal, it is deleted.
func RemoveHookFromRepo(repoDir string) error {
	hookPath := filepath.Join(repoDir, ".git", "hooks", "post-checkout")
	data, err := os.ReadFile(hookPath)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}

	content := string(data)
	markerIdx := strings.Index(content, hookMarker)
	if markerIdx == -1 {
		return nil
	}

	before := content[:markerIdx]
	// Trim trailing whitespace from before section
	before = strings.TrimRight(before, " \t\n")

	// If only a shebang line remains (or nothing), the hook is hourgit-only
	if before == "" || strings.TrimSpace(before) == "#!/bin/sh" {
		return os.Remove(hookPath)
	}

	// Write back the non-hourgit portion
	return os.WriteFile(hookPath, []byte(before+"\n"), 0755)
}

