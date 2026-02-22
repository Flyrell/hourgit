package project

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/Flyrell/hour-git/internal/stringutil"
)

// RepoConfig is the per-repo marker stored in .git/.hourgit.
type RepoConfig struct {
	Project string `json:"project"`
}

// ProjectEntry represents a single project in the global registry.
type ProjectEntry struct {
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

// RegisterProject ensures a project exists in the registry, adds the repo to it,
// creates the log directory, and writes the per-repo config.
// Returns the project entry and whether it was newly created.
func RegisterProject(homeDir, repoDir, projectName string) (*ProjectEntry, bool, error) {
	reg, err := ReadRegistry(homeDir)
	if err != nil {
		return nil, false, err
	}

	entry := FindProject(reg, projectName)
	created := entry == nil

	if created {
		entry = &ProjectEntry{
			Name:  projectName,
			Slug:  stringutil.Slugify(projectName),
			Repos: []string{},
		}
		reg.Projects = append(reg.Projects, *entry)
		// Point to the entry in the slice
		entry = &reg.Projects[len(reg.Projects)-1]
	}

	// Add repo if not already present
	found := false
	for _, r := range entry.Repos {
		if r == repoDir {
			found = true
			break
		}
	}
	if !found {
		entry.Repos = append(entry.Repos, repoDir)
	}

	// Create log directory
	if err := os.MkdirAll(LogDir(homeDir, entry.Slug), 0755); err != nil {
		return nil, false, err
	}

	// Write registry
	if err := WriteRegistry(homeDir, reg); err != nil {
		return nil, false, err
	}

	// Write per-repo config
	if err := WriteRepoConfig(repoDir, &RepoConfig{Project: projectName}); err != nil {
		return nil, false, err
	}

	return entry, created, nil
}
