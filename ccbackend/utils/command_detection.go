package utils

import (
	"regexp"
	"strings"
)

// CommandDetectionResult represents the result of command detection
type CommandDetectionResult struct {
	IsCommand   bool
	CommandText string
}

// DetectCommand checks if a message text contains a command after stripping mentions
func DetectCommand(messageText string) CommandDetectionResult {
	// Strip mentions from the message text
	strippedText := StripMentions(messageText)

	// Trim whitespace
	strippedText = strings.TrimSpace(strippedText)

	// Check if the text starts with --cmd
	if strings.HasPrefix(strippedText, "--cmd") {
		return CommandDetectionResult{
			IsCommand:   true,
			CommandText: strippedText,
		}
	}

	return CommandDetectionResult{
		IsCommand:   false,
		CommandText: "",
	}
}

// StripMentions removes Slack and Discord mentions from message text
func StripMentions(text string) string {
	// Remove Slack mentions: <@USER_ID> or <@USER_ID|username>
	slackMentionRegex := regexp.MustCompile(`<@[^>|]+(?:\|[^>]+)?>`)
	text = slackMentionRegex.ReplaceAllString(text, "")

	// Remove Discord mentions: <@USER_ID> or <@!USER_ID>
	discordMentionRegex := regexp.MustCompile(`<@!?[0-9]+>`)
	text = discordMentionRegex.ReplaceAllString(text, "")

	// Remove any extra whitespace
	text = strings.TrimSpace(text)

	return text
}