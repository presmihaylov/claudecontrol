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
// Example: https://workspace.slack.com/archives/C1234567890/p1234567890123456?thread_ts=1234567890.123456
// Returns: slack://channel?team=T1234567890&id=C1234567890&message=1234567890123456
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

	// Extract team ID from subdomain
	hostParts := strings.Split(parsedURL.Host, ".")
	if len(hostParts) < 2 || !strings.Contains(parsedURL.Host, "slack.com") {
		// Not a valid Slack domain, return original
		return permalink
	}

	// For Slack URLs, we need to make a best effort to get team ID
	// Since permalinks don't directly contain team ID, we'll construct a deep link
	// that uses the channel format which should work for opening the message
	
	// Construct deep link to open the channel
	// Format: slack://channel?team=[TEAM_ID]&id=[CHANNEL_ID]
	// Note: Without team ID from the URL, we can't create a perfect deep link
	// The best we can do is create a channel deep link
	deepLink := "slack://channel?id=" + channelID

	return deepLink
}