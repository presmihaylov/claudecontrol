package discord

import (
	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
)

// MockDiscordClient implements the clients.DiscordClient interface for testing
type MockDiscordClient struct {
	mock.Mock
}

// GetGuildByID mocks fetching specific Discord guild by ID
func (m *MockDiscordClient) GetGuildByID(guildID string) (*clients.DiscordGuild, error) {
	args := m.Called(guildID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordGuild), args.Error(1)
}
