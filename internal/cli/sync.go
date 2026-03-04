package cli

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
	"github.com/Flyrell/hourgit/internal/hashutil"
	"github.com/Flyrell/hourgit/internal/project"
	"github.com/Flyrell/hourgit/internal/reflog"
	"github.com/spf13/cobra"
)

// GitReflogFunc executes git reflog and returns its output.
// The since parameter is non-nil when LastSync is available.
type GitReflogFunc func(repoDir string, since *time.Time) (string, error)

// defaultGitReflog runs git reflog in the given repo directory.
func defaultGitReflog(repoDir string, since *time.Time) (string, error) {
	args := []string{"-C", repoDir, "reflog", "--date=iso"}
	if since != nil {
		args = append(args, fmt.Sprintf("--since=%s", since.Format("2006-01-02 15:04:05")))
	}
	out, err := exec.Command("git", args...).Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

var syncCmd = LeafCommand{
	Use:   "sync",
	Short: "Sync branch checkouts and commits from git reflog",
	StrFlags: []StringFlag{
		{Name: "project", Shorthand: "p", Usage: "project name or ID (auto-detected from repo if omitted)"},
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		homeDir, repoDir, err := getContextPaths()
		if err != nil {
			return err
		}

		projectFlag, _ := cmd.Flags().GetString("project")

		return runSync(cmd, homeDir, repoDir, projectFlag, defaultGitReflog)
	},
}.Build()

// commitHashPattern matches full or abbreviated commit hashes (7-40 hex chars).
var commitHashPattern = regexp.MustCompile(`^[0-9a-f]{7,40}$`)

func looksLikeCommitHash(name string) bool {
	return commitHashPattern.MatchString(name)
}

// resolveCommitBranch determines which branch a commit belongs to by finding
// the last checkout before the commit's timestamp. Returns the checkout's Next
// field (the branch that was active at that time).
func resolveCommitBranch(commitTime time.Time, checkoutRecords []reflog.CheckoutRecord) string {
	// checkoutRecords must be sorted chronologically (oldest first)
	branch := ""
	for _, rec := range checkoutRecords {
		if rec.Timestamp.After(commitTime) {
			break
		}
		branch = rec.Next
	}
	return branch
}

func runSync(
	cmd *cobra.Command,
	homeDir, repoDir, projectFlag string,
	gitReflog GitReflogFunc,
) error {
	proj, err := ResolveProjectContext(homeDir, repoDir, projectFlag)
	if err != nil {
		return err
	}

	// Read repo config to get LastSync
	repoCfg, err := project.ReadRepoConfig(repoDir)
	if err != nil {
		return err
	}

	var lastSync *time.Time
	if repoCfg != nil {
		lastSync = repoCfg.LastSync
	}

	// Get reflog output
	output, err := gitReflog(repoDir, lastSync)
	if err != nil {
		return fmt.Errorf("failed to read git reflog: %w", err)
	}

	// Parse reflog for checkouts and commits
	records := reflog.ParseReflog(output)
	commitRecords := reflog.ParseCommits(output)

	// Build known IDs set from existing checkout and commit entries
	existingCheckouts, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}
	existingCommits, err := entry.ReadAllCommitEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}
	knownIDs := make(map[string]bool, len(existingCheckouts)+len(existingCommits))
	for _, e := range existingCheckouts {
		knownIDs[e.ID] = true
	}
	for _, e := range existingCommits {
		knownIDs[e.ID] = true
	}

	// Build sorted checkout records (oldest first) for branch resolution
	sortedCheckouts := make([]reflog.CheckoutRecord, len(records))
	copy(sortedCheckouts, records)
	sort.Slice(sortedCheckouts, func(i, j int) bool {
		return sortedCheckouts[i].Timestamp.Before(sortedCheckouts[j].Timestamp)
	})

	// Process checkout records oldest-first
	var createdCheckouts int
	var newestTimestamp time.Time
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]

		// Skip detached HEAD (branch name looks like a commit hash)
		if looksLikeCommitHash(rec.Previous) || looksLikeCommitHash(rec.Next) {
			continue
		}

		// Skip remote refs
		if strings.Contains(rec.Previous, "remotes/") || strings.Contains(rec.Next, "remotes/") {
			continue
		}

		// Skip same-branch
		if rec.Previous == rec.Next {
			continue
		}

		// Generate deterministic ID
		seed := rec.CommitRef + rec.Timestamp.Format(time.RFC3339) + rec.Previous + rec.Next
		id := hashutil.GenerateIDFromSeed(seed)

		// Skip already-synced entries (dedup by ID)
		if knownIDs[id] {
			continue
		}

		e := entry.CheckoutEntry{
			ID:        id,
			Timestamp: rec.Timestamp,
			Previous:  rec.Previous,
			Next:      rec.Next,
			CommitRef: rec.CommitRef,
			Repo:      repoDir,
		}

		if err := entry.WriteCheckoutEntry(homeDir, proj.Slug, e); err != nil {
			return err
		}

		knownIDs[id] = true
		createdCheckouts++

		if rec.Timestamp.After(newestTimestamp) {
			newestTimestamp = rec.Timestamp
		}
	}

	// Process commit records oldest-first
	var createdCommits int
	for i := len(commitRecords) - 1; i >= 0; i-- {
		rec := commitRecords[i]

		// Generate deterministic ID
		seed := rec.CommitRef + rec.Timestamp.Format(time.RFC3339) + "commit"
		id := hashutil.GenerateIDFromSeed(seed)

		// Skip already-synced entries (dedup by ID)
		if knownIDs[id] {
			continue
		}

		branch := resolveCommitBranch(rec.Timestamp, sortedCheckouts)

		e := entry.CommitEntry{
			ID:        id,
			Timestamp: rec.Timestamp,
			Message:   rec.Message,
			CommitRef: rec.CommitRef,
			Branch:    branch,
			Repo:      repoDir,
		}

		if err := entry.WriteCommitEntry(homeDir, proj.Slug, e); err != nil {
			return err
		}

		knownIDs[id] = true
		createdCommits++

		if rec.Timestamp.After(newestTimestamp) {
			newestTimestamp = rec.Timestamp
		}
	}

	totalCreated := createdCheckouts + createdCommits

	// Update LastSync to the newest processed record's timestamp
	if totalCreated > 0 && repoCfg != nil && !newestTimestamp.IsZero() {
		repoCfg.LastSync = &newestTimestamp
		if err := project.WriteRepoConfig(repoDir, repoCfg); err != nil {
			return err
		}
	}

	if totalCreated == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("already up to date"))
	} else {
		parts := []string{}
		if createdCheckouts > 0 {
			parts = append(parts, fmt.Sprintf("%d checkout(s)", createdCheckouts))
		}
		if createdCommits > 0 {
			parts = append(parts, fmt.Sprintf("%d commit(s)", createdCommits))
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			Text(fmt.Sprintf("synced %s for project '%s'", strings.Join(parts, " and "), Primary(proj.Name))))
	}

	return nil
}
