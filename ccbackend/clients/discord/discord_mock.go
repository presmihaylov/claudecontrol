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

// GetBotUser mocks fetching bot user information
func (m *MockDiscordClient) GetBotUser() (*clients.DiscordBotUser, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordBotUser), args.Error(1)
}

// GetChannelByID mocks fetching specific Discord channel by ID
func (m *MockDiscordClient) GetChannelByID(channelID string) (*clients.DiscordChannel, error) {
	args := m.Called(channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordChannel), args.Error(1)
}

// PostMessage mocks posting a message to Discord
func (m *MockDiscordClient) PostMessage(
	channelID string,
	params clients.DiscordMessageParams,
) (*clients.DiscordPostMessageResponse, error) {
	args := m.Called(channelID, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordPostMessageResponse), args.Error(1)
}

// AddReaction mocks adding a reaction to a Discord message
func (m *MockDiscordClient) AddReaction(channelID, messageID, emoji string) error {
	args := m.Called(channelID, messageID, emoji)
	return args.Error(0)
}

// RemoveReaction mocks removing a reaction from a Discord message
func (m *MockDiscordClient) RemoveReaction(channelID, messageID, emoji string) error {
	args := m.Called(channelID, messageID, emoji)
	return args.Error(0)
}

// CreatePublicThread mocks creating a public thread from a Discord message
func (m *MockDiscordClient) CreatePublicThread(
	channelID, messageID, threadName string,
) (*clients.DiscordThreadResponse, error) {
	args := m.Called(channelID, messageID, threadName)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*clients.DiscordThreadResponse), args.Error(1)
}
