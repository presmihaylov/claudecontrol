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

	return result
}




