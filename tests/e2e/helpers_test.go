package e2e

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/stretchr/testify/require"
)

var fileCounter atomic.Int64

// TestEnv provides an isolated sandbox for e2e tests.
// Each TestEnv has its own HOME directory and set of git repos.
type TestEnv struct {
	T       *testing.T
	HomeDir string
	Repos   map[string]*TestRepo
}

// TestRepo represents a real git repository in a temp directory.
type TestRepo struct {
	Dir    string
	Name   string
	SHAs   []string // collected commit SHAs for use in simulated reflog
	Branch string   // default branch name (master or main depending on git version)
}

// NewTestEnv creates an isolated test environment with its own HOME dir.
func NewTestEnv(t *testing.T) *TestEnv {
	t.Helper()
	if skipE2E {
		t.Skip("e2e tests require git")
	}
	homeDir := t.TempDir()
	return &TestEnv{
		T:       t,
		HomeDir: homeDir,
		Repos:   make(map[string]*TestRepo),
	}
}

// AddRepo creates a new real git repository with multiple commits
// to provide valid SHAs for simulated reflog entries.
func (env *TestEnv) AddRepo(name string) *TestRepo {
	env.T.Helper()
	dir := env.T.TempDir()

	env.git(dir, "init")
	env.git(dir, "config", "user.name", "E2E Test")
	env.git(dir, "config", "user.email", "e2e@test.com")

	repo := &TestRepo{Dir: dir, Name: name}
	env.Repos[name] = repo

	// Create multiple commits to build a pool of valid SHAs.
	// Git reflog validates that SHAs reference real objects, so simulated
	// reflog entries must use SHAs from actual commits.
	//
	// Each ReflogBuilder operation (Checkout/Commit) consumes 2 SHAs
	// (oldSHA + newSHA), so 20 commits = 20 SHAs supports up to 10 reflog
	// entries per repo. If the index exceeds the pool size, SHAs wrap via
	// modulo (see ReflogBuilder.nextSHA).
	for i := 0; i < 20; i++ {
		n := fileCounter.Add(1)
		filePath := filepath.Join(dir, fmt.Sprintf("seed-%d.txt", n))
		require.NoError(env.T, os.WriteFile(filePath, []byte(fmt.Sprintf("seed %d\n", n)), 0644))
		env.git(dir, "add", filepath.Base(filePath))
		env.git(dir, "commit", "-m", fmt.Sprintf("seed commit %d", i))
		sha := strings.TrimSpace(env.git(dir, "rev-parse", "HEAD"))
		repo.SHAs = append(repo.SHAs, sha)
	}

	// Detect the default branch name (master vs main)
	repo.Branch = strings.TrimSpace(env.git(dir, "rev-parse", "--abbrev-ref", "HEAD"))

	// Clear the reflog so seed commits don't pollute sync results.
	// The SHAs are still valid git objects — they just won't appear in git reflog output.
	headLog := filepath.Join(dir, ".git", "logs", "HEAD")
	require.NoError(env.T, os.Truncate(headLog, 0))

	return repo
}

// Run executes the hourgit binary with the given args and HOME set to env.HomeDir.
// Returns stdout, stderr, and error.
func (env *TestEnv) Run(args ...string) (string, string, error) {
	env.T.Helper()
	return env.runCmd("", args...)
}

// RunInRepo executes hourgit from within the given repo directory.
func (env *TestEnv) RunInRepo(repoName string, args ...string) (string, string, error) {
	env.T.Helper()
	repo, ok := env.Repos[repoName]
	require.True(env.T, ok, "repo %q not found in test env", repoName)
	return env.runCmd(repo.Dir, args...)
}

// MustRunInRepo runs hourgit in a repo and fails the test if it errors.
func (env *TestEnv) MustRunInRepo(repoName string, args ...string) string {
	env.T.Helper()
	stdout, stderr, err := env.RunInRepo(repoName, args...)
	require.NoError(env.T, err, "hourgit %v failed:\nstdout: %s\nstderr: %s", args, stdout, stderr)
	return stdout
}

// MustRun runs hourgit and fails the test if it errors.
func (env *TestEnv) MustRun(args ...string) string {
	env.T.Helper()
	stdout, stderr, err := env.Run(args...)
	require.NoError(env.T, err, "hourgit %v failed:\nstdout: %s\nstderr: %s", args, stdout, stderr)
	return stdout
}

func (env *TestEnv) runCmd(dir string, args ...string) (string, string, error) {
	env.T.Helper()
	// Add --skip-updates and --skip-watcher to avoid interactive prompts and service checks
	fullArgs := append([]string{"--skip-updates", "--skip-watcher"}, args...)
	cmd := exec.Command(binaryPath, fullArgs...)
	cmd.Env = append(filterHostEnv(), "HOME="+env.HomeDir)
	if dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

// GitInRepo runs a git command in the given repo.
func (env *TestEnv) GitInRepo(repoName string, gitArgs ...string) string {
	env.T.Helper()
	repo, ok := env.Repos[repoName]
	require.True(env.T, ok, "repo %q not found in test env", repoName)
	return env.git(repo.Dir, gitArgs...)
}

func (env *TestEnv) git(dir string, args ...string) string {
	env.T.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(filterHostEnv(), "HOME="+env.HomeDir)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	require.NoError(env.T, err, "git %v in %s failed:\nstdout: %s\nstderr: %s", args, dir, stdout.String(), stderr.String())
	return stdout.String()
}

// GitCommit creates a unique file, stages, and commits it.
func (env *TestEnv) GitCommit(repoName, message string) {
	env.T.Helper()
	repo := env.Repos[repoName]
	require.NotNil(env.T, repo, "repo %q not found", repoName)

	n := fileCounter.Add(1)
	filePath := filepath.Join(repo.Dir, fmt.Sprintf("file-%d.txt", n))
	require.NoError(env.T, os.WriteFile(filePath, []byte(fmt.Sprintf("content %d\n", n)), 0644))

	env.git(repo.Dir, "add", filepath.Base(filePath))
	env.git(repo.Dir, "commit", "-m", message)
}

// GitCheckout checks out a branch in the given repo.
func (env *TestEnv) GitCheckout(repoName, branch string, create bool) {
	env.T.Helper()
	repo := env.Repos[repoName]
	require.NotNil(env.T, repo, "repo %q not found", repoName)

	if create {
		env.git(repo.Dir, "checkout", "-b", branch)
	} else {
		env.git(repo.Dir, "checkout", branch)
	}
}

// --- Entry readers ---

// ReadCheckoutEntries reads all checkout entries for a project slug.
func (env *TestEnv) ReadCheckoutEntries(slug string) []entry.CheckoutEntry {
	env.T.Helper()
	entries, err := entry.ReadAllCheckoutEntries(env.HomeDir, slug)
	require.NoError(env.T, err)
	return entries
}

// ReadCommitEntries reads all commit entries for a project slug.
func (env *TestEnv) ReadCommitEntries(slug string) []entry.CommitEntry {
	env.T.Helper()
	entries, err := entry.ReadAllCommitEntries(env.HomeDir, slug)
	require.NoError(env.T, err)
	return entries
}

// ReadLogEntries reads all manual log entries for a project slug.
func (env *TestEnv) ReadLogEntries(slug string) []entry.Entry {
	env.T.Helper()
	entries, err := entry.ReadAllEntries(env.HomeDir, slug)
	require.NoError(env.T, err)
	return entries
}

// ReadActivityStopEntries reads all activity stop entries for a project slug.
func (env *TestEnv) ReadActivityStopEntries(slug string) []entry.ActivityStopEntry {
	env.T.Helper()
	entries, err := entry.ReadAllActivityStopEntries(env.HomeDir, slug)
	require.NoError(env.T, err)
	return entries
}

// ReadActivityStartEntries reads all activity start entries for a project slug.
func (env *TestEnv) ReadActivityStartEntries(slug string) []entry.ActivityStartEntry {
	env.T.Helper()
	entries, err := entry.ReadAllActivityStartEntries(env.HomeDir, slug)
	require.NoError(env.T, err)
	return entries
}

// --- Config readers ---

// ReadConfig reads the global hourgit config.
func (env *TestEnv) ReadConfig() *project.Config {
	env.T.Helper()
	cfg, err := project.ReadConfig(env.HomeDir)
	require.NoError(env.T, err)
	return cfg
}

// ReadRepoConfig reads the per-repo hourgit config.
func (env *TestEnv) ReadRepoConfig(repoName string) *project.RepoConfig {
	env.T.Helper()
	repo := env.Repos[repoName]
	require.NotNil(env.T, repo, "repo %q not found", repoName)
	rc, err := project.ReadRepoConfig(repo.Dir)
	require.NoError(env.T, err)
	return rc
}

// FindProject finds a project by name in the config.
func (env *TestEnv) FindProject(name string) *project.ProjectEntry {
	env.T.Helper()
	cfg := env.ReadConfig()
	p := project.FindProject(cfg, name)
	require.NotNil(env.T, p, "project %q not found in config", name)
	return p
}

// --- Activity entry writers (for precise mode testing) ---

// WriteActivityStop writes an activity_stop entry directly to the project dir.
func (env *TestEnv) WriteActivityStop(slug string, timestamp time.Time, repo string) {
	env.T.Helper()
	e := entry.ActivityStopEntry{
		ID:        generateTestID(),
		Timestamp: timestamp,
		Repo:      repo,
	}
	err := entry.WriteActivityStopEntry(env.HomeDir, slug, e)
	require.NoError(env.T, err)
}

// WriteActivityStart writes an activity_start entry directly to the project dir.
func (env *TestEnv) WriteActivityStart(slug string, timestamp time.Time, repo string) {
	env.T.Helper()
	e := entry.ActivityStartEntry{
		ID:        generateTestID(),
		Timestamp: timestamp,
		Repo:      repo,
	}
	err := entry.WriteActivityStartEntry(env.HomeDir, slug, e)
	require.NoError(env.T, err)
}

// generateTestID creates a 7-char hex ID using the production hash scheme.
// Each call produces a unique ID via an incrementing counter.
func generateTestID() string {
	n := fileCounter.Add(1)
	return hashutil.GenerateIDFromSeed(fmt.Sprintf("e2e-activity-%d", n))
}

// filterHostEnv returns os.Environ() with HOME, XDG_CONFIG_HOME, and any
// HOURGIT_-prefixed variables removed, preventing the host environment from
// leaking into test subprocesses.
func filterHostEnv() []string {
	var filtered []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, "HOME=") ||
			strings.HasPrefix(e, "XDG_CONFIG_HOME=") ||
			strings.HasPrefix(e, "HOURGIT_") {
			continue
		}
		filtered = append(filtered, e)
	}
	return filtered
}

// --- Shared filter helpers ---

func filterCheckoutsByRepo(checkouts []entry.CheckoutEntry, repoDir string) []entry.CheckoutEntry {
	var filtered []entry.CheckoutEntry
	for _, c := range checkouts {
		if c.Repo == repoDir {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

func filterCommitsByRepo(commits []entry.CommitEntry, repoDir string) []entry.CommitEntry {
	var filtered []entry.CommitEntry
	for _, c := range commits {
		if c.Repo == repoDir {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

