package utils

import (
	"net/url"
	"strings"
)

func AssertInvariant(condition bool, message string) {
	if !condition {
		panic("invariant violated - " + message)
	}
}

// ConvertSlackPermalinkToDeepLink converts a Slack permalink URL to a deep link
// The backend now sends proper deep links, so this function mainly ensures compatibility
// and handles edge cases where old permalinks might still be present
func ConvertSlackPermalinkToDeepLink(permalink string) string {
	// If it's already a deep link, return as-is
	if strings.HasPrefix(permalink, "slack://") {
		return permalink
	}

	// Parse the URL
	parsedURL, err := url.Parse(permalink)
	if err != nil {
		// If we can't parse it, return the original
		return permalink
	}

	// Extract components from Slack permalink
	// Format: https://[workspace].slack.com/archives/[CHANNEL_ID]/p[MESSAGE_TS]
	pathParts := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
	
	if len(pathParts) < 3 || pathParts[0] != "archives" {
		// Not a valid Slack archives URL, return original
		return permalink
	}

	channelID := pathParts[1]
	messageTS := pathParts[2]

	// Remove 'p' prefix from message timestamp if present
	if strings.HasPrefix(messageTS, "p") {
		messageTS = messageTS[1:]
	}

	// Since we can't extract team ID from permalink, create a basic deep link
	// The backend should now be sending proper deep links with team ID, so this is a fallback
	deepLink := "slack://channel?id=" + channelID

	return deepLink
}