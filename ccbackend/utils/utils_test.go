package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertMarkdownToSlack(t *testing.T) {
	t.Run("ConvertBoldMarkdown", func(t *testing.T) {
		tests := []struct {
			name     string
			input    string
			expected string
		}{
			{
				name:     "Single bold word",
				input:    "This is **bold** text",
				expected: "This is *bold* text",
			},
			{
				name:     "Multiple bold words",
				input:    "This is **bold** and this is **also bold**",
				expected: "This is *bold* and this is *also bold*",
			},
			{
				name:     "Bold phrase with spaces",
				input:    "This is **bold phrase** text",
				expected: "This is *bold phrase* text",
			},
			{
				name:     "No bold markdown",
				input:    "This is regular text",
				expected: "This is regular text",
			},
			{
				name:     "Empty string",
				input:    "",
				expected: "",
			},
			{
				name:     "Only bold text",
				input:    "**completely bold**",
				expected: "*completely bold*",
			},
			{
				name:     "Bold with special characters",
				input:    "**bold with !@#$%^&*() characters**",
				expected: "*bold with !@#$%^&*() characters*",
			},
			{
				name:     "Multiple lines with bold",
				input:    "First line with **bold**\nSecond line with **more bold**",
				expected: "First line with *bold*\nSecond line with *more bold*",
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := ConvertMarkdownToSlack(tt.input)
				assert.Equal(t, tt.expected, result)
			})
		}
	})
}

func TestAssertInvariant(t *testing.T) {
	t.Run("TrueCondition", func(t *testing.T) {
		// Should not panic
		assert.NotPanics(t, func() {
			AssertInvariant(true, "This should not panic")
		})
	})

	t.Run("FalseCondition", func(t *testing.T) {
		// Should panic with the correct message
		assert.PanicsWithValue(t, "invariant violated - This should panic", func() {
			AssertInvariant(false, "This should panic")
		})
	})

	t.Run("ComplexCondition", func(t *testing.T) {
		x := 5
		y := 10
		
		// Should not panic
		assert.NotPanics(t, func() {
			AssertInvariant(x < y, "x should be less than y")
		})

		// Should panic
		assert.PanicsWithValue(t, "invariant violated - x should be greater than y", func() {
			AssertInvariant(x > y, "x should be greater than y")
		})
	})
}