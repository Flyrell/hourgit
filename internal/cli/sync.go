package cli

import (
	"fmt"
	"os/exec"
	"regexp"
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
	Short: "Sync branch checkouts from git reflog",
	StrFlags: []StringFlag{
		{Name: "project", Usage: "project name or ID (auto-detected from repo if omitted)"},
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

	// Parse reflog
	records := reflog.ParseReflog(output)

	// Build known IDs set from existing checkout entries
	existingEntries, err := entry.ReadAllCheckoutEntries(homeDir, proj.Slug)
	if err != nil {
		return err
	}
	knownIDs := make(map[string]bool, len(existingEntries))
	for _, e := range existingEntries {
		knownIDs[e.ID] = true
	}

	// Process records oldest-first
	var created int
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
		}

		if err := entry.WriteCheckoutEntry(homeDir, proj.Slug, e); err != nil {
			return err
		}

		knownIDs[id] = true
		created++

		if rec.Timestamp.After(newestTimestamp) {
			newestTimestamp = rec.Timestamp
		}
	}

	// Update LastSync to the newest processed record's timestamp
	if created > 0 && repoCfg != nil && !newestTimestamp.IsZero() {
		repoCfg.LastSync = &newestTimestamp
		if err := project.WriteRepoConfig(repoDir, repoCfg); err != nil {
			return err
		}
	}

	if created == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), Text("already up to date"))
	} else {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s\n",
			Text(fmt.Sprintf("synced %d checkout(s) for project '%s'", created, Primary(proj.Name))))
	}

	return nil
}
