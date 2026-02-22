package hashutil

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// GenerateID creates a 7-character hex ID from a name and the current timestamp.
func GenerateID(name string) string {
	seed := name + "\x00" + fmt.Sprintf("%d", time.Now().UnixNano())
	return GenerateIDFromSeed(seed)
}

// GenerateIDFromSeed creates a deterministic 7-character hex ID from a seed string.
func GenerateIDFromSeed(seed string) string {
	hash := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("%x", hash[:4])[:7]
}
