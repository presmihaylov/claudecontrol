package anthropic

import (
	"context"
	"time"

	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
)

// MockAnthropicClient is a mock implementation of clients.AnthropicClient
type MockAnthropicClient struct {
	mock.Mock
}

// ExchangeCodeForTokens mocks the OAuth token exchange functionality
func (m *MockAnthropicClient) ExchangeCodeForTokens(
	ctx context.Context,
	authCode, codeVerifier string,
) (*clients.AnthropicTokens, error) {
	args := m.Called(ctx, authCode, codeVerifier)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AnthropicTokens), args.Error(1)
}

// RefreshAccessToken mocks the token refresh functionality
func (m *MockAnthropicClient) RefreshAccessToken(
	ctx context.Context,
	refreshToken string,
) (*clients.AnthropicTokens, error) {
	args := m.Called(ctx, refreshToken)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.AnthropicTokens), args.Error(1)
}

// NewMockAnthropicClient creates a new mock client for testing
func NewMockAnthropicClient() *MockAnthropicClient {
	return &MockAnthropicClient{}
}

// WithTokenExchangeResponse configures mock to return specific tokens on ExchangeCodeForTokens
func (m *MockAnthropicClient) WithTokenExchangeResponse(tokens *clients.AnthropicTokens) *MockAnthropicClient {
	m.On("ExchangeCodeForTokens", mock.Anything, mock.Anything, mock.Anything).Return(tokens, nil)
	return m
}

// WithRefreshTokenResponse configures mock to return specific tokens on RefreshAccessToken
func (m *MockAnthropicClient) WithRefreshTokenResponse(tokens *clients.AnthropicTokens) *MockAnthropicClient {
	m.On("RefreshAccessToken", mock.Anything, mock.Anything).Return(tokens, nil)
	return m
}

// CreateTestTokens creates sample AnthropicTokens for testing
func CreateTestTokens() *clients.AnthropicTokens {
	return &clients.AnthropicTokens{
		AccessToken:  "test-access-token-123",
		RefreshToken: "test-refresh-token-456",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
}

// CreateRefreshedTestTokens creates new sample AnthropicTokens for refresh scenarios
func CreateRefreshedTestTokens() *clients.AnthropicTokens {
	return &clients.AnthropicTokens{
		AccessToken:  "refreshed-access-token-789",
		RefreshToken: "refreshed-refresh-token-abc",
		ExpiresAt:    time.Now().Add(time.Hour),
	}
}
