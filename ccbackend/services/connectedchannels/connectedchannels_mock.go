package connectedchannels

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockConnectedChannelsService struct {
	mock.Mock
}

// Slack-specific methods
func (m *MockConnectedChannelsService) UpsertSlackConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
) (*models.SlackConnectedChannel, error) {
	args := m.Called(ctx, orgID, teamID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackConnectedChannel), args.Error(1)
}

func (m *MockConnectedChannelsService) GetSlackConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
) (mo.Option[*models.SlackConnectedChannel], error) {
	args := m.Called(ctx, orgID, teamID, channelID)
	if args.Get(0) == nil {
		return mo.None[*models.SlackConnectedChannel](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.SlackConnectedChannel]), args.Error(1)
}

func (m *MockConnectedChannelsService) UpdateSlackChannelDefaultRepo(
	ctx context.Context,
	orgID models.OrgID,
	teamID string,
	channelID string,
	repoURL string,
) (*models.SlackConnectedChannel, error) {
	args := m.Called(ctx, orgID, teamID, channelID, repoURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackConnectedChannel), args.Error(1)
}


// Discord-specific methods
func (m *MockConnectedChannelsService) UpsertDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (*models.DiscordConnectedChannel, error) {
	args := m.Called(ctx, orgID, guildID, channelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscordConnectedChannel), args.Error(1)
}

func (m *MockConnectedChannelsService) GetDiscordConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
) (mo.Option[*models.DiscordConnectedChannel], error) {
	args := m.Called(ctx, orgID, guildID, channelID)
	if args.Get(0) == nil {
		return mo.None[*models.DiscordConnectedChannel](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.DiscordConnectedChannel]), args.Error(1)
}

func (m *MockConnectedChannelsService) UpdateDiscordChannelDefaultRepo(
	ctx context.Context,
	orgID models.OrgID,
	guildID string,
	channelID string,
	repoURL string,
) (*models.DiscordConnectedChannel, error) {
	args := m.Called(ctx, orgID, guildID, channelID, repoURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscordConnectedChannel), args.Error(1)
}


