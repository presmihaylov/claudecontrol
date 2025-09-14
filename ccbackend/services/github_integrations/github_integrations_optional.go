package github_integrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UnconfiguredGitHubIntegrationsService returns errors for all operations when GitHub is not configured
type UnconfiguredGitHubIntegrationsService struct{}

// NewUnconfiguredGitHubIntegrationsService creates a new unconfigured GitHub integrations service
func NewUnconfiguredGitHubIntegrationsService() *UnconfiguredGitHubIntegrationsService {
	return &UnconfiguredGitHubIntegrationsService{}
}

func (s *UnconfiguredGitHubIntegrationsService) CreateGitHubIntegration(
	ctx context.Context,
	orgID models.OrgID,
	authCode, installationID string,
) (*models.GitHubIntegration, error) {
	return nil, fmt.Errorf("service GitHub is not configured")
}

func (s *UnconfiguredGitHubIntegrationsService) ListGitHubIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubIntegration, error) {
	return nil, fmt.Errorf("service GitHub is not configured")
}

func (s *UnconfiguredGitHubIntegrationsService) GetGitHubIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	return mo.None[*models.GitHubIntegration](), fmt.Errorf("service GitHub is not configured")
}

func (s *UnconfiguredGitHubIntegrationsService) DeleteGitHubIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	return fmt.Errorf("service GitHub is not configured")
}

func (s *UnconfiguredGitHubIntegrationsService) ListAvailableRepositories(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubRepository, error) {
	return nil, fmt.Errorf("service GitHub is not configured")
}
