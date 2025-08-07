package slack

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"regexp"

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
	params clients.SlackMessageParams,
) (*clients.SlackPostMessageResponse, error) {
	var sdkOptions []slack.MsgOption
	if params.Text != "" {
		sdkOptions = append(sdkOptions, slack.MsgOptionText(params.Text, false))
	}
	if params.ThreadTS != "" {
		sdkOptions = append(sdkOptions, slack.MsgOptionTS(params.ThreadTS))
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

// ResolveMentionsInMessage resolves user mentions like <@U123456> to display names
// in incoming Slack messages before forwarding them to ccagent
func (c *SlackClient) ResolveMentionsInMessage(ctx context.Context, message string) string {
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
		user, err := c.GetUserInfoContext(ctx, userID)
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
func getUserDisplayName(user *clients.SlackUser) string {
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
