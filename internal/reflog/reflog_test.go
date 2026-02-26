package reflog

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestParseReflogStandardCheckout(t *testing.T) {
	input := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: checkout: moving from main to feature-x`

	records := ParseReflog(input)

	assert.Len(t, records, 1)
	assert.Equal(t, "abc1234", records[0].CommitRef)
	assert.Equal(t, time.Date(2025, 6, 15, 14, 30, 0, 0, time.UTC), records[0].Timestamp)
	assert.Equal(t, "main", records[0].Previous)
	assert.Equal(t, "feature-x", records[0].Next)
}

func TestParseReflogBranchNamesWithSlashes(t *testing.T) {
	input := `def5678 HEAD@{2025-06-15 10:00:00 +0200}: checkout: moving from feature/ENG-641/item to release/v2.0`

	records := ParseReflog(input)

	assert.Len(t, records, 1)
	assert.Equal(t, "feature/ENG-641/item", records[0].Previous)
	assert.Equal(t, "release/v2.0", records[0].Next)
}

func TestParseReflogSkipsNonCheckoutLines(t *testing.T) {
	input := `abc1234 HEAD@{2025-06-15 14:30:00 +0000}: commit: fix bug
def5678 HEAD@{2025-06-15 14:00:00 +0000}: checkout: moving from main to develop
ghi9012 HEAD@{2025-06-15 13:30:00 +0000}: rebase: checkout feature
jkl3456 HEAD@{2025-06-15 13:00:00 +0000}: pull: merge
mno7890 HEAD@{2025-06-15 12:00:00 +0000}: cherry-pick: fix`

	records := ParseReflog(input)

	assert.Len(t, records, 1)
	assert.Equal(t, "def5678", records[0].CommitRef)
	assert.Equal(t, "main", records[0].Previous)
	assert.Equal(t, "develop", records[0].Next)
}

func TestParseReflogEmptyInput(t *testing.T) {
	records := ParseReflog("")
	assert.Empty(t, records)
}

func TestParseReflogMalformedTimestamp(t *testing.T) {
	input := `abc1234 HEAD@{not-a-date}: checkout: moving from main to feature`

	records := ParseReflog(input)

	assert.Empty(t, records)
}

func TestParseReflogMultipleCheckouts(t *testing.T) {
	input := `abc1234 HEAD@{2025-06-15 16:00:00 +0000}: checkout: moving from develop to main
def5678 HEAD@{2025-06-15 14:00:00 +0000}: checkout: moving from main to develop`

	records := ParseReflog(input)

	assert.Len(t, records, 2)
	// Newest first (reflog order)
	assert.Equal(t, "abc1234", records[0].CommitRef)
	assert.Equal(t, "develop", records[0].Previous)
	assert.Equal(t, "main", records[0].Next)
	assert.Equal(t, "def5678", records[1].CommitRef)
	assert.Equal(t, "main", records[1].Previous)
	assert.Equal(t, "develop", records[1].Next)
}

func TestParseReflogTimezoneConversion(t *testing.T) {
	// +0200 means 14:30 local = 12:30 UTC
	input := `abc1234 HEAD@{2025-06-15 14:30:00 +0200}: checkout: moving from main to feature`

	records := ParseReflog(input)

	assert.Len(t, records, 1)
	assert.Equal(t, time.Date(2025, 6, 15, 12, 30, 0, 0, time.UTC), records[0].Timestamp)
}
