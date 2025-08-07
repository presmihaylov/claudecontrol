package clients

import (
	"fmt"
	"net/http"
)

// MockSlackClient implements SlackClient interface for testing
type MockSlackClient struct {
	// MockGetOAuthV2Response allows setting custom response or error for testing
	MockGetOAuthV2Response func(httpClient *http.Client, clientID, clientSecret, code, redirectURL string) (*OAuthV2Response, error)
}

// NewMockSlackClient creates a new mock Slack client
func NewMockSlackClient() *MockSlackClient {
	return &MockSlackClient{}
}

// GetOAuthV2Response implements SlackClient interface for testing
func (m *MockSlackClient) GetOAuthV2Response(
	httpClient *http.Client,
	clientID, clientSecret, code, redirectURL string,
) (*OAuthV2Response, error) {
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
