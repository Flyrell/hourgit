package entry

import "time"

// CommitEntry represents a git commit event captured from reflog.
type CommitEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	CommitRef string    `json:"commit_ref"`
	Branch    string    `json:"branch"`
	Repo      string    `json:"repo,omitempty"`
}
