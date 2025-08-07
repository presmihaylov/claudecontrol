package clients

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"
)

// SlackOAuthClient defines the interface for Slack OAuth operations
type SlackOAuthClient interface {
	GetOAuthV2Response(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error)
}

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	SlackOAuthClient

	// Bot operations
	AuthTest() (*slack.AuthTestResponse, error)
	GetPermalink(params *slack.PermalinkParameters) (string, error)

	// User operations
	GetUserInfoContext(ctx context.Context, userID string) (*slack.User, error)

	// Message operations
	PostMessage(channelID string, options ...slack.MsgOption) (string, string, error)

	// Reaction operations
	GetReactions(item slack.ItemRef, params slack.GetReactionsParameters) ([]slack.ItemReaction, error)
	AddReaction(name string, item slack.ItemRef) error
	RemoveReaction(name string, item slack.ItemRef) error
}
