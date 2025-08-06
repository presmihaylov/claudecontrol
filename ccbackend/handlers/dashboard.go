package handlers

import (
	"context"
	"log"

	"ccbackend/models"
	"ccbackend/services"
)

type DashboardAPIHandler struct {
	services services.DashboardServicesInterface
}

func NewDashboardAPIHandler(services services.DashboardServicesInterface) *DashboardAPIHandler {
	return &DashboardAPIHandler{
		services: services,
	}
}

// ListSlackIntegrations returns all Slack integrations for a user
func (h *DashboardAPIHandler) ListSlackIntegrations(user *models.User) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Listing Slack integrations for user: %s", user.ID)

	integrations, err := h.services.GetSlackIntegrationsByUserID(user.ID)
	if err != nil {
		log.Printf("❌ Failed to get Slack integrations: %v", err)
		return nil, err
	}

	log.Printf("✅ Retrieved %d Slack integrations for user: %s", len(integrations), user.ID)
	return integrations, nil
}

// CreateSlackIntegration creates a new Slack integration for a user
func (h *DashboardAPIHandler) CreateSlackIntegration(slackAuthToken, redirectURL string, user *models.User) (*models.SlackIntegration, error) {
	log.Printf("➕ Creating Slack integration for user: %s", user.ID)

	// Create Slack integration using the authenticated user ID
	integration, err := h.services.CreateSlackIntegration(slackAuthToken, redirectURL, user.ID)
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

	// Delete the integration (service will get user from context)
	if err := h.services.DeleteSlackIntegration(ctx, integrationID); err != nil {
		log.Printf("❌ Failed to delete Slack integration: %v", err)
		return err
	}

	log.Printf("✅ Slack integration deleted successfully: %s", integrationID)
	return nil
}

// GenerateCCAgentSecretKey generates a new secret key for a Slack integration
func (h *DashboardAPIHandler) GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error) {
	log.Printf("🔑 Generating CCAgent secret key for integration: %s", integrationID)

	// Generate the secret key (service will get user from context)
	secretKey, err := h.services.GenerateCCAgentSecretKey(ctx, integrationID)
	if err != nil {
		log.Printf("❌ Failed to generate CCAgent secret key: %v", err)
		return "", err
	}

	log.Printf("✅ CCAgent secret key generated successfully for integration: %s", integrationID)
	return secretKey, nil
}
