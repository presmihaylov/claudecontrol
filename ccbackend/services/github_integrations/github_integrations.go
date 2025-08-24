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
	organizationID models.OrgID,
	authCode, installationID string,
) (*models.GitHubIntegration, error) {
	log.Printf("📋 Starting to create GitHub integration for org: %s, installation: %s", organizationID, installationID)

	if organizationID == "" {
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
		OrgID:                organizationID,
	}

	if err := s.githubRepo.CreateGitHubIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create GitHub integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created GitHub integration with ID: %s", integration.ID)
	return integration, nil
}

func (s *GitHubIntegrationsService) ListGitHubIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.GitHubIntegration, error) {
	log.Printf("📋 Starting to list GitHub integrations for org: %s", organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	integrations, err := s.githubRepo.GetGitHubIntegrationsByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list GitHub integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d GitHub integrations", len(integrations))
	return integrations, nil
}

func (s *GitHubIntegrationsService) GetGitHubIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.GitHubIntegration], error) {
	log.Printf("📋 Starting to get GitHub integration by ID: %s", id)

	if id == "" {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("integration ID cannot be empty")
	}
	if !core.IsValidULID(id) {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	integration, err := s.githubRepo.GetGitHubIntegrationByID(ctx, id)
	if err != nil {
		return mo.None[*models.GitHubIntegration](), fmt.Errorf("failed to get GitHub integration: %w", err)
	}

	if integration.IsPresent() {
		log.Printf("📋 Completed successfully - found GitHub integration: %s", id)
	} else {
		log.Printf("📋 Completed successfully - GitHub integration not found: %s", id)
	}

	return integration, nil
}

func (s *GitHubIntegrationsService) DeleteGitHubIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete GitHub integration: %s for org: %s", integrationID, organizationID)

	if organizationID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if integrationID == "" {
		return fmt.Errorf("integration ID cannot be empty")
	}
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	// Get the integration to retrieve the installation ID
	integrationOpt, err := s.githubRepo.GetGitHubIntegrationByID(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub integration: %w", err)
	}

	integration, exists := integrationOpt.Get()
	if !exists {
		return fmt.Errorf("GitHub integration not found")
	}

	// Verify the integration belongs to this organization
	if integration.OrgID != organizationID {
		return fmt.Errorf("GitHub integration does not belong to this organization")
	}

	// Attempt to uninstall the GitHub App
	// We continue with deletion even if uninstall fails (app might already be uninstalled)
	if err := s.githubClient.UninstallApp(ctx, integration.GitHubInstallationID); err != nil {
		log.Printf("⚠️ Failed to uninstall GitHub App (installation ID: %s): %v", integration.GitHubInstallationID, err)
		// Continue with deletion - the app might already be uninstalled
	}

	if err := s.githubRepo.DeleteGitHubIntegration(ctx, organizationID, integrationID); err != nil {
		return fmt.Errorf("failed to delete GitHub integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted GitHub integration: %s", integrationID)
	return nil
}
