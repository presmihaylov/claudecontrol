package clients

import (
	"context"
	"fmt"
	"net/http"

	"ccbackend/models"
)

// MockSlackClient implements SlackClient interface for testing
type MockSlackClient struct {
	// OAuth operations
	MockGetOAuthV2Response func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error)

	// Bot operations
	MockAuthTest     func() (*models.SlackAuthTestResponse, error)
	MockGetPermalink func(params *models.SlackPermalinkParameters) (string, error)

	// User operations
	MockGetUserInfoContext func(ctx context.Context, userID string) (*models.SlackUser, error)

	// Message operations
	MockPostMessage func(channelID string, options ...models.SlackMessageOption) (*models.SlackPostMessageResponse, error)

	// Reaction operations
	MockGetReactions   func(item models.SlackItemRef, params models.SlackGetReactionsParameters) ([]models.SlackItemReaction, error)
	MockAddReaction    func(name string, item models.SlackItemRef) error
	MockRemoveReaction func(name string, item models.SlackItemRef) error
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
func (m *MockSlackClient) AuthTest() (*models.SlackAuthTestResponse, error) {
	if m.MockAuthTest != nil {
		return m.MockAuthTest()
	}

	// Default mock response
	return &models.SlackAuthTestResponse{
		UserID: "U123456789",
		TeamID: "T123456789",
	}, nil
}

// GetPermalink implements SlackClient interface for testing
func (m *MockSlackClient) GetPermalink(params *models.SlackPermalinkParameters) (string, error) {
	if m.MockGetPermalink != nil {
		return m.MockGetPermalink(params)
	}

	// Default mock response
	return fmt.Sprintf("https://test-workspace.slack.com/archives/%s/p%s", params.Channel, params.TS), nil
}

// GetUserInfoContext implements SlackClient interface for testing
func (m *MockSlackClient) GetUserInfoContext(ctx context.Context, userID string) (*models.SlackUser, error) {
	if m.MockGetUserInfoContext != nil {
		return m.MockGetUserInfoContext(ctx, userID)
	}

	// Default mock response
	return &models.SlackUser{
		ID:   userID,
		Name: "testuser",
		Profile: models.SlackUserProfile{
			DisplayName: "Test User",
			RealName:    "Test User",
		},
	}, nil
}

// PostMessage implements SlackClient interface for testing
func (m *MockSlackClient) PostMessage(channelID string, options ...models.SlackMessageOption) (*models.SlackPostMessageResponse, error) {
	if m.MockPostMessage != nil {
		return m.MockPostMessage(channelID, options...)
	}

	// Default mock response
	return &models.SlackPostMessageResponse{
		Channel:   channelID,
		Timestamp: "1234567890.123456",
	}, nil
}

// GetReactions implements SlackClient interface for testing
func (m *MockSlackClient) GetReactions(item models.SlackItemRef, params models.SlackGetReactionsParameters) ([]models.SlackItemReaction, error) {
	if m.MockGetReactions != nil {
		return m.MockGetReactions(item, params)
	}

	// Default mock response - no reactions
	return []models.SlackItemReaction{}, nil
}

// AddReaction implements SlackClient interface for testing
func (m *MockSlackClient) AddReaction(name string, item models.SlackItemRef) error {
	if m.MockAddReaction != nil {
		return m.MockAddReaction(name, item)
	}

	// Default success
	return nil
}

// RemoveReaction implements SlackClient interface for testing
func (m *MockSlackClient) RemoveReaction(name string, item models.SlackItemRef) error {
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
