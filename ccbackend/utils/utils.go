package utils

import (
	"context"
	"fmt"
	"log"
	"regexp"

	"ccbackend/models"
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

// SlackUserInfoClient interface for getting user information from Slack
type SlackUserInfoClient interface {
	GetUserInfoContext(ctx context.Context, user string) (*models.SlackUser, error)
}

// ResolveMentionsInSlackMessage resolves user mentions like <@U123456> to display names
// in incoming Slack messages before forwarding them to ccagent
func ResolveMentionsInSlackMessage(ctx context.Context, message string, slackClient SlackUserInfoClient) string {
	// Regex to match user mentions in the format <@U123456>
	mentionRegex := regexp.MustCompile(`<@([UW][A-Z0-9]+)>`)

	// Find all unique user IDs mentioned in the message
	matches := mentionRegex.FindAllStringSubmatch(message, -1)
	if len(matches) == 0 {
		return message // No mentions found
	}

	// Create a map to cache user info to avoid duplicate API calls
	userCache := make(map[string]string)

	// Resolve each unique user ID
	for _, match := range matches {
		userID := match[1] // The captured group contains the user ID

		// Skip if we already resolved this user
		if _, exists := userCache[userID]; exists {
			continue
		}

		// Get user info from Slack API
		user, err := slackClient.GetUserInfoContext(ctx, userID)
		if err != nil {
			log.Printf("⚠️ Failed to resolve user mention %s: %v", userID, err)
			// If we can't resolve, keep the original mention format
			userCache[userID] = fmt.Sprintf("<@%s>", userID)
			continue
		}

		// Get the best display name available
		displayName := getUserDisplayName(user)
		userCache[userID] = fmt.Sprintf("@%s", displayName)

		log.Printf("🔍 Resolved user mention %s to %s", userID, displayName)
	}

	// Replace all mentions in the message
	result := mentionRegex.ReplaceAllStringFunc(message, func(match string) string {
		// Extract user ID from the match
		submatches := mentionRegex.FindStringSubmatch(match)
		if len(submatches) > 1 {
			userID := submatches[1]
			if resolvedName, exists := userCache[userID]; exists {
				return resolvedName
			}
		}
		return match // Fallback to original mention if something went wrong
	})

	return result
}

// getUserDisplayName extracts the best available display name from a Slack user object
func getUserDisplayName(user *models.SlackUser) string {
	// Priority: DisplayName > RealName > Name > ID
	if user.Profile.DisplayName != "" {
		return user.Profile.DisplayName
	}
	if user.Profile.RealName != "" {
		return user.Profile.RealName
	}
	if user.Name != "" {
		return user.Name
	}
	return user.ID // Fallback to user ID if no name is available
}
