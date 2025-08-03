package utils

import (
	"testing"
)

func TestAssertInvariant(t *testing.T) {
	t.Run("TrueCondition", func(t *testing.T) {
		// Should not panic
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("AssertInvariant(true, message) panicked: %v", r)
			}
		}()
		AssertInvariant(true, "This should not panic")
	})

	t.Run("FalseCondition", func(t *testing.T) {
		// Should panic with the correct message
		defer func() {
			if r := recover(); r != nil {
				expected := "invariant violated - This should panic"
				if r != expected {
					t.Errorf("AssertInvariant(false, message) panicked with %v, expected %v", r, expected)
				}
			} else {
				t.Error("AssertInvariant(false, message) did not panic")
			}
		}()
		AssertInvariant(false, "This should panic")
	})
}