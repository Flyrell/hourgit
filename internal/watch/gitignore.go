package watch

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// ShouldIgnore checks if a file path should be ignored based on the repo's
// .gitignore patterns and built-in exclusions. Reads .gitignore from disk
// on each call — use ShouldIgnoreWithPatterns for the hot path.
func ShouldIgnore(repoDir, filePath string) bool {
	patterns := LoadGitignorePatterns(repoDir)
	return ShouldIgnoreWithPatterns(repoDir, filePath, patterns)
}

// ShouldIgnoreWithPatterns checks if a file path should be ignored using
// pre-loaded gitignore patterns. Use this on the hot path to avoid re-reading
// .gitignore from disk on every event.
func ShouldIgnoreWithPatterns(repoDir, filePath string, patterns []string) bool {
	// Always exclude .git directory
	rel, err := filepath.Rel(repoDir, filePath)
	if err != nil {
		return true
	}
	parts := strings.Split(rel, string(filepath.Separator))
	for _, p := range parts {
		if p == ".git" {
			return true
		}
	}

	return matchesAnyPattern(rel, patterns)
}

// LoadGitignorePatterns reads .gitignore from the repo root and returns patterns.
func LoadGitignorePatterns(repoDir string) []string {
	path := filepath.Join(repoDir, ".gitignore")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()

	var patterns []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns
}

// matchesAnyPattern checks if a relative path matches any gitignore pattern.
// Supports basic patterns: exact names, directory prefixes, and wildcard globs.
func matchesAnyPattern(relPath string, patterns []string) bool {
	for _, pattern := range patterns {
		if matchPattern(relPath, pattern) {
			return true
		}
	}
	return false
}

// matchPattern checks a single gitignore pattern against a relative path.
func matchPattern(relPath, pattern string) bool {
	// Handle negation (we don't support it — skip)
	if strings.HasPrefix(pattern, "!") {
		return false
	}

	// Strip trailing slash (directory marker)
	pattern = strings.TrimSuffix(pattern, "/")

	// Check each path component for basename match
	parts := strings.Split(relPath, string(filepath.Separator))

	// If pattern contains a slash, match against the full path
	if strings.Contains(pattern, "/") {
		matched, _ := filepath.Match(pattern, relPath)
		return matched
	}

	// Otherwise, match against any path component
	for _, part := range parts {
		matched, _ := filepath.Match(pattern, part)
		if matched {
			return true
		}
	}
	return false
}
