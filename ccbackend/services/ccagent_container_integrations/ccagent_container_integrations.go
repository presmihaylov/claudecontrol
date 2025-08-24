package ccagentcontainerintegrations

import (
	"context"
	"fmt"
	"log"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"

	"github.com/samber/mo"
)

// CCAgentContainerIntegrationsService handles CCAgent container integration operations
type CCAgentContainerIntegrationsService struct {
	repo *db.PostgresCCAgentContainerIntegrationsRepository
}

// NewCCAgentContainerIntegrationsService creates a new service instance
func NewCCAgentContainerIntegrationsService(
	repo *db.PostgresCCAgentContainerIntegrationsRepository,
) *CCAgentContainerIntegrationsService {
	return &CCAgentContainerIntegrationsService{
		repo: repo,
	}
}

// CreateCCAgentContainerIntegration creates a new CCAgent container integration
func (s *CCAgentContainerIntegrationsService) CreateCCAgentContainerIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	instancesCount int,
	repoURL string,
) (*models.CCAgentContainerIntegration, error) {
	log.Printf("📋 Starting to create CCAgent container integration for org: %s", organizationID)

	// Validation
	if instancesCount < 1 || instancesCount > 10 {
		return nil, fmt.Errorf("instances_count must be between 1 and 10")
	}
	if repoURL == "" {
		return nil, fmt.Errorf("repo_url cannot be empty")
	}

	// Create new integration
	integration := &models.CCAgentContainerIntegration{
		ID:             core.NewID("cci"),
		InstancesCount: instancesCount,
		RepoURL:        repoURL,
		OrgID:          organizationID,
	}

	if err := s.repo.CreateCCAgentContainerIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create CCAgent container integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created CCAgent container integration with ID: %s", integration.ID)
	return integration, nil
}

// ListCCAgentContainerIntegrations retrieves all CCAgent container integrations for an organization
func (s *CCAgentContainerIntegrationsService) ListCCAgentContainerIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.CCAgentContainerIntegration, error) {
	log.Printf("📋 Starting to list CCAgent container integrations for org: %s", organizationID)

	integrations, err := s.repo.ListCCAgentContainerIntegrations(ctx, string(organizationID))
	if err != nil {
		return nil, fmt.Errorf("failed to list CCAgent container integrations: %w", err)
	}

	log.Printf(
		"📋 Completed successfully - found %d CCAgent container integrations for org: %s",
		len(integrations),
		organizationID,
	)
	return integrations, nil
}

// GetCCAgentContainerIntegrationByID retrieves a CCAgent container integration by ID
func (s *CCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	log.Printf("📋 Starting to get CCAgent container integration: %s for org: %s", id, organizationID)

	if !core.IsValidULID(id) {
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf("invalid integration ID")
	}

	integration, err := s.repo.GetCCAgentContainerIntegrationByID(ctx, string(organizationID), id)
	if err != nil {
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf(
			"failed to get CCAgent container integration: %w",
			err,
		)
	}

	if integration.IsPresent() {
		log.Printf("📋 Completed successfully - found CCAgent container integration with ID: %s", id)
	} else {
		log.Printf("📋 Completed successfully - no CCAgent container integration found with ID: %s", id)
	}

	return integration, nil
}

// DeleteCCAgentContainerIntegration deletes a CCAgent container integration
func (s *CCAgentContainerIntegrationsService) DeleteCCAgentContainerIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete CCAgent container integration: %s for org: %s", integrationID, organizationID)

	// Validate ID
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("invalid integration ID")
	}

	// Delete the integration (repository method now handles organization scoping)
	if err := s.repo.DeleteCCAgentContainerIntegration(ctx, string(organizationID), integrationID); err != nil {
		return fmt.Errorf("failed to delete CCAgent container integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted CCAgent container integration: %s", integrationID)
	return nil
}
