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
	// This regex matches **text** where text contains any characters except **
	boldRegex := regexp.MustCompile(`\*\*([^*]+(?:\*[^*])*[^*]*)\*\*`)

	// Replace all instances of **text** with *text*
	result := boldRegex.ReplaceAllString(message, "*$1*")

	// This regex matches lines that begin with one or more hashtags followed by space and content
	// Captures the heading content after removing hashtags and leading space
	headingRegex := regexp.MustCompile(`(?m)^#+\s+(.+)$`)

	// Replace all heading lines with bold text
	result = headingRegex.ReplaceAllString(result, "*$1*")

	return result
}




