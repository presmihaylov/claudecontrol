package core

import (
	"crypto/rand"
	"errors"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// NewID generates a new ULID with the given prefix.
// The format is: prefix_ULID
// Example: core.NewID("u") returns "u_01G0EZ1XTM37C5X11SQTDNCTM1"
func NewID(prefix string) (string, error) {
	if prefix == "" || strings.TrimSpace(prefix) == "" {
		return "", errors.New("prefix cannot be empty")
	}

	// Generate a new ULID with current timestamp and crypto/rand entropy
	entropy := ulid.Monotonic(rand.Reader, 0)
	id := ulid.MustNew(ulid.Timestamp(time.Now()), entropy)

	// Return formatted ID with lowercase prefix
	return strings.ToLower(strings.TrimSpace(prefix)) + "_" + id.String(), nil
}