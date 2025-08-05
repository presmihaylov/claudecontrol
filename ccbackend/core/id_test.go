package core

import (
	"regexp"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewID_ValidPrefix(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:     "simple prefix",
			prefix:   "si",
			expected: "si",
		},
		{
			name:     "uppercase prefix gets lowercased",
			prefix:   "SI",
			expected: "si",
		},
		{
			name:     "mixed case prefix gets lowercased",
			prefix:   "SlackIntegration",
			expected: "slackintegration",
		},
		{
			name:     "prefix with leading/trailing spaces gets trimmed",
			prefix:   "  si  ",
			expected: "si",
		},
		{
			name:     "single character prefix",
			prefix:   "u",
			expected: "u",
		},
		{
			name:     "longer prefix",
			prefix:   "agent_job_assignment",
			expected: "agent_job_assignment",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			id := NewID(tc.prefix)

			// Check format: prefix_ULID
			parts := strings.Split(id, "_")
			require.Len(t, parts, 2, "ID should have exactly one underscore separating prefix and ULID")

			// Check prefix is correct
			assert.Equal(t, tc.expected, parts[0], "Prefix should be cleaned correctly")

			// Check ULID part is valid
			ulidPart := parts[1]
			assert.Len(t, ulidPart, 26, "ULID should be 26 characters long")

			// Verify it's a valid ULID format (base32 encoded)
			ulidRegex := regexp.MustCompile("^[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$")
			assert.True(t, ulidRegex.MatchString(ulidPart), "ULID part should match base32 format")

			// Verify we can parse it as a ULID
			_, err := ulid.Parse(ulidPart)
			assert.NoError(t, err, "ULID part should be parseable as valid ULID")
		})
	}
}

func TestNewID_EmptyPrefix_Panics(t *testing.T) {
	testCases := []struct {
		name   string
		prefix string
	}{
		{
			name:   "empty string",
			prefix: "",
		},
		{
			name:   "only spaces",
			prefix: "   ",
		},
		{
			name:   "only tabs",
			prefix: "\t\t",
		},
		{
			name:   "mixed whitespace",
			prefix: " \t \n ",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Panics(t, func() {
				NewID(tc.prefix)
			}, "Should panic with empty or whitespace-only prefix")
		})
	}
}

func TestNewID_Uniqueness(t *testing.T) {
	// Generate multiple IDs with the same prefix and verify they're unique
	prefix := "test"
	numIDs := 1000
	ids := make(map[string]bool)

	for i := 0; i < numIDs; i++ {
		id := NewID(prefix)
		
		// Check that this ID hasn't been generated before
		assert.False(t, ids[id], "Generated ID should be unique: %s", id)
		ids[id] = true

		// Also verify format for each ID
		parts := strings.Split(id, "_")
		require.Len(t, parts, 2)
		assert.Equal(t, prefix, parts[0])
		assert.Len(t, parts[1], 26)
	}

	assert.Len(t, ids, numIDs, "Should have generated exactly %d unique IDs", numIDs)
}

func TestNewID_FormatExample(t *testing.T) {
	// Test the specific format mentioned in the requirements
	id := NewID("si")
	
	// Should match pattern like: si_01G0EZ1XTM37C5X11SQTDNCTM1
	pattern := regexp.MustCompile("^si_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$")
	assert.True(t, pattern.MatchString(id), "ID should match the required format: %s", id)
	
	// Verify the prefix part
	assert.True(t, strings.HasPrefix(id, "si_"), "ID should start with 'si_'")
	
	// Verify total length (prefix + underscore + ULID)
	expectedLength := len("si") + 1 + 26 // prefix + underscore + ULID
	assert.Len(t, id, expectedLength, "ID should have correct total length")
}

func TestNewID_ULIDProperties(t *testing.T) {
	// Test that generated ULIDs have proper timestamp ordering
	id1 := NewID("test")
	id2 := NewID("test")
	
	// Extract ULID parts
	ulid1 := strings.Split(id1, "_")[1]
	ulid2 := strings.Split(id2, "_")[1]
	
	// Parse ULIDs to verify they're valid and get timestamps
	parsed1, err := ulid.Parse(ulid1)
	require.NoError(t, err)
	
	parsed2, err := ulid.Parse(ulid2)
	require.NoError(t, err)
	
	// ULIDs should be lexicographically sortable (second should be >= first)
	assert.True(t, ulid2 >= ulid1, "ULIDs should be lexicographically sortable")
	
	// Timestamps should be reasonable (generated around the same time)
	time1 := parsed1.Time()
	time2 := parsed2.Time()
	
	// They should be generated within a reasonable time window (1 second)
	timeDiff := time2 - time1
	assert.True(t, timeDiff >= 0, "Second ULID should have timestamp >= first")
	assert.True(t, timeDiff < 1000, "ULIDs should be generated within 1 second of each other")
}