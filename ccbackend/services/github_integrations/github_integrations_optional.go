package github_integrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// OptionalGitHubIntegrationsService returns errors for all operations when GitHub is not configured
type OptionalGitHubIntegrationsService struct{}

// NewOptionalGitHubIntegrationsService creates a new optional GitHub integrations service
func NewOptionalGitHubIntegrationsService() *OptionalGitHubIntegrationsService {
	return &OptionalGitHubIntegrationsService{}
}

func (s *OptionalGitHubIntegrationsService) CreateGitHubIntegration(
	ctx context.Context,
	orgID models.OrgID,
	authCode, installationID string,
) (*models.GitHubIntegration, error) {
	return nil, fmt.Errorf("Service GitHub is not configured")
}

func (s *OptionalGitHubIntegrationsService) ListGitHubIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubIntegration, error) {
	return nil, fmt.Errorf("Service GitHub is not configured")
}

func (s *OptionalGitHubIntegrationsService) GetGitHubIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	return mo.None[*models.GitHubIntegration](), fmt.Errorf("Service GitHub is not configured")
}

func (s *OptionalGitHubIntegrationsService) DeleteGitHubIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error {
	return fmt.Errorf("Service GitHub is not configured")
}

func (s *OptionalGitHubIntegrationsService) ListAvailableRepositories(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubRepository, error) {
	return nil, fmt.Errorf("Service GitHub is not configured")
}