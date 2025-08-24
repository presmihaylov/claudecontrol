package github_integrations

import (
	"context"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

type MockGitHubIntegrationsService struct {
	mock.Mock
}

func (m *MockGitHubIntegrationsService) CreateGitHubIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	authCode, installationID string,
) (*models.GitHubIntegration, error) {
	args := m.Called(ctx, organizationID, authCode, installationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.GitHubIntegration), args.Error(1)
}

func (m *MockGitHubIntegrationsService) ListGitHubIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.GitHubIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.GitHubIntegration), args.Error(1)
}

func (m *MockGitHubIntegrationsService) GetGitHubIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	args := m.Called(ctx, organizationID, id)
	if args.Get(0) == nil {
		return mo.None[*models.GitHubIntegration](), args.Error(1)
	}
	return args.Get(0).(mo.Option[*models.GitHubIntegration]), args.Error(1)
}

func (m *MockGitHubIntegrationsService) DeleteGitHubIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}

func (m *MockGitHubIntegrationsService) ListAvailableRepositories(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.GitHubRepository, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]models.GitHubRepository), args.Error(1)
}
