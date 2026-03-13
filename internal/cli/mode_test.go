package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateModeEmpty(t *testing.T) {
	err := validateMode("")
	assert.NoError(t, err)
}

func TestValidateModeStandard(t *testing.T) {
	err := validateMode("standard")
	assert.NoError(t, err)
}

func TestValidateModePrecise(t *testing.T) {
	err := validateMode("precise")
	assert.NoError(t, err)
}

func TestValidateModeInvalid(t *testing.T) {
	err := validateMode("foobar")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --mode value")
}
