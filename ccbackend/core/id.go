package core

import (
	"fmt"
	"strings"

	"github.com/oklog/ulid/v2"
)

// NewID generates a new ULID with the specified prefix.
// The resulting ID follows the format: prefix_ULID
// Example: NewID("si") returns "si_01G0EZ1XTM37C5X11SQTDNCTM1"
func NewID(prefix string) string {
	if prefix == "" || strings.TrimSpace(prefix) == "" {
		panic("Prefix cannot be empty")
	}

	cleanPrefix := strings.TrimSpace(strings.ToLower(prefix))
	ulid := ulid.Make()

	return fmt.Sprintf("%s_%s", cleanPrefix, ulid.String())
}