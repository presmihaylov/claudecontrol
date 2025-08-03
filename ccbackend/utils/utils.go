package utils

import (
	"fmt"
	"regexp"
	"strings"
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

	// Convert lines starting with hashtags (headings) to bold text
	// This regex matches lines starting with one or more # followed by optional space and text
	headingRegex := regexp.MustCompile(`(?m)^#+\s*(.+)$`)
	result = headingRegex.ReplaceAllString(result, "*$1*")

	return result
}

// CreateSlackDeepLink creates a Slack deep link for opening a specific message in the native Slack app
// Format: slack://channel?team={slackTeamId}&id={slackChannelId}&message={slackMessageTimestamp}
func CreateSlackDeepLink(teamID, channelID, messageTS string) string {
	// Convert message timestamp from format like "1640995200.123456" to "1640995200123456"
	// by removing the decimal point
	messageTimestamp := strings.ReplaceAll(messageTS, ".", "")
	
	return fmt.Sprintf("slack://channel?team=%s&id=%s&message=%s", teamID, channelID, messageTimestamp)
}

// ResolveMentionsInSlackMessage resolves user mentions in Slack messages
// This is a placeholder implementation - in main branch this function
// would resolve @mentions to readable names using the Slack API
func ResolveMentionsInSlackMessage(ctx interface{}, message string, slackClient interface{}) string {
	// For now, just return the message as-is
	// The full implementation would use slackClient to resolve user mentions
	return message
}
