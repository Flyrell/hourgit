package e2e

import (
	"strings"
	"testing"
	"time"

	"github.com/Flyrell/hourgit/internal/stringutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPipeline1_TwoRepos_TwoProjects tests two repos with separate projects:
// Repo A -> Project "Alpha" (precise tracking)
// Repo B -> Project "Beta"  (normal tracking)
func TestPipeline1_TwoRepos_TwoProjects(t *testing.T) {
	t.Parallel()
	env := NewTestEnv(t)

	repoA := env.AddRepo("repo-a")
	repoB := env.AddRepo("repo-b")

	// ===== PHASE 1: Simulated reflog =====

	// Simulate 2 weeks of checkouts for Repo A (precise project)
	baseTime := time.Date(2026, 3, 9, 9, 0, 0, 0, time.UTC) // Monday March 9

	reflogA := NewReflogBuilder(repoA)
	reflogA.Checkout(repoA.Branch, "feat-a", baseTime)
	reflogA.Commit("feat-a: initial setup", baseTime.Add(30*time.Minute))
	reflogA.Commit("feat-a: add tests", baseTime.Add(2*time.Hour))
	reflogA.Checkout("feat-a", "feat-b", baseTime.Add(4*time.Hour))
	reflogA.Commit("feat-b: scaffold", baseTime.Add(5*time.Hour))
	// Week 2
	reflogA.Checkout("feat-b", repoA.Branch, baseTime.Add(7*24*time.Hour))
	reflogA.Commit(repoA.Branch+": merge cleanup", baseTime.Add(7*24*time.Hour+time.Hour))
	reflogA.WriteTo(t)

	// Simulate 2 weeks of checkouts for Repo B (normal project)
	reflogB := NewReflogBuilder(repoB)
	reflogB.Checkout(repoB.Branch, "bugfix-1", baseTime.Add(time.Hour))
	reflogB.Commit("bugfix-1: fix login issue", baseTime.Add(2*time.Hour))
	reflogB.Checkout("bugfix-1", repoB.Branch, baseTime.Add(3*time.Hour))
	reflogB.Checkout(repoB.Branch, "feat-x", baseTime.Add(24*time.Hour+9*time.Hour))
	reflogB.Commit("feat-x: start feature", baseTime.Add(24*time.Hour+10*time.Hour))
	reflogB.WriteTo(t)

	// Initialize projects
	env.MustRunInRepo("repo-a", "init", "--project", "Alpha", "--mode", "precise", "--yes")
	env.MustRunInRepo("repo-b", "init", "--project", "Beta", "--yes")

	// Write activity entries for precise mode on Repo A
	// Idle gap: 30 min idle starting 1h into feat-a work
	alphaSlug := stringutil.Slugify("Alpha")
	env.WriteActivityStop(alphaSlug, baseTime.Add(time.Hour), repoA.Dir)
	env.WriteActivityStart(alphaSlug, baseTime.Add(time.Hour+30*time.Minute), repoA.Dir)

	// Sync both repos
	syncOutA := env.MustRunInRepo("repo-a", "sync")
	syncOutB := env.MustRunInRepo("repo-b", "sync")

	// --- Phase 1 tests ---

	// Verify sync output mentions checkouts and commits
	assert.Contains(t, syncOutA, "checkout")
	assert.Contains(t, syncOutB, "checkout")

	// Verify checkout entries created for each project separately
	alphaCheckouts := env.ReadCheckoutEntries(alphaSlug)
	betaSlug := stringutil.Slugify("Beta")
	betaCheckouts := env.ReadCheckoutEntries(betaSlug)

	assert.Len(t, alphaCheckouts, 3, "Alpha should have exactly 3 checkouts")
	assert.Len(t, betaCheckouts, 3, "Beta should have exactly 3 checkouts")

	// Verify commit entries have correct branch attribution
	alphaCommits := env.ReadCommitEntries(alphaSlug)
	betaCommits := env.ReadCommitEntries(betaSlug)

	assert.Len(t, alphaCommits, 4, "Alpha should have exactly 4 commits")
	assert.Len(t, betaCommits, 2, "Beta should have exactly 2 commits")

	// Check commit branch attribution for Alpha
	for _, c := range alphaCommits {
		if strings.Contains(c.Message, "feat-a") {
			assert.Equal(t, "feat-a", c.Branch, "feat-a commit should be on feat-a branch")
		}
		if strings.Contains(c.Message, "feat-b") {
			assert.Equal(t, "feat-b", c.Branch, "feat-b commit should be on feat-b branch")
		}
	}

	// Verify Repo A has precise mode, Repo B doesn't
	alphaProj := env.FindProject("Alpha")
	betaProj := env.FindProject("Beta")
	assert.True(t, alphaProj.Precise, "Alpha should have precise tracking")
	assert.False(t, betaProj.Precise, "Beta should not have precise tracking")

	// Verify status shows correct project/branch
	statusA := env.MustRunInRepo("repo-a", "status")
	assert.Contains(t, statusA, "Alpha")

	statusB := env.MustRunInRepo("repo-b", "status")
	assert.Contains(t, statusB, "Beta")

	// Verify history shows entries per project (use --project flag for isolation)
	historyA := env.MustRunInRepo("repo-a", "history", "--project", "Alpha")
	assert.Contains(t, historyA, "feat-a")
	assert.NotContains(t, historyA, "bugfix-1", "Alpha history should not contain Beta entries")

	historyB := env.MustRunInRepo("repo-b", "history", "--project", "Beta")
	assert.Contains(t, historyB, "bugfix-1")
	assert.NotContains(t, historyB, "feat-a", "Beta history should not contain Alpha entries")

	// Verify project list shows both projects
	projectList := env.MustRun("project", "list")
	assert.Contains(t, projectList, "Alpha")
	assert.Contains(t, projectList, "Beta")

	// Verify activity entries exist for Alpha
	actStops := env.ReadActivityStopEntries(alphaSlug)
	actStarts := env.ReadActivityStartEntries(alphaSlug)
	assert.Len(t, actStops, 1, "Alpha should have 1 activity stop")
	assert.Len(t, actStarts, 1, "Alpha should have 1 activity start")

	// ===== PHASE 2: Real git changes =====

	// Make real changes in Repo A
	env.GitCheckout("repo-a", "feat-c", true)
	env.GitCommit("repo-a", "feat-c: new feature")

	// Make real changes in Repo B
	env.GitCheckout("repo-b", "feat-y", true)
	env.GitCommit("repo-b", "feat-y: started work")

	// Sync again
	syncOutA2 := env.MustRunInRepo("repo-a", "sync")
	syncOutB2 := env.MustRunInRepo("repo-b", "sync")

	// Verify incremental sync picked up new entries
	assert.Contains(t, syncOutA2, "checkout")
	assert.Contains(t, syncOutB2, "checkout")

	// Verify LastSync was updated
	repoConfigA := env.ReadRepoConfig("repo-a")
	require.NotNil(t, repoConfigA)
	assert.NotNil(t, repoConfigA.LastSync, "LastSync should be set after sync")

	repoConfigB := env.ReadRepoConfig("repo-b")
	require.NotNil(t, repoConfigB)
	assert.NotNil(t, repoConfigB.LastSync, "LastSync should be set after sync")

	// New checkout entries should exist alongside simulated ones
	alphaCheckouts2 := env.ReadCheckoutEntries(alphaSlug)
	betaCheckouts2 := env.ReadCheckoutEntries(betaSlug)
	assert.Greater(t, len(alphaCheckouts2), len(alphaCheckouts), "Alpha should have more checkouts after phase 2")
	assert.Greater(t, len(betaCheckouts2), len(betaCheckouts), "Beta should have more checkouts after phase 2")

	// Add manual log entries
	logOutA := env.MustRunInRepo("repo-a", "log", "add", "--duration", "1h30m", "--task", "ALPHA-1", "--date", "2026-03-26", "--yes", "Manual research work")
	assert.NotEmpty(t, logOutA)

	logOutB := env.MustRunInRepo("repo-b", "log", "add", "--duration", "45m", "--task", "BETA-1", "--date", "2026-03-26", "--yes", "Bug analysis")
	assert.NotEmpty(t, logOutB)

	// Verify manual log entries created
	alphaLogs := env.ReadLogEntries(alphaSlug)
	betaLogs := env.ReadLogEntries(betaSlug)
	assert.Len(t, alphaLogs, 1, "Alpha should have 1 manual log entry")
	assert.Len(t, betaLogs, 1, "Beta should have 1 manual log entry")

	assert.Equal(t, 90, alphaLogs[0].Minutes, "Alpha log should be 90 minutes")
	assert.Equal(t, "ALPHA-1", alphaLogs[0].Task)
	assert.Equal(t, 45, betaLogs[0].Minutes, "Beta log should be 45 minutes")
	assert.Equal(t, "BETA-1", betaLogs[0].Task)

	// History should show all entry types
	historyA2 := env.MustRunInRepo("repo-a", "history")
	assert.Contains(t, historyA2, "Manual research work")

	historyB2 := env.MustRunInRepo("repo-b", "history")
	assert.Contains(t, historyB2, "Bug analysis")

	// Status should show new current branches
	statusA2 := env.MustRunInRepo("repo-a", "status")
	assert.Contains(t, statusA2, "feat-c")

	statusB2 := env.MustRunInRepo("repo-b", "status")
	assert.Contains(t, statusB2, "feat-y")

	// Test log edit: change duration
	alphaLogID := alphaLogs[0].ID
	env.MustRunInRepo("repo-a", "log", "edit", alphaLogID, "--duration", "2h", "--yes")

	// Verify edit took effect
	alphaLogsEdited := env.ReadLogEntries(alphaSlug)
	require.Len(t, alphaLogsEdited, 1)
	assert.Equal(t, 120, alphaLogsEdited[0].Minutes, "Edited entry should be 120 minutes")
	assert.Equal(t, alphaLogID, alphaLogsEdited[0].ID, "Entry ID should be preserved after edit")

	// Test log remove: remove a checkout entry
	require.NotEmpty(t, alphaCheckouts2)
	removeTarget := alphaCheckouts2[0].ID
	env.MustRunInRepo("repo-a", "log", "remove", removeTarget, "--yes")

	// Verify removal
	alphaCheckouts3 := env.ReadCheckoutEntries(alphaSlug)
	assert.Equal(t, len(alphaCheckouts2)-1, len(alphaCheckouts3), "Should have one fewer checkout after removal")
	for _, c := range alphaCheckouts3 {
		assert.NotEqual(t, removeTarget, c.ID, "Removed entry should not exist")
	}
}
