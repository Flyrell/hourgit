package entry

import "time"

// CheckoutEntry represents a branch checkout event recorded by the post-checkout hook.
type CheckoutEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Previous  string    `json:"previous"`
	Next      string    `json:"next"`
	CommitRef string    `json:"commit_ref,omitempty"`
}
