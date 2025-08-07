package clients

import (
	"context"
	"fmt"
	"net/http"

	"github.com/slack-go/slack"
)

// MockSlackClient implements SlackClient interface for testing
type MockSlackClient struct {
	// OAuth operations
	MockGetOAuthV2Response func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error)

	// Bot operations
	MockAuthTest     func() (*slack.AuthTestResponse, error)
	MockGetPermalink func(params *slack.PermalinkParameters) (string, error)

	// User operations
	MockGetUserInfoContext func(ctx context.Context, userID string) (*slack.User, error)

	// Message operations
	MockPostMessage func(channelID string, options ...slack.MsgOption) (string, string, error)

	// Reaction operations
	MockGetReactions   func(item slack.ItemRef, params slack.GetReactionsParameters) ([]slack.ItemReaction, error)
	MockAddReaction    func(name string, item slack.ItemRef) error
	MockRemoveReaction func(name string, item slack.ItemRef) error
}

// NewMockSlackClient creates a new mock Slack client
func NewMockSlackClient() *MockSlackClient {
	return &MockSlackClient{}
}

// GetOAuthV2Response implements SlackClient interface for testing
func (m *MockSlackClient) GetOAuthV2Response(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error) {
	if m.MockGetOAuthV2Response != nil {
		return m.MockGetOAuthV2Response(httpClient, clientID, clientSecret, code, redirectURL)
	}

	// Default mock response for testing
	return &OAuthV2Response{
		TeamID:      "T123456789",
		TeamName:    "Test Team",
		AccessToken: "xoxb-test-token-123",
	}, nil
}

// AuthTest implements SlackClient interface for testing
func (m *MockSlackClient) AuthTest() (*slack.AuthTestResponse, error) {
	if m.MockAuthTest != nil {
		return m.MockAuthTest()
	}

	// Default mock response
	return &slack.AuthTestResponse{
		UserID: "U123456789",
		TeamID: "T123456789",
	}, nil
}

// GetPermalink implements SlackClient interface for testing
func (m *MockSlackClient) GetPermalink(params *slack.PermalinkParameters) (string, error) {
	if m.MockGetPermalink != nil {
		return m.MockGetPermalink(params)
	}

	// Default mock response
	return fmt.Sprintf("https://test-workspace.slack.com/archives/%s/p%s", params.Channel, params.Ts), nil
}

// GetUserInfoContext implements SlackClient interface for testing
func (m *MockSlackClient) GetUserInfoContext(ctx context.Context, userID string) (*slack.User, error) {
	if m.MockGetUserInfoContext != nil {
		return m.MockGetUserInfoContext(ctx, userID)
	}

	// Default mock response
	return &slack.User{
		ID:   userID,
		Name: "testuser",
		Profile: slack.UserProfile{
			DisplayName: "Test User",
			RealName:    "Test User",
		},
	}, nil
}

// PostMessage implements SlackClient interface for testing
func (m *MockSlackClient) PostMessage(channelID string, options ...slack.MsgOption) (string, string, error) {
	if m.MockPostMessage != nil {
		return m.MockPostMessage(channelID, options...)
	}

	// Default mock response
	return channelID, "1234567890.123456", nil
}

// GetReactions implements SlackClient interface for testing
func (m *MockSlackClient) GetReactions(item slack.ItemRef, params slack.GetReactionsParameters) ([]slack.ItemReaction, error) {
	if m.MockGetReactions != nil {
		return m.MockGetReactions(item, params)
	}

	// Default mock response - no reactions
	return []slack.ItemReaction{}, nil
}

// AddReaction implements SlackClient interface for testing
func (m *MockSlackClient) AddReaction(name string, item slack.ItemRef) error {
	if m.MockAddReaction != nil {
		return m.MockAddReaction(name, item)
	}

	// Default success
	return nil
}

// RemoveReaction implements SlackClient interface for testing
func (m *MockSlackClient) RemoveReaction(name string, item slack.ItemRef) error {
	if m.MockRemoveReaction != nil {
		return m.MockRemoveReaction(name, item)
	}

	// Default success
	return nil
}

// WithOAuthV2Response sets up mock to return specific response
func (m *MockSlackClient) WithOAuthV2Response(response *OAuthV2Response) *MockSlackClient {
	m.MockGetOAuthV2Response = func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error) {
		return response, nil
	}
	return m
}

// WithOAuthV2Error sets up mock to return specific error
func (m *MockSlackClient) WithOAuthV2Error(err error) *MockSlackClient {
	m.MockGetOAuthV2Response = func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error) {
		return nil, err
	}
	return m
}

// WithOAuthV2Validation sets up mock with parameter validation
func (m *MockSlackClient) WithOAuthV2Validation() *MockSlackClient {
	m.MockGetOAuthV2Response = func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error) {
		if clientID == "" {
			return nil, fmt.Errorf("client ID is required")
		}
		if clientSecret == "" {
			return nil, fmt.Errorf("client secret is required")
		}
		if code == "" {
			return nil, fmt.Errorf("authorization code is required")
		}

		return &OAuthV2Response{
			TeamID:      "T123456789",
			TeamName:    "Test Team",
			AccessToken: "xoxb-test-token-123",
		}, nil
	}
	return m
}
