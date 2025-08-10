package slackintegrations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockSlackIntegrationsService is a mock implementation of the SlackIntegrationsService interface
type MockSlackIntegrationsService struct {
	mock.Mock
}

func (m *MockSlackIntegrationsService) CreateSlackIntegration(
	ctx context.Context,
	organizationID models.OrgID, slackAuthCode, redirectURL string,
) (*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID, slackAuthCode, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID models.OrgID,
) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetAllSlackIntegrations(
	ctx context.Context,
) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) DeleteSlackIntegration(
	ctx context.Context,
	organizationID models.OrgID, integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByTeamID(
	ctx context.Context,
	teamID string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, teamID)
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, id)
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}
