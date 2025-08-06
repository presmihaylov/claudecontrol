package handlers

import (
	"context"

	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockUsersService implements UsersServiceInterface for testing
type MockUsersService struct {
	mock.Mock
}

func (m *MockUsersService) GetOrCreateUser(authProvider, authProviderID string) (*models.User, error) {
	args := m.Called(authProvider, authProviderID)
	return args.Get(0).(*models.User), args.Error(1)
}

// MockSlackIntegrationsService implements SlackIntegrationsServiceInterface for testing
type MockSlackIntegrationsService struct {
	mock.Mock
}

func (m *MockSlackIntegrationsService) CreateSlackIntegration(slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error) {
	args := m.Called(slackAuthCode, redirectURL, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByUserID(userID string) ([]*models.SlackIntegration, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetAllSlackIntegrations() ([]*models.SlackIntegration, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) DeleteSlackIntegration(ctx context.Context, integrationID string) error {
	args := m.Called(ctx, integrationID)
	return args.Error(0)
}

func (m *MockSlackIntegrationsService) GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error) {
	args := m.Called(ctx, integrationID)
	return args.String(0), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationBySecretKey(secretKey string) (*models.SlackIntegration, error) {
	args := m.Called(secretKey)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByTeamID(teamID string) (*models.SlackIntegration, error) {
	args := m.Called(teamID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByID(id string) (*models.SlackIntegration, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}
