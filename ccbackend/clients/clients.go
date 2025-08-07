package clients

import (
	"context"
	"net/http"
)

// OAuthV2Response represents our custom OAuth response with only needed fields
type OAuthV2Response struct {
	TeamID      string
	TeamName    string
	AccessToken string
}

// SlackOAuthClient defines the interface for Slack OAuth operations
type SlackOAuthClient interface {
	GetOAuthV2Response(
		httpClient *http.Client,
		clientID, clientSecret, code, redirectURL string,
	) (*OAuthV2Response, error)
}

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	SlackOAuthClient

	// Bot operations
	AuthTest() (*SlackAuthTestResponse, error)
	GetPermalink(params *SlackPermalinkParameters) (string, error)

	// User operations
	GetUserInfoContext(ctx context.Context, userID string) (*SlackUser, error)
	ResolveMentionsInMessage(ctx context.Context, message string) string

	// Message operations
	PostMessage(channelID string, params SlackMessageParams) (*SlackPostMessageResponse, error)

	// Reaction operations
	GetReactions(item SlackItemRef, params SlackGetReactionsParameters) ([]SlackItemReaction, error)
	AddReaction(name string, item SlackItemRef) error
	RemoveReaction(name string, item SlackItemRef) error
}
