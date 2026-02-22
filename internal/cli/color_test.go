package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestColorHelpers(t *testing.T) {
	tests := []struct {
		name   string
		fn     func(string) string
		input  string
	}{
		{"Primary", Primary, "hello"},
		{"Error", Error, "something failed"},
		{"Warning", Warning, "be careful"},
		{"Info", Info, "note this"},
		{"Silent", Silent, "quiet text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fn(tt.input)
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.input)
		})
	}
}
