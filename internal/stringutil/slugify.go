package stringutil

import (
	"regexp"
	"strings"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9]+`)

// Slugify converts a string to a URL-friendly slug.
// It lowercases the input, replaces non-alphanumeric characters with hyphens,
// collapses consecutive hyphens, and trims leading/trailing hyphens.
func Slugify(name string) string {
	s := strings.ToLower(name)
	s = nonAlphanumeric.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}
