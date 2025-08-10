package discordintegrations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockDiscordIntegrationsService is a mock implementation of the DiscordIntegrationsService interface
type MockDiscordIntegrationsService struct {
	mock.Mock
}

func (m *MockDiscordIntegrationsService) CreateDiscordIntegration(
	ctx context.Context,
	organizationID models.OrgID, discordAuthCode, guildID, redirectURL string,
) (*models.DiscordIntegration, error) {
	args := m.Called(ctx, organizationID, discordAuthCode, guildID, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.DiscordIntegration), args.Error(1)
}

func (m *MockDiscordIntegrationsService) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.DiscordIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscordIntegration), args.Error(1)
}

func (m *MockDiscordIntegrationsService) GetAllDiscordIntegrations(
	ctx context.Context,
) ([]*models.DiscordIntegration, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.DiscordIntegration), args.Error(1)
}

func (m *MockDiscordIntegrationsService) DeleteDiscordIntegration(
	ctx context.Context,
	organizationID models.OrgID, integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}

func (m *MockDiscordIntegrationsService) GetDiscordIntegrationByGuildID(
	ctx context.Context,
	guildID string,
) (mo.Option[*models.DiscordIntegration], error) {
	args := m.Called(ctx, guildID)
	if args.Get(0) == nil {
		return mo.None[*models.DiscordIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.DiscordIntegration]), args.Error(1)
}

func (m *MockDiscordIntegrationsService) GetDiscordIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.DiscordIntegration], error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return mo.None[*models.DiscordIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.DiscordIntegration]), args.Error(1)
}
