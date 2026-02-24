package entry

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntryJSONRoundTrip(t *testing.T) {
	e := Entry{
		ID:        "abc1234",
		Start:     time.Date(2025, 6, 15, 9, 0, 0, 0, time.UTC),
		Minutes:   180,
		Message:   "morning work",
		Task:      "feature-branch",
		CreatedAt: time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC),
	}

	data, err := json.Marshal(e)
	require.NoError(t, err)

	var decoded Entry
	require.NoError(t, json.Unmarshal(data, &decoded))

	assert.Equal(t, e.ID, decoded.ID)
	assert.Equal(t, e.Minutes, decoded.Minutes)
	assert.Equal(t, e.Message, decoded.Message)
	assert.Equal(t, e.Task, decoded.Task)
	assert.True(t, e.Start.Equal(decoded.Start))
	assert.True(t, e.CreatedAt.Equal(decoded.CreatedAt))
}

func TestEntryJSONOmitsEmptyTask(t *testing.T) {
	e := Entry{
		ID:      "abc1234",
		Minutes: 60,
		Message: "no task",
	}

	data, err := json.Marshal(e)
	require.NoError(t, err)

	assert.NotContains(t, string(data), `"task"`)
}
