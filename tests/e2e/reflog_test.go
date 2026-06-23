package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// ReflogBuilder constructs simulated git reflog entries that can be written
// directly to .git/logs/HEAD. Git validates that SHAs reference real objects,
// so the builder uses SHAs from the repo's pre-created seed commits.
type ReflogBuilder struct {
	entries  []rawReflogEntry
	repo     *TestRepo
	shaIndex int // cycles through repo.SHAs
}

type rawReflogEntry struct {
	oldSHA    string
	newSHA    string
	timestamp time.Time
	action    string // e.g. "checkout: moving from master to feature" or "commit: add feature"
}

// NewReflogBuilder creates a new builder for simulated reflog entries.
// The repo must have been created with AddRepo (which pre-creates seed commits with valid SHAs).
func NewReflogBuilder(repo *TestRepo) *ReflogBuilder {
	return &ReflogBuilder{
		repo:     repo,
		shaIndex: 0,
	}
}

// nextSHA returns the next available real SHA from the repo's seed commits.
func (b *ReflogBuilder) nextSHA() string {
	sha := b.repo.SHAs[b.shaIndex%len(b.repo.SHAs)]
	b.shaIndex++
	return sha
}

// Checkout adds a simulated branch checkout to the reflog.
func (b *ReflogBuilder) Checkout(from, to string, at time.Time) *ReflogBuilder {
	oldSHA := b.nextSHA()
	newSHA := b.nextSHA()

	b.entries = append(b.entries, rawReflogEntry{
		oldSHA:    oldSHA,
		newSHA:    newSHA,
		timestamp: at,
		action:    fmt.Sprintf("checkout: moving from %s to %s", from, to),
	})
	return b
}

// Commit adds a simulated commit entry to the reflog.
func (b *ReflogBuilder) Commit(message string, at time.Time) *ReflogBuilder {
	oldSHA := b.nextSHA()
	newSHA := b.nextSHA()

	b.entries = append(b.entries, rawReflogEntry{
		oldSHA:    oldSHA,
		newSHA:    newSHA,
		timestamp: at,
		action:    fmt.Sprintf("commit: %s", message),
	})
	return b
}

// WriteTo writes the simulated reflog entries to the repo's .git/logs/HEAD file.
// Entries are appended after any existing content (e.g., the seed commits from AddRepo).
func (b *ReflogBuilder) WriteTo(t *testing.T) {
	t.Helper()

	logsDir := filepath.Join(b.repo.Dir, ".git", "logs")
	require.NoError(t, os.MkdirAll(logsDir, 0755))

	headLog := filepath.Join(logsDir, "HEAD")

	f, err := os.OpenFile(headLog, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	require.NoError(t, err)
	defer func() { _ = f.Close() }()

	for _, e := range b.entries {
		// Raw reflog format:
		// old_sha new_sha Author <email> unix_timestamp timezone\taction
		line := fmt.Sprintf("%s %s E2E Test <e2e@test.com> %d +0000\t%s\n",
			e.oldSHA, e.newSHA, e.timestamp.Unix(), e.action)
		_, err := f.WriteString(line)
		require.NoError(t, err)
	}
}
