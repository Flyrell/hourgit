package hashutil

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

var hexPattern = regexp.MustCompile(`^[0-9a-f]{7}$`)

func TestGenerateIDFormat(t *testing.T) {
	id := GenerateID("test-project")
	assert.Regexp(t, hexPattern, id)
}

func TestGenerateIDUniqueness(t *testing.T) {
	id1 := GenerateID("test-project")
	id2 := GenerateID("test-project")
	assert.NotEqual(t, id1, id2, "IDs from successive calls should differ due to timestamp")
}

func TestGenerateIDFromSeedDeterministic(t *testing.T) {
	id1 := GenerateIDFromSeed("fixed-seed")
	id2 := GenerateIDFromSeed("fixed-seed")
	assert.Equal(t, id1, id2)
}

func TestGenerateIDFromSeedFormat(t *testing.T) {
	id := GenerateIDFromSeed("any-seed-value")
	assert.Regexp(t, hexPattern, id)
}

func TestGenerateIDFromSeedDifferentInputs(t *testing.T) {
	id1 := GenerateIDFromSeed("seed-a")
	id2 := GenerateIDFromSeed("seed-b")
	assert.NotEqual(t, id1, id2)
}
