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
	organizationID, slackAuthCode, redirectURL string,
) (*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID, slackAuthCode, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID string,
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
	organizationID, integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
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

// MockOrganizationsService implements OrganizationsService for testing
type MockOrganizationsService struct {
	mock.Mock
}

func (m *MockOrganizationsService) CreateOrganization(ctx context.Context) (*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return mo.None[*models.Organization](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}

func (m *MockOrganizationsService) GenerateCCAgentSecretKey(
	ctx context.Context,
	organizationID string,
) (string, error) {
	args := m.Called(ctx, organizationID)
	return args.String(0), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, secretKey)
	if args.Get(0) == nil {
		return mo.None[*models.Organization](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}

func (m *MockOrganizationsService) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}
