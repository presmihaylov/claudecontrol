package core

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "valid prefix",
			prefix: "u",
		},
		{
			name:   "valid multi-character prefix",
			prefix: "si",
		},
		{
			name:   "uppercase prefix gets lowercased",
			prefix: "USER",
		},
		{
			name:   "prefix with spaces gets trimmed",
			prefix: "  job  ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewID(tt.prefix)

			// Check the format: prefix_ULID
			expectedPrefix := strings.ToLower(strings.TrimSpace(tt.prefix)) + "_"
			if !strings.HasPrefix(got, expectedPrefix) {
				t.Errorf("NewID() = %v, want prefix %v", got, expectedPrefix)
			}

			// Check ULID pattern: 26 characters, base32 encoded
			ulidPart := strings.TrimPrefix(got, expectedPrefix)
			if len(ulidPart) != 26 {
				t.Errorf("NewID() ULID part length = %v, want 26", len(ulidPart))
			}

			// Check ULID format using regex (base32 characters: 0-9, A-Z excluding I, L, O, U)
			ulidPattern := regexp.MustCompile(`^[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
			if !ulidPattern.MatchString(ulidPart) {
				t.Errorf("NewID() ULID part %v does not match expected pattern", ulidPart)
			}

			// Verify format matches the example pattern: w_01G0EZ1XTM37C5X11SQTDNCTM1
			fullPattern := regexp.MustCompile(`^[a-z0-9]+_[0123456789ABCDEFGHJKMNPQRSTVWXYZ]{26}$`)
			if !fullPattern.MatchString(got) {
				t.Errorf("NewID() = %v does not match expected format pattern", got)
			}
		})
	}
}

func TestNewIDPanic(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
	}{
		{
			name:   "empty prefix panics",
			prefix: "",
		},
		{
			name:   "whitespace-only prefix panics",
			prefix: "   ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					t.Errorf("NewID() expected panic but got none")
				}
			}()

			NewID(tt.prefix)
		})
	}
}

func TestNewIDUniqueness(t *testing.T) {
	prefix := "test"
	ids := make(map[string]bool)
	numTests := 1000

	for i := 0; i < numTests; i++ {
		id := NewID(prefix)

		if ids[id] {
			t.Errorf("NewID() generated duplicate ID: %v", id)
		}
		ids[id] = true
	}

	if len(ids) != numTests {
		t.Errorf("Expected %d unique IDs, got %d", numTests, len(ids))
	}
}
