package e2e

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipeline3_TwoRepos_OneProject_Normal tests two repos sharing a single
// project with standard (non-precise) tracking.
// Repo A -> Project "Delta" (standard)
// Repo B -> Project "Delta" (standard)
func TestPipeline3_TwoRepos_OneProject_Normal(t *testing.T) {
	t.Parallel()
	env := NewTestEnv(t)

	repoA := env.AddRepo("repo-a")
	repoB := env.AddRepo("repo-b")

	// ===== PHASE 1: Simulated reflog =====

	// Simulate 1 week of checkouts for both repos
	baseTime := time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC) // Monday March 23

	// Repo A: various branch work over the week
	reflogA := NewReflogBuilder(repoA)
	reflogA.Checkout(repoA.Branch, "feature-auth", baseTime)
	reflogA.Commit("auth: add login form", baseTime.Add(time.Hour))
	reflogA.Commit("auth: add validation", baseTime.Add(2*time.Hour))
	reflogA.Checkout("feature-auth", "feature-api", baseTime.Add(4*time.Hour))
	reflogA.Commit("api: REST endpoints", baseTime.Add(5*time.Hour))
	// Day 2
	reflogA.Checkout("feature-api", repoA.Branch, baseTime.Add(24*time.Hour))
	reflogA.Commit("merge: version bump", baseTime.Add(24*time.Hour+30*time.Minute))
	// Day 3
	reflogA.Checkout(repoA.Branch, "feature-auth", baseTime.Add(2*24*time.Hour))
	reflogA.Commit("auth: fix tests", baseTime.Add(2*24*time.Hour+time.Hour))
	reflogA.WriteTo(t)

	// Repo B: different branches, same week
	reflogB := NewReflogBuilder(repoB)
	reflogB.Checkout(repoB.Branch, "feature-docs", baseTime.Add(2*time.Hour))
	reflogB.Commit("docs: update readme", baseTime.Add(3*time.Hour))
	// Day 2
	reflogB.Checkout("feature-docs", "feature-ci", baseTime.Add(24*time.Hour+time.Hour))
	reflogB.Commit("ci: add pipeline", baseTime.Add(24*time.Hour+2*time.Hour))
	reflogB.Commit("ci: add test stage", baseTime.Add(24*time.Hour+3*time.Hour))
	// Day 3
	reflogB.Checkout("feature-ci", repoB.Branch, baseTime.Add(2*24*time.Hour+2*time.Hour))
	reflogB.Commit("merge: ci integration", baseTime.Add(2*24*time.Hour+3*time.Hour))
	reflogB.WriteTo(t)

	// Initialize Repo A with the project
	env.MustRunInRepo("repo-a", "init", "--project", "Delta", "--yes")

	// Initialize Repo B and assign to same project
	env.MustRunInRepo("repo-b", "init", "--yes")
	env.MustRunInRepo("repo-b", "project", "assign", "Delta", "--force", "--yes")

	deltaSlug := stringutil.Slugify("Delta")

	// Sync both repos
	env.MustRunInRepo("repo-a", "sync")
	env.MustRunInRepo("repo-b", "sync")

	// --- Phase 1 tests ---

	// Verify entries from both repos in same project dir
	checkouts := env.ReadCheckoutEntries(deltaSlug)
	commits := env.ReadCommitEntries(deltaSlug)
	assert.Len(t, checkouts, 7, "Delta should have exactly 7 checkouts from both repos")
	assert.Len(t, commits, 9, "Delta should have exactly 9 commits from both repos")

	// Verify no activity entries (standard mode)
	actStops := env.ReadActivityStopEntries(deltaSlug)
	actStarts := env.ReadActivityStartEntries(deltaSlug)
	assert.Empty(t, actStops, "Standard mode should have no activity stops")
	assert.Empty(t, actStarts, "Standard mode should have no activity starts")

	// Verify project is NOT precise
	deltaProj := env.FindProject("Delta")
	assert.False(t, deltaProj.Precise, "Delta should not have precise tracking")

	// History shows cross-repo entries
	history := env.MustRunInRepo("repo-a", "history", "--limit", "50")
	assert.Contains(t, history, "feature-auth")
	assert.Contains(t, history, "feature-docs")

	// Sync deduplication: re-sync should not create duplicates
	env.MustRunInRepo("repo-a", "sync")
	env.MustRunInRepo("repo-b", "sync")

	checkoutsAfterResync := env.ReadCheckoutEntries(deltaSlug)
	commitsAfterResync := env.ReadCommitEntries(deltaSlug)
	assert.Equal(t, len(checkouts), len(checkoutsAfterResync), "Re-sync should not duplicate checkouts")
	assert.Equal(t, len(commits), len(commitsAfterResync), "Re-sync should not duplicate commits")

	// ===== PHASE 2: Real git changes =====

	// Repo A: multiple checkouts and commits
	env.GitCheckout("repo-a", "hotfix-1", true)
	env.GitCommit("repo-a", "hotfix-1: critical fix")
	env.GitCheckout("repo-a", repoA.Branch, false)
	env.GitCommit("repo-a", "merge: hotfix applied")

	// Repo B: checkout to a branch that also exists in Repo A's reflog (test cross-repo naming)
	env.GitCheckout("repo-b", "feature-auth", true) // same name as in repo-a's simulated reflog
	env.GitCommit("repo-b", "auth: separate implementation")

	// Sync both
	env.MustRunInRepo("repo-a", "sync")
	env.MustRunInRepo("repo-b", "sync")

	// --- Phase 2 tests ---

	// Verify all new entries synced
	checkouts2 := env.ReadCheckoutEntries(deltaSlug)
	commits2 := env.ReadCommitEntries(deltaSlug)
	assert.Greater(t, len(checkouts2), len(checkouts), "Should have more checkouts after phase 2")
	assert.Greater(t, len(commits2), len(commits), "Should have more commits after phase 2")

	// Verify same branch name in different repos uses Repo field for distinction
	authCommits := filterCommitsByBranch(commits2, "feature-auth")
	if len(authCommits) > 0 {
		repoAAuth := filterCommitsByRepo(authCommits, repoA.Dir)
		repoBAuth := filterCommitsByRepo(authCommits, repoB.Dir)
		assert.NotEmpty(t, repoAAuth, "Should have feature-auth commits from repo-a")
		assert.NotEmpty(t, repoBAuth, "Should have feature-auth commits from repo-b")
	}

	// Test log CRUD operations
	// Add
	addOut := env.MustRunInRepo("repo-a", "log", "add", "--duration", "3h", "--task", "DELTA-1", "--date", "2026-03-26", "--yes", "Design meeting")
	assert.NotEmpty(t, addOut)

	logs := env.ReadLogEntries(deltaSlug)
	require.Len(t, logs, 1)
	logID := logs[0].ID
	assert.Equal(t, 180, logs[0].Minutes)
	assert.Equal(t, "DELTA-1", logs[0].Task)

	// Edit
	env.MustRunInRepo("repo-a", "log", "edit", logID, "--duration", "2h30m", "--message", "Design review meeting", "--yes")
	logsEdited := env.ReadLogEntries(deltaSlug)
	require.Len(t, logsEdited, 1)
	assert.Equal(t, 150, logsEdited[0].Minutes)
	assert.Equal(t, "Design review meeting", logsEdited[0].Message)
	assert.Equal(t, logID, logsEdited[0].ID, "ID should be preserved")

	// Remove
	env.MustRunInRepo("repo-a", "log", "remove", logID, "--yes")
	logsRemoved := env.ReadLogEntries(deltaSlug)
	assert.Empty(t, logsRemoved, "Log entry should be removed")

	// Test PDF export
	pdfOut := env.MustRunInRepo("repo-a", "report", "--month", "3", "--year", "2026", "--export", "pdf")
	assert.Contains(t, pdfOut, ".pdf")

	// Verify PDF file was created — simulated data covers March 2026 so a PDF must exist
	pdfPath := filepath.Join(repoA.Dir, "delta-2026-month-03.pdf")
	assert.FileExists(t, pdfPath, "PDF export should create file for March 2026")
}

func filterCommitsByBranch(commits []entry.CommitEntry, branch string) []entry.CommitEntry {
	var filtered []entry.CommitEntry
	for _, c := range commits {
		if c.Branch == branch {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
