//go:build linux

package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestServiceFileContent(t *testing.T) {
	content := ServiceFileContent("/usr/local/bin/hourgit")

	assert.Contains(t, content, "ExecStart=/usr/local/bin/hourgit watch")
	assert.Contains(t, content, "Restart=always")
	assert.Contains(t, content, "[Unit]")
	assert.Contains(t, content, "[Service]")
	assert.Contains(t, content, "[Install]")
}
