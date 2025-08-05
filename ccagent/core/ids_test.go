package core

import (
	"regexp"
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	tests := []struct {
		name    string
		prefix  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid prefix",
			prefix:  "u",
			wantErr: false,
		},
		{
			name:    "valid multi-character prefix",
			prefix:  "si",
			wantErr: false,
		},
		{
			name:    "uppercase prefix gets lowercased",
			prefix:  "USER",
			wantErr: false,
		},
		{
			name:    "prefix with spaces gets trimmed",
			prefix:  "  job  ",
			wantErr: false,
		},
		{
			name:    "empty prefix returns error",
			prefix:  "",
			wantErr: true,
			errMsg:  "prefix cannot be empty",
		},
		{
			name:    "whitespace-only prefix returns error",
			prefix:  "   ",
			wantErr: true,
			errMsg:  "prefix cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewID(tt.prefix)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewID() expected error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("NewID() error = %v, want error containing %v", err, tt.errMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("NewID() unexpected error = %v", err)
				return
			}

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

func TestNewIDUniqueness(t *testing.T) {
	prefix := "test"
	ids := make(map[string]bool)
	numTests := 1000

	for i := 0; i < numTests; i++ {
		id, err := NewID(prefix)
		if err != nil {
			t.Fatalf("NewID() unexpected error = %v", err)
		}

		if ids[id] {
			t.Errorf("NewID() generated duplicate ID: %v", id)
		}
		ids[id] = true
	}

	if len(ids) != numTests {
		t.Errorf("Expected %d unique IDs, got %d", numTests, len(ids))
	}
}

func TestNewIDExampleFormat(t *testing.T) {
	// Test that the output matches the example format from the user
	id, err := NewID("w")
	if err != nil {
		t.Fatalf("NewID() unexpected error = %v", err)
	}

	// Should match format: w_01G0EZ1XTM37C5X11SQTDNCTM1
	if !strings.HasPrefix(id, "w_") {
		t.Errorf("NewID() = %v, expected to start with 'w_'", id)
	}

	if len(id) != 28 { // w_ (2) + ULID (26) = 28
		t.Errorf("NewID() length = %v, expected 28 characters", len(id))
	}
}
