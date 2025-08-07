package utils

import (
	"regexp"
)

func AssertInvariant(condition bool, message string) {
	if !condition {
		panic("invariant violated - " + message)
	}
}

func ConvertMarkdownToSlack(message string) string {
	result := message

	// Step 1: Convert markdown links [text](url) to Slack format <url|text>
	// This must be done first to avoid conflicts with other formatting
	linkRegex := regexp.MustCompile(`\[([^\]]+)\]\(([^)]+)\)`)
	result = linkRegex.ReplaceAllString(result, "<$2|$1>")

	// Step 2: Handle headings with embedded bold markdown by extracting and converting the content first
	headingRegex := regexp.MustCompile(`(?m)^#+\s*(.+)$`)
	result = headingRegex.ReplaceAllStringFunc(result, func(match string) string {
		// Extract the heading content after the hashtags
		content := regexp.MustCompile(`^#+\s*(.+)$`).ReplaceAllString(match, "$1")
		// Convert any **bold** to *bold* within the heading content
		boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*`)
		content = boldRegex.ReplaceAllString(content, "$1")
		// Return as Slack bold format
		return "*" + content + "*"
	})

	// Step 3: Convert remaining **text** (double asterisks) to *text* (single asterisks)
	// This handles bold text that's not inside headings
	boldRegex := regexp.MustCompile(`\*\*(.+?)\*\*`)
	result = boldRegex.ReplaceAllString(result, "*$1*")

	return result
}
