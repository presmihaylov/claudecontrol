package discord

import (
	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
)

// MockDiscordClient implements the clients.DiscordClient interface for testing
type MockDiscordClient struct {
	mock.Mock
}

// ExchangeCodeForToken mocks the Discord OAuth code exchange
func (m *MockDiscordClient) ExchangeCodeForToken(
	clientID, clientSecret, code, redirectURL string,
) (*clients.DiscordOAuthResponse, error) {
	args := m.Called(clientID, clientSecret, code, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordOAuthResponse), args.Error(1)
}

// GetGuildByID mocks fetching specific Discord guild by ID
func (m *MockDiscordClient) GetGuildByID(guildID string) (*clients.DiscordGuild, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordGuild), args.Error(1)
}
