package github_integrations

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type GitHubIntegrationsService struct {
	githubRepo   *db.PostgresGitHubIntegrationsRepository
	githubClient clients.GitHubClient
}

func NewGitHubIntegrationsService(
	repo *db.PostgresGitHubIntegrationsRepository,
	githubClient clients.GitHubClient,
) *GitHubIntegrationsService {
	return &GitHubIntegrationsService{
		githubRepo:   repo,
		githubClient: githubClient,
	}
}

func (s *GitHubIntegrationsService) CreateGitHubIntegration(
	ctx context.Context,
	orgID models.OrgID,
	authCode, installationID string,
) (*models.GitHubIntegration, error) {
	log.Printf("📋 Starting to create GitHub integration for org: %s, installation: %s", orgID, installationID)

	if orgID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}
	if authCode == "" {
		return nil, fmt.Errorf("auth code cannot be empty")
	}
	if installationID == "" {
		return nil, fmt.Errorf("installation ID cannot be empty")
	}

	// Exchange OAuth code for access token
	accessToken, err := s.githubClient.ExchangeCodeForAccessToken(ctx, authCode)
	if err != nil {
		log.Printf("❌ Failed to exchange code for token: %v", err)
		return nil, fmt.Errorf("failed to verify GitHub installation: %w", err)
	}

	// Create the integration
	integration := &models.GitHubIntegration{
		ID:                   core.NewID("ghi"),
		GitHubInstallationID: installationID,
		GitHubAccessToken:    accessToken,
		OrgID:                orgID,
	}

	if err := s.githubRepo.CreateGitHubIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create GitHub integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created GitHub integration with ID: %s", integration.ID)
	return integration, nil
}

func (s *GitHubIntegrationsService) ListGitHubIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubIntegration, error) {
	log.Printf("📋 Starting to list GitHub integrations for org: %s", orgID)
	if orgID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	integrations, err := s.githubRepo.GetGitHubIntegrationsByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d GitHub integrations", len(integrations))
	return integrations, nil
}

func (s *GitHubIntegrationsService) GetGitHubIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	log.Printf("📋 Starting to get GitHub integration by ID: %s for org: %s", id, orgID)
	if orgID == "" {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("organization ID cannot be empty")
	}
	if !core.IsValidULID(id) {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	maybeInt, err := s.githubRepo.GetGitHubIntegrationByID(ctx, orgID, id)
	if err != nil {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("failed to get GitHub integration: %w", err)
	}

	if maybeInt.IsPresent() {
		log.Printf("📋 Completed successfully - found GitHub integration: %s", id)
	} else {
		log.Printf("📋 Completed successfully - GitHub integration not found: %s", id)
	}

	return maybeInt, nil
}

func (s *GitHubIntegrationsService) DeleteGitHubIntegration(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete GitHub integration: %s for org: %s", integrationID, orgID)

	if orgID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	// Get the integration to retrieve the installation ID
	integrationOpt, err := s.githubRepo.GetGitHubIntegrationByID(ctx, orgID, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub integration: %w", err)
	}
	integration, exists := integrationOpt.Get()
	if !exists {
		return fmt.Errorf("GitHub integration not found")
	}

	// Attempt to uninstall the GitHub App
	// We continue with deletion even if uninstall fails (app might already be uninstalled)
	if err := s.githubClient.UninstallApp(ctx, integration.GitHubInstallationID); err != nil {
		log.Printf("⚠️ Failed to uninstall GitHub App (installation ID: %s): %v", integration.GitHubInstallationID, err)
		// Continue with deletion - the app might already be uninstalled
	}

	if err := s.githubRepo.DeleteGitHubIntegration(ctx, orgID, integrationID); err != nil {
		return fmt.Errorf("failed to delete GitHub integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted GitHub integration: %s", integrationID)
	return nil
}

func (s *GitHubIntegrationsService) ListAvailableRepositories(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.GitHubRepository, error) {
	log.Printf("📋 Starting to list available GitHub repositories for org: %s", orgID)

	if orgID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	// Get the GitHub integration for the organization
	integrations, err := s.githubRepo.GetGitHubIntegrationsByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub integrations: %w", err)
	}
	if len(integrations) == 0 {
		return []models.GitHubRepository{}, nil
	}

	// Use the first integration (typically there's only one per org)
	integration := integrations[0]

	// Get repositories accessible by the GitHub App installation
	repositories, err := s.githubClient.ListInstalledRepositories(ctx, integration.GitHubInstallationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub repositories: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d accessible repositories", len(repositories))
	return repositories, nil
}
