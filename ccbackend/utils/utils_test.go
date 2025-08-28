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
			{
				name:     "Already Slack formatted text (single asterisks)",
				input:    "*Everything working successfully*",
				expected: "*Everything working successfully*",
			},
			{
				name:     "Mixed Slack and Markdown formatting",
				input:    "*Already bold* and **needs conversion**",
				expected: "*Already bold* and *needs conversion*",
			},
			{
				name:     "Complex message with mixed formatting",
				input:    "Excellent! *Everything working successfully* :white_check_mark:\n\n*Summary*\n\nI've successfully completed all the requested tasks:\n\n*:white_check_mark: **DB Layer Updates*** - Modified all database layer components",
				expected: "Excellent! *Everything working successfully* :white_check_mark:\n\n*Summary*\n\nI've successfully completed all the requested tasks:\n\n*:white_check_mark: *DB Layer Updates** - Modified all database layer components",
			},
			{
				name:     "Heading with bold markdown inside",
				input:    "## 🧪 **GitUseCase Testing Implementation Complete**\n### **✅ Interface Extraction**\n- **GitClientInterface**: 23 methods covering all Git operations",
				expected: "*🧪 GitUseCase Testing Implementation Complete*\n*✅ Interface Extraction*\n- *GitClientInterface*: 23 methods covering all Git operations",
			},
			{
				name:     "Simple markdown link",
				input:    "Check out [GitHub](https://github.com)",
				expected: "Check out <https://github.com|GitHub>",
			},
			{
				name:     "Multiple markdown links",
				input:    "Visit [GitHub](https://github.com) and [Google](https://google.com)",
				expected: "Visit <https://github.com|GitHub> and <https://google.com|Google>",
			},
			{
				name:     "Markdown link with bold text",
				input:    "**Important**: Check [GitHub](https://github.com)",
				expected: "*Important*: Check <https://github.com|GitHub>",
			},
			{
				name:     "Bold text around markdown link",
				input:    "This is **[GitHub](https://github.com)** repository",
				expected: "This is *<https://github.com|GitHub>* repository",
			},
			{
				name:     "Complex case with numbered list, bold, and links",
				input:    "**1. [github.com/IBM/fp-go](http://github.com/IBM/fp-go)** - Full functional programming library",
				expected: "*1. <http://github.com/IBM/fp-go|github.com/IBM/fp-go>* - Full functional programming library",
			},
			{
				name:     "Link in heading",
				input:    "# Check [GitHub](https://github.com)",
				expected: "*Check <https://github.com|GitHub>*",
			},
			{
				name:     "Multiple links with bold and headings",
				input:    "## **Third-Party Libraries**\n**1. [github.com/IBM/fp-go](http://github.com/IBM/fp-go)** - Full functional programming library\n**2. [github.com/samber/mo](http://github.com/samber/mo)** - Modern functional utilities",
				expected: "*Third-Party Libraries*\n*1. <http://github.com/IBM/fp-go|github.com/IBM/fp-go>* - Full functional programming library\n*2. <http://github.com/samber/mo|github.com/samber/mo>* - Modern functional utilities",
			},
			{
				name:     "Link with special characters in text",
				input:    "Visit [My Site (Beta)!](https://example.com)",
				expected: "Visit <https://example.com|My Site (Beta)!>",
			},
			{
				name:     "Link with no protocol",
				input:    "Check [example.com](example.com)",
				expected: "Check <example.com|example.com>",
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

func TestSanitiseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "HTTPS GitHub URL",
			input:    "https://github.com/presmihaylov/foobar",
			expected: "github.com/presmihaylov/foobar",
		},
		{
			name:     "HTTP GitHub URL",
			input:    "http://github.com/presmihaylov/foobar",
			expected: "github.com/presmihaylov/foobar",
		},
		{
			name:     "Already sanitized URL",
			input:    "github.com/presmihaylov/foobar",
			expected: "github.com/presmihaylov/foobar",
		},
		{
			name:     "HTTPS URL with path",
			input:    "https://example.com/path/to/resource",
			expected: "example.com/path/to/resource",
		},
		{
			name:     "HTTP URL with query params and fragment",
			input:    "http://example.com/path/to/resource?query=1&foo=bar#fragment",
			expected: "example.com/path/to/resource",
		},
		{
			name:     "FTP URL",
			input:    "ftp://my.site.net/data",
			expected: "my.site.net/data",
		},
		{
			name:     "URL with port",
			input:    "https://localhost:8080/api/endpoint",
			expected: "localhost:8080/api/endpoint",
		},
		{
			name:     "GitHub SSH URL (should return as-is due to parse error)",
			input:    "git@github.com:presmihaylov/foobar.git",
			expected: "git@github.com:presmihaylov/foobar.git",
		},
		{
			name:     "Complex URL with subdomain",
			input:    "https://api.github.com/repos/presmihaylov/foobar",
			expected: "api.github.com/repos/presmihaylov/foobar",
		},
		{
			name:     "URL with only query params",
			input:    "https://example.com?query=1&foo=bar",
			expected: "example.com",
		},
		{
			name:     "URL with only fragment",
			input:    "https://example.com#section",
			expected: "example.com",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Invalid URL format",
			input:    "not-a-valid-url",
			expected: "not-a-valid-url",
		},
		{
			name:     "URL with user info",
			input:    "https://user:pass@example.com/path",
			expected: "example.com/path",
		},
		{
			name:     "GitLab URL",
			input:    "https://gitlab.com/group/project",
			expected: "gitlab.com/group/project",
		},
		{
			name:     "Bitbucket URL",
			input:    "https://bitbucket.org/user/repo",
			expected: "bitbucket.org/user/repo",
		},
		{
			name:     "URL without scheme with path",
			input:    "github.com/user/repo/tree/main",
			expected: "github.com/user/repo/tree/main",
		},
		{
			name:     "URL without scheme (no-scheme/path pattern)",
			input:    "no-scheme/path",
			expected: "no-scheme/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitiseURL(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
