//go:build darwin

package watch

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlistContent(t *testing.T) {
	content := PlistContent("/usr/local/bin/hourgit")

	assert.Contains(t, content, "com.hourgit.watch")
	assert.Contains(t, content, "/usr/local/bin/hourgit")
	assert.Contains(t, content, "<string>watch</string>")
	assert.Contains(t, content, "<key>KeepAlive</key>")
	assert.Contains(t, content, "<key>RunAtLoad</key>")
	assert.True(t, strings.HasPrefix(content, "<?xml"))
}
