package core

import (
	"regexp"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID(t *testing.T) {
	t.Run("generates valid ULID with prefix", func(t *testing.T) {
		prefix := "si"
		id := NewID(prefix)

		// Should match pattern: prefix_ULID
		parts := strings.Split(id, "_")
		require.Len(t, parts, 2, "ID should have exactly one underscore separating prefix and ULID")

		assert.Equal(t, prefix, parts[0], "First part should be the prefix")

		// Validate the ULID part
		ulidPart := parts[1]
		_, err := ulid.Parse(ulidPart)
		assert.NoError(t, err, "ULID part should be valid")

		// ULID should be 26 characters (Crockford Base32)
		assert.Len(t, ulidPart, 26, "ULID should be 26 characters long")

		// Should match ULID pattern (Base32 alphabet)
		ulidPattern := regexp.MustCompile(`^[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
		assert.True(t, ulidPattern.MatchString(ulidPart), "ULID should match Base32 pattern")
	})

	t.Run("converts prefix to lowercase", func(t *testing.T) {
		id := NewID("SI")
		assert.True(t, strings.HasPrefix(id, "si_"), "Prefix should be converted to lowercase")
	})

	t.Run("trims whitespace from prefix", func(t *testing.T) {
		id := NewID("  si  ")
		assert.True(t, strings.HasPrefix(id, "si_"), "Whitespace should be trimmed from prefix")
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		id1 := NewID("test")
		id2 := NewID("test")
		assert.NotEqual(t, id1, id2, "Each generated ID should be unique")
	})

	t.Run("works with different prefixes", func(t *testing.T) {
		prefixes := []string{"u", "si", "aa", "j", "aja", "psm"}
		
		for _, prefix := range prefixes {
			id := NewID(prefix)
			assert.True(t, strings.HasPrefix(id, prefix+"_"), "ID should start with prefix and underscore")
			
			parts := strings.Split(id, "_")
			require.Len(t, parts, 2)
			
			_, err := ulid.Parse(parts[1])
			assert.NoError(t, err, "ULID part should be valid for prefix: %s", prefix)
		}
	})

	t.Run("panics with empty prefix", func(t *testing.T) {
		assert.Panics(t, func() {
			NewID("")
		}, "Should panic with empty prefix")
	})

	t.Run("panics with whitespace-only prefix", func(t *testing.T) {
		assert.Panics(t, func() {
			NewID("   ")
		}, "Should panic with whitespace-only prefix")
	})

	t.Run("conforms to expected pattern like w_01G0EZ1XTM37C5X11SQTDNCTM1", func(t *testing.T) {
		id := NewID("w")
		
		// Should match pattern: w_XXXXXXXXXXXXXXXXXXXXXXXXXX (w + underscore + 26 char ULID)
		expectedPattern := regexp.MustCompile(`^w_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
		assert.True(t, expectedPattern.MatchString(id), "ID should match the expected pattern w_XXXXXXXXXXXXXXXXXXXXXXXXXX")
		
		assert.Len(t, id, 28, "Total ID length should be 28 characters (1 char prefix + 1 underscore + 26 char ULID)")
	})
}