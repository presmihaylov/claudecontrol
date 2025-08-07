package slack

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"

	"ccbackend/clients"
)

// SlackClientImpl implements the SlackClient interface using the slack-go/slack SDK
type SlackClientImpl struct {
	*slack.Client
}

// NewSlackClient creates a new Slack client with the provided auth token
func NewSlackClient(authToken string) clients.SlackClient {
	return &SlackClientImpl{
		Client: slack.New(authToken),
	}
}

// GetOAuthV2Response exchanges an OAuth authorization code for access tokens
func (c *SlackClientImpl) GetOAuthV2Response(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*clients.OAuthV2Response, error) {
	slackResponse, err := slack.GetOAuthV2Response(httpClient, clientID, clientSecret, code, redirectURL)
	if err != nil {
		return nil, err
	}

	// Map Slack SDK response to our custom response struct
	return &clients.OAuthV2Response{
		TeamID:      slackResponse.Team.ID,
		TeamName:    slackResponse.Team.Name,
		AccessToken: slackResponse.AccessToken,
	}, nil
}

// AuthTest verifies the bot token and returns information about the bot
func (c *SlackClientImpl) AuthTest() (*slack.AuthTestResponse, error) {
	return c.Client.AuthTest()
}

// GetPermalink gets a permalink URL for a message
func (c *SlackClientImpl) GetPermalink(params *slack.PermalinkParameters) (string, error) {
	return c.Client.GetPermalink(params)
}

// GetUserInfoContext gets information about a Slack user
func (c *SlackClientImpl) GetUserInfoContext(ctx context.Context, userID string) (*slack.User, error) {
	return c.Client.GetUserInfoContext(ctx, userID)
}

// PostMessage sends a message to a Slack channel
func (c *SlackClientImpl) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	return c.Client.PostMessage(channelID, options...)
}

// GetReactions gets the reactions on a message
func (c *SlackClientImpl) GetReactions(item slack.ItemRef, params slack.GetReactionsParameters) ([]slack.ItemReaction, error) {
	return c.Client.GetReactions(item, params)
}

// AddReaction adds a reaction to a message
func (c *SlackClientImpl) AddReaction(name string, item slack.ItemRef) error {
	return c.Client.AddReaction(name, item)
}

// RemoveReaction removes a reaction from a message
func (c *SlackClientImpl) RemoveReaction(name string, item slack.ItemRef) error {
	return c.Client.RemoveReaction(name, item)
}
