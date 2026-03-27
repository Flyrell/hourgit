package e2e

import (
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipeline2_TwoRepos_OneProject_Precise tests two repos sharing a single
// project with precise tracking enabled.
// Repo A -> Project "Gamma" (precise)
// Repo B -> Project "Gamma" (precise)
func TestPipeline2_TwoRepos_OneProject_Precise(t *testing.T) {
	t.Parallel()
	env := NewTestEnv(t)

	repoA := env.AddRepo("repo-a")
	repoB := env.AddRepo("repo-b")

	// ===== PHASE 1: Simulated reflog =====

	// Interleave timestamps across repos for cross-repo chronology testing
	baseTime := time.Date(2026, 3, 16, 9, 0, 0, 0, time.UTC) // Monday March 16

	// Repo A: feat-1 and feat-2 work
	reflogA := NewReflogBuilder(repoA)
	reflogA.Checkout(repoA.Branch, "feat-1", baseTime)
	reflogA.Commit("feat-1: initial work", baseTime.Add(30*time.Minute))
	reflogA.Commit("feat-1: add validation", baseTime.Add(2*time.Hour))
	reflogA.Checkout("feat-1", "feat-2", baseTime.Add(4*time.Hour))
	reflogA.Commit("feat-2: database schema", baseTime.Add(5*time.Hour))
	reflogA.WriteTo(t)

	// Repo B: feat-3 and feat-4 work (interleaved timestamps with Repo A)
	reflogB := NewReflogBuilder(repoB)
	reflogB.Checkout(repoB.Branch, "feat-3", baseTime.Add(time.Hour))
	reflogB.Commit("feat-3: API endpoint", baseTime.Add(90*time.Minute))
	reflogB.Checkout("feat-3", "feat-4", baseTime.Add(3*time.Hour))
	reflogB.Commit("feat-4: frontend component", baseTime.Add(3*time.Hour+30*time.Minute))
	reflogB.Commit("feat-4: add styles", baseTime.Add(4*time.Hour+30*time.Minute))
	reflogB.WriteTo(t)

	// Initialize Repo A with the project
	env.MustRunInRepo("repo-a", "init", "--project", "Gamma", "--mode", "precise", "--yes")

	// Initialize Repo B and assign to same project.
	// init --yes without --project only installs the post-checkout hook (no project
	// is created). The --yes flag auto-accepts shell completion install. The actual
	// project assignment happens in the next command.
	env.MustRunInRepo("repo-b", "init", "--yes")
	env.MustRunInRepo("repo-b", "project", "assign", "Gamma", "--force", "--yes")

	gammaSlug := stringutil.Slugify("Gamma")

	// Add activity entries for both repos (idle gaps)
	// Repo A: 20 min idle gap during feat-1 work
	env.WriteActivityStop(gammaSlug, baseTime.Add(time.Hour), repoA.Dir)
	env.WriteActivityStart(gammaSlug, baseTime.Add(time.Hour+20*time.Minute), repoA.Dir)

	// Repo B: 15 min idle gap during feat-3 work
	env.WriteActivityStop(gammaSlug, baseTime.Add(2*time.Hour), repoB.Dir)
	env.WriteActivityStart(gammaSlug, baseTime.Add(2*time.Hour+15*time.Minute), repoB.Dir)

	// Sync both repos
	env.MustRunInRepo("repo-a", "sync")
	env.MustRunInRepo("repo-b", "sync")

	// --- Phase 1 tests ---

	// All entries should be in single project dir
	checkouts := env.ReadCheckoutEntries(gammaSlug)
	commits := env.ReadCommitEntries(gammaSlug)
	assert.Len(t, checkouts, 4, "Gamma should have exactly 4 checkouts from both repos")
	assert.Len(t, commits, 6, "Gamma should have exactly 6 commits from both repos")

	// Verify Repo field distinguishes repos
	repoACheckouts := filterCheckoutsByRepo(checkouts, repoA.Dir)
	repoBCheckouts := filterCheckoutsByRepo(checkouts, repoB.Dir)
	assert.Len(t, repoACheckouts, 2, "Should have exactly 2 checkouts from repo-a")
	assert.Len(t, repoBCheckouts, 2, "Should have exactly 2 checkouts from repo-b")

	repoACommits := filterCommitsByRepo(commits, repoA.Dir)
	repoBCommits := filterCommitsByRepo(commits, repoB.Dir)
	assert.Len(t, repoACommits, 3, "Should have exactly 3 commits from repo-a")
	assert.Len(t, repoBCommits, 3, "Should have exactly 3 commits from repo-b")

	// History should show entries from both repos interleaved by time
	history := env.MustRunInRepo("repo-a", "history", "--limit", "50")
	assert.Contains(t, history, "feat-1")
	assert.Contains(t, history, "feat-3")

	// Status from each repo should show different current branches
	statusA := env.MustRunInRepo("repo-a", "status")
	assert.Contains(t, statusA, "Gamma")

	statusB := env.MustRunInRepo("repo-b", "status")
	assert.Contains(t, statusB, "Gamma")

	// Verify activity entries exist for both repos
	actStops := env.ReadActivityStopEntries(gammaSlug)
	actStarts := env.ReadActivityStartEntries(gammaSlug)
	assert.Len(t, actStops, 2, "Should have activity stops for both repos")
	assert.Len(t, actStarts, 2, "Should have activity starts for both repos")

	// Verify project is precise
	gammaProj := env.FindProject("Gamma")
	assert.True(t, gammaProj.Precise, "Gamma should have precise tracking")
	assert.Len(t, gammaProj.Repos, 2, "Gamma should have 2 repos")

	// ===== PHASE 2: Real git changes =====

	// Real changes in Repo A
	env.GitCheckout("repo-a", "feat-5", true)
	env.GitCommit("repo-a", "feat-5: new API")

	// Real changes in Repo B
	env.GitCheckout("repo-b", "feat-6", true)
	env.GitCommit("repo-b", "feat-6: dashboard")

	// Sync both
	env.MustRunInRepo("repo-a", "sync")
	env.MustRunInRepo("repo-b", "sync")

	// --- Phase 2 tests ---

	// Verify incremental sync per-repo (each has own LastSync)
	repoConfigA := env.ReadRepoConfig("repo-a")
	repoConfigB := env.ReadRepoConfig("repo-b")
	require.NotNil(t, repoConfigA)
	require.NotNil(t, repoConfigB)
	assert.NotNil(t, repoConfigA.LastSync)
	assert.NotNil(t, repoConfigB.LastSync)

	// More entries should exist now
	checkouts2 := env.ReadCheckoutEntries(gammaSlug)
	commits2 := env.ReadCommitEntries(gammaSlug)
	assert.Greater(t, len(checkouts2), len(checkouts), "Should have more checkouts after phase 2")
	assert.Greater(t, len(commits2), len(commits), "Should have more commits after phase 2")

	// Add manual log entry via --project flag (not from within a repo)
	env.MustRun("log", "add", "--project", "Gamma", "--duration", "2h", "--task", "GAMMA-1", "--date", "2026-03-26", "--yes", "Cross-repo planning")

	// Verify manual log
	logs := env.ReadLogEntries(gammaSlug)
	assert.Len(t, logs, 1, "Should have 1 manual log")
	assert.Equal(t, "GAMMA-1", logs[0].Task)
	assert.Equal(t, 120, logs[0].Minutes)
	assert.Equal(t, "Cross-repo planning", logs[0].Message)

	// History should show manual log
	history2 := env.MustRunInRepo("repo-a", "history")
	assert.Contains(t, history2, "Cross-repo planning")
}
