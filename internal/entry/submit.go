package entry

import "time"

// SubmitEntry marks a date range as submitted.
type SubmitEntry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	From      time.Time `json:"from"`
	To        time.Time `json:"to"`
	CreatedAt time.Time `json:"created_at"`
}
