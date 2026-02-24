package entry

const TypeGeneratedDay = "generated_day"

// GeneratedDayEntry marks a day as "generated" — checkout attribution is skipped for that day.
type GeneratedDayEntry struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Date string `json:"date"` // "2006-01-02" format
}
