package entry

import "time"

// ActivityStopEntry is written after idle_threshold_minutes of no file changes.
// Timestamp records the last observed file change, not when the debounce fired.
type ActivityStopEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Repo      string    `json:"repo,omitempty"`
}

// ActivityStartEntry is written when file changes resume after a stop.
type ActivityStartEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	Repo      string    `json:"repo,omitempty"`
}
