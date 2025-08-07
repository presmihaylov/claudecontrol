package slack

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"

	"ccbackend/clients"
	"ccbackend/models"
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
func (c *SlackClientImpl) AuthTest() (*models.SlackAuthTestResponse, error) {
	response, err := c.Client.AuthTest()
	if err != nil {
		return nil, err
	}

	return &models.SlackAuthTestResponse{
		UserID: response.UserID,
		TeamID: response.TeamID,
	}, nil
}

// GetPermalink gets a permalink URL for a message
func (c *SlackClientImpl) GetPermalink(params *models.SlackPermalinkParameters) (string, error) {
	sdkParams := &slack.PermalinkParameters{
		Channel: params.Channel,
		Ts:      params.TS,
	}
	return c.Client.GetPermalink(sdkParams)
}

// GetUserInfoContext gets information about a Slack user
func (c *SlackClientImpl) GetUserInfoContext(ctx context.Context, userID string) (*models.SlackUser, error) {
	user, err := c.Client.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &models.SlackUser{
		ID:   user.ID,
		Name: user.Name,
		Profile: models.SlackUserProfile{
			DisplayName: user.Profile.DisplayName,
			RealName:    user.Profile.RealName,
		},
	}, nil
}

// PostMessage sends a message to a Slack channel
func (c *SlackClientImpl) PostMessage(channelID string, options ...models.SlackMessageOption) (string, string, error) {
	// Convert our custom options to SDK options
	var config models.SlackMessageConfig
	for _, opt := range options {
		opt.Apply(&config)
	}

	var sdkOptions []slack.MsgOption
	if config.Text != "" {
		sdkOptions = append(sdkOptions, slack.MsgOptionText(config.Text, false))
	}
	if config.ThreadTS != "" {
		sdkOptions = append(sdkOptions, slack.MsgOptionTS(config.ThreadTS))
	}

	return c.Client.PostMessage(channelID, sdkOptions...)
}

// GetReactions gets the reactions on a message
func (c *SlackClientImpl) GetReactions(item models.SlackItemRef, params models.SlackGetReactionsParameters) ([]models.SlackItemReaction, error) {
	sdkItem := slack.ItemRef{
		Channel:   item.Channel,
		Timestamp: item.Timestamp,
	}
	sdkParams := slack.GetReactionsParameters{} // Our params struct is empty for now

	reactions, err := c.Client.GetReactions(sdkItem, sdkParams)
	if err != nil {
		return nil, err
	}

	// Convert SDK reactions to our custom reactions
	var customReactions []models.SlackItemReaction
	for _, reaction := range reactions {
		customReactions = append(customReactions, models.SlackItemReaction{
			Name:  reaction.Name,
			Users: reaction.Users,
		})
	}

	return customReactions, nil
}

// AddReaction adds a reaction to a message
func (c *SlackClientImpl) AddReaction(name string, item models.SlackItemRef) error {
	sdkItem := slack.ItemRef{
		Channel:   item.Channel,
		Timestamp: item.Timestamp,
	}
	return c.Client.AddReaction(name, sdkItem)
}

// RemoveReaction removes a reaction from a message
func (c *SlackClientImpl) RemoveReaction(name string, item models.SlackItemRef) error {
	sdkItem := slack.ItemRef{
		Channel:   item.Channel,
		Timestamp: item.Timestamp,
	}
	return c.Client.RemoveReaction(name, sdkItem)
}
