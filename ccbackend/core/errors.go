package core

import (
	"errors"
	"regexp"
)

// ErrNotFound is a sentinel error for "not found" cases
var ErrNotFound = errors.New("not found")

// IsNotFoundError checks if an error is a "not found" error
// This function handles both the new ErrNotFound sentinel error and legacy string-based errors
func IsNotFoundError(err error) bool {
	if err == nil {
		return false
	}
	// Check for the new sentinel error
	if errors.Is(err, ErrNotFound) {
		return true
	}
	// Check for legacy string-based errors for backward compatibility
	return containsNotFound(err.Error())
}

// containsNotFound checks if an error message contains "not found"
func containsNotFound(errMsg string) bool {
	// Use case-insensitive matching for various "not found" formats
	return len(errMsg) > 0 && (regexp.MustCompile(`(?i)not found`).MatchString(errMsg))
}
