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
func NewCCAgentContainerIntegrationsService(repo *db.PostgresCCAgentContainerIntegrationsRepository) *CCAgentContainerIntegrationsService {
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

	// Check if integration already exists for this organization
	existingOpt, err := s.repo.GetCCAgentContainerIntegrationByOrgID(ctx, string(organizationID))
	if err != nil {
		return nil, fmt.Errorf("failed to check existing integration: %w", err)
	}
	if existingOpt.IsPresent() {
		return nil, fmt.Errorf("CCAgent container integration already exists for this organization")
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

// GetCCAgentContainerIntegrationByOrgID retrieves a CCAgent container integration by organization ID
func (s *CCAgentContainerIntegrationsService) GetCCAgentContainerIntegrationByOrgID(
	ctx context.Context,
	organizationID models.OrgID,
) (mo.Option[*models.CCAgentContainerIntegration], error) {
	log.Printf("📋 Starting to get CCAgent container integration for org: %s", organizationID)

	integration, err := s.repo.GetCCAgentContainerIntegrationByOrgID(ctx, string(organizationID))
	if err != nil {
		return mo.None[*models.CCAgentContainerIntegration](), fmt.Errorf("failed to get CCAgent container integration: %w", err)
	}

	if integration.IsPresent() {
		log.Printf("📋 Completed successfully - found CCAgent container integration with ID: %s", integration.MustGet().ID)
	} else {
		log.Printf("📋 Completed successfully - no CCAgent container integration found for org: %s", organizationID)
	}

	return integration, nil
}

// UpdateCCAgentContainerIntegration updates an existing CCAgent container integration
func (s *CCAgentContainerIntegrationsService) UpdateCCAgentContainerIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
	updates map[string]any,
) (*models.CCAgentContainerIntegration, error) {
	log.Printf("📋 Starting to update CCAgent container integration: %s for org: %s", id, organizationID)

	// Validate ID
	if !core.IsValidULID(id) {
		return nil, fmt.Errorf("invalid integration ID")
	}

	// Check if integration exists and belongs to the organization
	existingOpt, err := s.repo.GetCCAgentContainerIntegrationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing integration: %w", err)
	}
	if !existingOpt.IsPresent() {
		return nil, fmt.Errorf("CCAgent container integration not found")
	}

	existing := existingOpt.MustGet()
	if existing.OrgID != organizationID {
		return nil, fmt.Errorf("CCAgent container integration does not belong to this organization")
	}

	// Validate updates
	if instancesCount, ok := updates["instances_count"].(int); ok {
		if instancesCount < 1 || instancesCount > 10 {
			return nil, fmt.Errorf("instances_count must be between 1 and 10")
		}
	}
	if repoURL, ok := updates["repo_url"].(string); ok {
		if repoURL == "" {
			return nil, fmt.Errorf("repo_url cannot be empty")
		}
	}

	// Update the integration
	if err := s.repo.UpdateCCAgentContainerIntegration(ctx, id, updates); err != nil {
		return nil, fmt.Errorf("failed to update CCAgent container integration: %w", err)
	}

	// Get updated integration
	updatedOpt, err := s.repo.GetCCAgentContainerIntegrationByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get updated integration: %w", err)
	}
	if !updatedOpt.IsPresent() {
		return nil, fmt.Errorf("failed to retrieve updated integration")
	}

	updated := updatedOpt.MustGet()
	log.Printf("📋 Completed successfully - updated CCAgent container integration: %s", id)
	return updated, nil
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

	// Check if integration exists and belongs to the organization
	existingOpt, err := s.repo.GetCCAgentContainerIntegrationByID(ctx, integrationID)
	if err != nil {
		return fmt.Errorf("failed to get existing integration: %w", err)
	}
	if !existingOpt.IsPresent() {
		return fmt.Errorf("CCAgent container integration not found")
	}

	existing := existingOpt.MustGet()
	if existing.OrgID != organizationID {
		return fmt.Errorf("CCAgent container integration does not belong to this organization")
	}

	// Delete the integration
	if err := s.repo.DeleteCCAgentContainerIntegration(ctx, integrationID); err != nil {
		return fmt.Errorf("failed to delete CCAgent container integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted CCAgent container integration: %s", integrationID)
	return nil
}