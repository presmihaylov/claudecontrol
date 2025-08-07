package clients

import (
	"context"
	"net/http"

	"ccbackend/models"
)

// OAuthV2Response represents our custom OAuth response with only needed fields
type OAuthV2Response struct {
	TeamID      string
	TeamName    string
	AccessToken string
}

// SlackOAuthClient defines the interface for Slack OAuth operations
type SlackOAuthClient interface {
	GetOAuthV2Response(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error)
}

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	SlackOAuthClient

	// Bot operations
	AuthTest() (*models.SlackAuthTestResponse, error)
	GetPermalink(params *models.SlackPermalinkParameters) (string, error)

	// User operations
	GetUserInfoContext(ctx context.Context, userID string) (*models.SlackUser, error)

	// Message operations
	PostMessage(channelID string, options ...models.SlackMessageOption) (*models.SlackPostMessageResponse, error)

	// Reaction operations
	GetReactions(item models.SlackItemRef, params models.SlackGetReactionsParameters) ([]models.SlackItemReaction, error)
	AddReaction(name string, item models.SlackItemRef) error
	RemoveReaction(name string, item models.SlackItemRef) error
}
