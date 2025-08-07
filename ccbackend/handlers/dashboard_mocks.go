package handlers

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockUsersService implements UsersService for testing
type MockUsersService struct {
	mock.Mock
}

func (m *MockUsersService) GetOrCreateUser(
	ctx context.Context,
	authProvider, authProviderID string,
) (*models.User, error) {
	args := m.Called(ctx, authProvider, authProviderID)
	return args.Get(0).(*models.User), args.Error(1)
}

// MockSlackIntegrationsService implements SlackIntegrationsService for testing
type MockSlackIntegrationsService struct {
	mock.Mock
}

func (m *MockSlackIntegrationsService) CreateSlackIntegration(
	ctx context.Context,
	slackAuthCode, redirectURL string,
	userID string,
) (*models.SlackIntegration, error) {
	args := m.Called(ctx, slackAuthCode, redirectURL, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByUserID(
	ctx context.Context,
	userID string,
) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx, userID)
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

func (m *MockSlackIntegrationsService) DeleteSlackIntegration(ctx context.Context, integrationID string) error {
	args := m.Called(ctx, integrationID)
	return args.Error(0)
}

func (m *MockSlackIntegrationsService) GenerateCCAgentSecretKey(
	ctx context.Context,
	integrationID string,
) (string, error) {
	args := m.Called(ctx, integrationID)
	return args.String(0), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, secretKey)
	if args.Get(0) == nil {
		return mo.None[*models.SlackIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByTeamID(
	ctx context.Context,
	teamID string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, teamID)
	if args.Get(0) == nil {
		return mo.None[*models.SlackIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return mo.None[*models.SlackIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}
