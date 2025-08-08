package handlers

import (
	"context"
	"log"

	"ccbackend/models"
	"ccbackend/services"
)

type DashboardAPIHandler struct {
	usersService             services.UsersService
	slackIntegrationsService services.SlackIntegrationsService
}

func NewDashboardAPIHandler(
	usersService services.UsersService,
	slackIntegrationsService services.SlackIntegrationsService,
) *DashboardAPIHandler {
	return &DashboardAPIHandler{
		usersService:             usersService,
		slackIntegrationsService: slackIntegrationsService,
	}
}

// ListSlackIntegrations returns all Slack integrations for an organization
func (h *DashboardAPIHandler) ListSlackIntegrations(
	ctx context.Context,
	user *models.User,
) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Listing Slack integrations for organization: %s", user.OrganizationID)
	integrations, err := h.slackIntegrationsService.GetSlackIntegrationsByOrganizationID(ctx)
	if err != nil {
		log.Printf("❌ Failed to get Slack integrations: %v", err)
		return nil, err
	}

	log.Printf("✅ Retrieved %d Slack integrations for organization: %s", len(integrations), user.OrganizationID)
	return integrations, nil
}

// CreateSlackIntegration creates a new Slack integration for an organization
func (h *DashboardAPIHandler) CreateSlackIntegration(
	ctx context.Context,
	slackAuthToken, redirectURL string,
	user *models.User,
) (*models.SlackIntegration, error) {
	log.Printf("➕ Creating Slack integration for organization: %s", user.OrganizationID)
	integration, err := h.slackIntegrationsService.CreateSlackIntegration(ctx, slackAuthToken, redirectURL)
	if err != nil {
		log.Printf("❌ Failed to create Slack integration: %v", err)
		return nil, err
	}

	log.Printf("✅ Slack integration created successfully: %s", integration.ID)
	return integration, nil
}

// DeleteSlackIntegration deletes a Slack integration by ID
func (h *DashboardAPIHandler) DeleteSlackIntegration(ctx context.Context, integrationID string) error {
	log.Printf("🗑️ Deleting Slack integration: %s", integrationID)
	if err := h.slackIntegrationsService.DeleteSlackIntegration(ctx, integrationID); err != nil {
		log.Printf("❌ Failed to delete Slack integration: %v", err)
		return err
	}

	log.Printf("✅ Slack integration deleted successfully: %s", integrationID)
	return nil
}

// GenerateCCAgentSecretKey generates a new secret key for a Slack integration
func (h *DashboardAPIHandler) GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error) {
	log.Printf("🔑 Generating CCAgent secret key for integration: %s", integrationID)
	secretKey, err := h.slackIntegrationsService.GenerateCCAgentSecretKey(ctx, integrationID)
	if err != nil {
		log.Printf("❌ Failed to generate CCAgent secret key: %v", err)
		return "", err
	}

	log.Printf("✅ CCAgent secret key generated successfully for integration: %s", integrationID)
	return secretKey, nil
}
