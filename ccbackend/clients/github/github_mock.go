package github

import (
	"context"

	"github.com/stretchr/testify/mock"
)

// MockGitHubClient is a mock implementation of the GitHubClient interface
type MockGitHubClient struct {
	mock.Mock
}

// ExchangeCodeForAccessToken mocks the OAuth code exchange
func (m *MockGitHubClient) ExchangeCodeForAccessToken(ctx context.Context, code string) (string, error) {
	args := m.Called(ctx, code)
	return args.String(0), args.Error(1)
}

// UninstallApp mocks the GitHub App uninstall operation
func (m *MockGitHubClient) UninstallApp(ctx context.Context, installationID string) error {
	args := m.Called(ctx, installationID)
	return args.Error(0)
}
