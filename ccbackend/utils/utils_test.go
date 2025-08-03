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
			{
				name:     "Heading level 1",
				input:    "# Heading 1",
				expected: "*Heading 1*",
			},
			{
				name:     "Heading level 2",
				input:    "## Heading 2",
				expected: "*Heading 2*",
			},
			{
				name:     "Heading level 3",
				input:    "### Heading 3",
				expected: "*Heading 3*",
			},
			{
				name:     "Multiple headings",
				input:    "# First Heading\nSome text\n## Second Heading",
				expected: "*First Heading*\nSome text\n*Second Heading*",
			},
			{
				name:     "Heading without space after #",
				input:    "#NoSpace",
				expected: "*NoSpace*",
			},
			{
				name:     "Heading with extra spaces",
				input:    "##   Lots of spaces",
				expected: "*Lots of spaces*",
			},
			{
				name:     "Mixed bold and headings",
				input:    "# Main Title\nThis has **bold text** in it\n## Subtitle",
				expected: "*Main Title*\nThis has *bold text* in it\n*Subtitle*",
			},
			{
				name:     "Hashtag in middle of line (not heading)",
				input:    "This is not # a heading",
				expected: "This is not # a heading",
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