package project

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/schedule"
	"github.com/Flyrell/hourgit/internal/stringutil"
)

// HookMarker is the comment marker written into the post-checkout hook.
// Use this constant for detection — never redefine it elsewhere.
const HookMarker = "# Installed by hourgit"

var appVersion = "dev"

// SetVersion sets the app version stamped into config.json on write.
func SetVersion(v string) {
	appVersion = v
}

// RepoConfig is the per-repo marker stored in .git/.hourgit.
type RepoConfig struct {
	Project   string `json:"project"`
	ProjectID string `json:"project_id,omitempty"`
}

// ProjectEntry represents a single project in the global registry.
type ProjectEntry struct {
	ID        string                   `json:"id"`
	Name      string                   `json:"name"`
	Slug      string                   `json:"slug"`
	Repos     []string                 `json:"repos"`
	Schedules []schedule.ScheduleEntry `json:"schedules,omitempty"`
}

// Config holds the global hourgit configuration including projects and defaults.
type Config struct {
	Version  string                   `json:"version"`
	Defaults []schedule.ScheduleEntry `json:"defaults"`
	Projects []ProjectEntry           `json:"projects"`
}

// HourgitDir returns the global hourgit config directory.
func HourgitDir(homeDir string) string {
	return filepath.Join(homeDir, ".hourgit")
}

// ConfigPath returns the path to the global config.json.
func ConfigPath(homeDir string) string {
	return filepath.Join(HourgitDir(homeDir), "config.json")
}

// LogDir returns the directory for a project's log entries.
func LogDir(homeDir, slug string) string {
	return filepath.Join(HourgitDir(homeDir), slug)
}

// ReadConfig reads the global hourgit configuration.
// Returns a fresh config with factory defaults if the file does not exist.
func ReadConfig(homeDir string) (*Config, error) {
	data, err := os.ReadFile(ConfigPath(homeDir))
	if errors.Is(err, os.ErrNotExist) {
		return &Config{
			Defaults: schedule.DefaultSchedules(),
		}, nil
	}
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// WriteConfig writes the global hourgit configuration, creating the directory if needed.
func WriteConfig(homeDir string, cfg *Config) error {
	dir := HourgitDir(homeDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	cfg.Version = appVersion

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath(homeDir), data, 0644)
}

// FindProject looks up a project by name in the config.
// Returns nil if not found.
func FindProject(cfg *Config, name string) *ProjectEntry {
	for i := range cfg.Projects {
		if cfg.Projects[i].Name == name {
			return &cfg.Projects[i]
		}
	}
	return nil
}

// FindProjectByID looks up a project by ID in the config.
// Returns nil if not found.
func FindProjectByID(cfg *Config, id string) *ProjectEntry {
	for i := range cfg.Projects {
		if cfg.Projects[i].ID == id {
			return &cfg.Projects[i]
		}
	}
	return nil
}

// ResolveProject looks up a project by ID first, then by name.
// Returns nil if not found by either.
func ResolveProject(cfg *Config, identifier string) *ProjectEntry {
	if entry := FindProjectByID(cfg, identifier); entry != nil {
		return entry
	}
	return FindProject(cfg, identifier)
}

// ResolveOrCreateResult holds the outcome of ResolveOrCreate.
type ResolveOrCreateResult struct {
	Entry   *ProjectEntry
	Created bool
}

// ResolveOrCreate looks up a project by ID or name. If not found, it prompts
// the user to create it. Returns nil result (no error) if the user declines.
func ResolveOrCreate(homeDir, identifier string, promptCreate func(name string) (bool, error)) (*ResolveOrCreateResult, error) {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}
	if entry := ResolveProject(cfg, identifier); entry != nil {
		return &ResolveOrCreateResult{Entry: entry, Created: false}, nil
	}
	confirmed, err := promptCreate(identifier)
	if err != nil {
		return nil, err
	}
	if !confirmed {
		return nil, nil
	}
	entry, err := CreateProject(homeDir, identifier)
	if err != nil {
		return nil, err
	}
	return &ResolveOrCreateResult{Entry: entry, Created: true}, nil
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

	var rc RepoConfig
	if err := json.Unmarshal(data, &rc); err != nil {
		return nil, err
	}
	return &rc, nil
}

// WriteRepoConfig writes the per-repo hourgit config to .git/.hourgit.
func WriteRepoConfig(repoDir string, rc *RepoConfig) error {
	data, err := json.MarshalIndent(rc, "", "  ")
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

// CreateProject creates a new project in the config.
// Returns an error if a project with the same name already exists.
func CreateProject(homeDir, name string) (*ProjectEntry, error) {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	if existing := FindProject(cfg, name); existing != nil {
		return nil, fmt.Errorf("project '%s' already exists (%s)", name, existing.ID)
	}

	entry := ProjectEntry{
		ID:        hashutil.GenerateID(name),
		Name:      name,
		Slug:      stringutil.Slugify(name),
		Repos:     []string{},
		Schedules: GetDefaults(cfg),
	}
	cfg.Projects = append(cfg.Projects, entry)

	if err := os.MkdirAll(LogDir(homeDir, entry.Slug), 0755); err != nil {
		return nil, err
	}

	if err := WriteConfig(homeDir, cfg); err != nil {
		return nil, err
	}

	return &entry, nil
}

// AssignProject assigns a repository to an existing project.
// It adds repoDir to the project's repos list (deduplicated) and writes the per-repo config.
func AssignProject(homeDir, repoDir string, entry *ProjectEntry) error {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return err
	}

	cfgEntry := FindProjectByID(cfg, entry.ID)
	if cfgEntry == nil {
		return fmt.Errorf("project '%s' not found in registry", entry.Name)
	}

	// Add repo if not already present
	found := false
	for _, r := range cfgEntry.Repos {
		if r == repoDir {
			found = true
			break
		}
	}
	if !found {
		cfgEntry.Repos = append(cfgEntry.Repos, repoDir)
	}

	if err := WriteConfig(homeDir, cfg); err != nil {
		return err
	}

	return WriteRepoConfig(repoDir, &RepoConfig{Project: cfgEntry.Name, ProjectID: cfgEntry.ID})
}

// RemoveProject removes a project from the config by ID or name.
// Returns the removed entry so the caller can handle cleanup.
func RemoveProject(homeDir, identifier string) (*ProjectEntry, error) {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return nil, err
	}

	idx := -1
	for i := range cfg.Projects {
		if cfg.Projects[i].ID == identifier || cfg.Projects[i].Name == identifier {
			idx = i
			break
		}
	}

	if idx == -1 {
		return nil, fmt.Errorf("project '%s' not found", identifier)
	}

	removed := cfg.Projects[idx]
	cfg.Projects = append(cfg.Projects[:idx], cfg.Projects[idx+1:]...)

	if err := WriteConfig(homeDir, cfg); err != nil {
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

// GetDefaults returns the user's default schedules, falling back to factory settings.
func GetDefaults(cfg *Config) []schedule.ScheduleEntry {
	if len(cfg.Defaults) > 0 {
		return cfg.Defaults
	}
	return schedule.DefaultSchedules()
}

// SetDefaults updates the default schedules in the config.
func SetDefaults(homeDir string, schedules []schedule.ScheduleEntry) error {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return err
	}
	cfg.Defaults = schedules
	return WriteConfig(homeDir, cfg)
}

// ResetDefaults resets the default schedules to factory settings.
func ResetDefaults(homeDir string) error {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return err
	}
	cfg.Defaults = schedule.DefaultSchedules()
	return WriteConfig(homeDir, cfg)
}

// GetSchedules returns the schedules for a project, falling back to defaults if empty.
func GetSchedules(cfg *Config, projectID string) []schedule.ScheduleEntry {
	entry := FindProjectByID(cfg, projectID)
	if entry == nil || len(entry.Schedules) == 0 {
		return GetDefaults(cfg)
	}
	return entry.Schedules
}

// SetSchedules updates the schedules for a project in the config.
func SetSchedules(homeDir, projectID string, schedules []schedule.ScheduleEntry) error {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return err
	}

	entry := FindProjectByID(cfg, projectID)
	if entry == nil {
		return fmt.Errorf("project '%s' not found", projectID)
	}

	entry.Schedules = schedules
	return WriteConfig(homeDir, cfg)
}

// ResetSchedules resets a project's schedules to the current defaults.
func ResetSchedules(homeDir, projectID string) error {
	cfg, err := ReadConfig(homeDir)
	if err != nil {
		return err
	}
	defaults := GetDefaults(cfg)
	return SetSchedules(homeDir, projectID, defaults)
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
	markerIdx := strings.Index(content, HookMarker)
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
