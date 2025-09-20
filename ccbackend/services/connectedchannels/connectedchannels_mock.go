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

func (m *MockConnectedChannelsService) UpsertConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	channelID string,
	channelType string,
) (*models.ConnectedChannel, error) {
	args := m.Called(ctx, orgID, channelID, channelType)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ConnectedChannel), args.Error(1)
}

func (m *MockConnectedChannelsService) GetConnectedChannelByChannelID(
	ctx context.Context,
	orgID models.OrgID,
	channelID string,
	channelType string,
) (mo.Option[*models.ConnectedChannel], error) {
	args := m.Called(ctx, orgID, channelID, channelType)
	if args.Get(0) == nil {
		return mo.None[*models.ConnectedChannel](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ConnectedChannel]), args.Error(1)
}

func (m *MockConnectedChannelsService) GetConnectedChannelByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.ConnectedChannel], error) {
	args := m.Called(ctx, orgID, id)
	if args.Get(0) == nil {
		return mo.None[*models.ConnectedChannel](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.ConnectedChannel]), args.Error(1)
}

func (m *MockConnectedChannelsService) GetConnectedChannelsByOrganization(
	ctx context.Context,
	orgID models.OrgID,
) ([]*models.ConnectedChannel, error) {
	args := m.Called(ctx, orgID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ConnectedChannel), args.Error(1)
}

func (m *MockConnectedChannelsService) DeleteConnectedChannel(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) error {
	args := m.Called(ctx, orgID, id)
	return args.Error(0)
}

func (m *MockConnectedChannelsService) UpdateConnectedChannelDefaultRepoURL(
	ctx context.Context,
	orgID models.OrgID,
	id string,
	defaultRepoURL *string,
) error {
	args := m.Called(ctx, orgID, id, defaultRepoURL)
	return args.Error(0)
}