package entry

import "time"

const (
	TypeLog           = "log"
	TypeCheckout      = "checkout"
	TypeSubmit        = "submit"
	TypeCommit        = "commit"
	TypeActivityStop  = "activity_stop"
	TypeActivityStart = "activity_start"
)

// Entry represents a single time log entry (a "time commit").
type Entry struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Start     time.Time `json:"start"`
	Minutes   int       `json:"minutes"`
	Message   string    `json:"message"`
	Task      string    `json:"task,omitempty"`
	Source    string    `json:"source,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}
