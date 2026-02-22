package stringutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlugify(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple lowercase", "hello", "hello"},
		{"mixed case", "My Project", "my-project"},
		{"special characters", "foo@bar!baz", "foo-bar-baz"},
		{"consecutive specials", "foo---bar", "foo-bar"},
		{"leading trailing specials", "---foo---", "foo"},
		{"numbers preserved", "project123", "project123"},
		{"spaces replaced", "my cool project", "my-cool-project"},
		{"mixed specials", "Hello, World! (2024)", "hello-world-2024"},
		{"empty string", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Slugify(tt.input))
		})
	}
}
