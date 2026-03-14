package timetrack

import (
	"time"

	"github.com/Flyrell/hourgit/internal/entry"
)

// deductLogOverlaps removes manual log time ranges from checkout segments,
// similar to how idle gaps are carved out. Each log's [Start, Start+Minutes)
// range is treated as a gap to remove from overlapping segments.
// Checkout-generated entries are skipped (they are already materialized logs).
func deductLogOverlaps(
	segments []sessionSegment,
	logs []entry.Entry,
	year int, month time.Month,
	loc *time.Location,
) []sessionSegment {
	// Collect log ranges for the target month
	var gaps []idleGap
	for _, l := range logs {
		logStart := l.Start.In(loc)
		if logStart.Year() != year || logStart.Month() != month {
			continue
		}
		// Skip checkout-generated entries — they don't reduce checkout time
		if l.Source == "checkout-generated" {
			continue
		}
		logEnd := logStart.Add(time.Duration(l.Minutes) * time.Minute)
		gaps = append(gaps, idleGap{
			stop:  logStart,
			start: logEnd,
		})
	}

	if len(gaps) == 0 {
		return segments
	}

	// Reuse the same gap-splitting logic as idle trimming
	var result []sessionSegment
	for _, seg := range segments {
		trimmed := applyGapsToSegment(seg, gaps)
		result = append(result, trimmed...)
	}
	return result
}
