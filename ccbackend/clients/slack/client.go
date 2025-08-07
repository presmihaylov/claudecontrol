package slack

import (
	"context"
	"net/http"

	"github.com/slack-go/slack"

	"ccbackend/clients"
)

// SlackClient implements the clients.SlackClient interface using the slack-go/slack SDK
type SlackClient struct {
	*slack.Client
}

// NewSlackClient creates a new Slack client with the provided auth token
func NewSlackClient(authToken string) clients.SlackClient {
	return &SlackClient{
		Client: slack.New(authToken),
	}
}

// NewSlackOAuthClient creates a new Slack client for OAuth operations only
// This can be used when you don't have an auth token yet
func NewSlackOAuthClient() clients.SlackOAuthClient {
	return &SlackClient{
		Client: slack.New(""), // Empty token for OAuth-only operations
	}
}

// GetOAuthV2Response exchanges an OAuth authorization code for access tokens
func (c *SlackClient) GetOAuthV2Response(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*clients.OAuthV2Response, error) {
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
func (c *SlackClient) AuthTest() (*clients.SlackAuthTestResponse, error) {
	response, err := c.Client.AuthTest()
	if err != nil {
		return nil, err
	}

	return &clients.SlackAuthTestResponse{
		UserID: response.UserID,
		TeamID: response.TeamID,
	}, nil
}

// GetPermalink gets a permalink URL for a message
func (c *SlackClient) GetPermalink(params *clients.SlackPermalinkParameters) (string, error) {
	sdkParams := &slack.PermalinkParameters{
		Channel: params.Channel,
		Ts:      params.TS,
	}
	return c.Client.GetPermalink(sdkParams)
}

// GetUserInfoContext gets information about a Slack user
func (c *SlackClient) GetUserInfoContext(ctx context.Context, userID string) (*clients.SlackUser, error) {
	user, err := c.Client.GetUserInfoContext(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &clients.SlackUser{
		ID:   user.ID,
		Name: user.Name,
		Profile: clients.SlackUserProfile{
			DisplayName: user.Profile.DisplayName,
			RealName:    user.Profile.RealName,
		},
	}, nil
}

// PostMessage sends a message to a Slack channel
func (c *SlackClient) PostMessage(
	channelID string,
	options ...clients.SlackMessageOption,
) (*clients.SlackPostMessageResponse, error) {
	// Convert our custom options to SDK options
	var config clients.SlackMessageConfig
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

	channel, timestamp, err := c.Client.PostMessage(channelID, sdkOptions...)
	if err != nil {
		return nil, err
	}

	return &clients.SlackPostMessageResponse{
		Channel:   channel,
		Timestamp: timestamp,
	}, nil
}

// GetReactions gets the reactions on a message
func (c *SlackClient) GetReactions(
	item clients.SlackItemRef,
	params clients.SlackGetReactionsParameters,
) ([]clients.SlackItemReaction, error) {
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
	var customReactions []clients.SlackItemReaction
	for _, reaction := range reactions {
		customReactions = append(customReactions, clients.SlackItemReaction{
			Name:  reaction.Name,
			Users: reaction.Users,
		})
	}

	return customReactions, nil
}

// AddReaction adds a reaction to a message
func (c *SlackClient) AddReaction(name string, item clients.SlackItemRef) error {
	sdkItem := slack.ItemRef{
		Channel:   item.Channel,
		Timestamp: item.Timestamp,
	}
	return c.Client.AddReaction(name, sdkItem)
}

// RemoveReaction removes a reaction from a message
func (c *SlackClient) RemoveReaction(name string, item clients.SlackItemRef) error {
	sdkItem := slack.ItemRef{
		Channel:   item.Channel,
		Timestamp: item.Timestamp,
	}
	return c.Client.RemoveReaction(name, sdkItem)
}
