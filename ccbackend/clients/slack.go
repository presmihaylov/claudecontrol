package clients

import (
	"net/http"

	"github.com/slack-go/slack"
)

// OAuthV2Response represents our custom OAuth response with only needed fields
type OAuthV2Response struct {
	TeamID      string
	TeamName    string
	AccessToken string
}

// SlackClient defines the interface for Slack API operations
type SlackClient interface {
	// GetOAuthV2Response exchanges an OAuth authorization code for access tokens
	GetOAuthV2Response(
		httpClient *http.Client,
		clientID, clientSecret, code, redirectURL string,
	) (*OAuthV2Response, error)
}

// ConcreteSlackClient implements SlackClient using the slack-go/slack SDK
type ConcreteSlackClient struct{}

// NewConcreteSlackClient creates a new concrete Slack client
func NewConcreteSlackClient() *ConcreteSlackClient {
	return &ConcreteSlackClient{}
}

// GetOAuthV2Response implements SlackClient interface
func (c *ConcreteSlackClient) GetOAuthV2Response(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*OAuthV2Response, error) {
	slackResponse, err := slack.GetOAuthV2Response(httpClient, clientID, clientSecret, code, redirectURL)
	if err != nil {
		return nil, err
	}

	// Map Slack SDK response to our custom response struct
	return &OAuthV2Response{
		TeamID:      slackResponse.Team.ID,
		TeamName:    slackResponse.Team.Name,
		AccessToken: slackResponse.AccessToken,
	}, nil
}
