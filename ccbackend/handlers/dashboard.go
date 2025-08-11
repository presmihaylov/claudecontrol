package handlers

import (
	"context"
	"fmt"
	"log"

	"ccbackend/appctx"
	"ccbackend/models"
	"ccbackend/services"
)

type DashboardAPIHandler struct {
	usersService               services.UsersService
	slackIntegrationsService   services.SlackIntegrationsService
	discordIntegrationsService services.DiscordIntegrationsService
	organizationsService       services.OrganizationsService
	agentsService              services.AgentsService
}

func NewDashboardAPIHandler(
	usersService services.UsersService,
	slackIntegrationsService services.SlackIntegrationsService,
	discordIntegrationsService services.DiscordIntegrationsService,
	organizationsService services.OrganizationsService,
	agentsService services.AgentsService,
) *DashboardAPIHandler {
	return &DashboardAPIHandler{
		usersService:               usersService,
		slackIntegrationsService:   slackIntegrationsService,
		discordIntegrationsService: discordIntegrationsService,
		organizationsService:       organizationsService,
		agentsService:              agentsService,
	}
}

// ListSlackIntegrations returns all Slack integrations for an organization
func (h *DashboardAPIHandler) ListSlackIntegrations(
	ctx context.Context,
	user *models.User,
) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Listing Slack integrations for organization: %s", user.OrgID)
	integrations, err := h.slackIntegrationsService.GetSlackIntegrationsByOrganizationID(ctx, user.OrgID)
	if err != nil {
		log.Printf("❌ Failed to get Slack integrations: %v", err)
		return nil, err
	}

	log.Printf("✅ Retrieved %d Slack integrations for organization: %s", len(integrations), user.OrgID)
	return integrations, nil
}

// CreateSlackIntegration creates a new Slack integration for an organization
func (h *DashboardAPIHandler) CreateSlackIntegration(
	ctx context.Context,
	slackAuthToken, redirectURL string,
	user *models.User,
) (*models.SlackIntegration, error) {
	log.Printf("➕ Creating Slack integration for organization: %s", user.OrgID)
	integration, err := h.slackIntegrationsService.CreateSlackIntegration(
		ctx,
		user.OrgID,
		slackAuthToken,
		redirectURL,
	)
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
	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return fmt.Errorf("organization not found in context")
	}
	if err := h.slackIntegrationsService.DeleteSlackIntegration(ctx, models.OrgID(org.ID), integrationID); err != nil {
		log.Printf("❌ Failed to delete Slack integration: %v", err)
		return err
	}

	log.Printf("✅ Slack integration deleted successfully: %s", integrationID)
	return nil
}

// GetOrganization returns the organization for the authenticated user
func (h *DashboardAPIHandler) GetOrganization(ctx context.Context) (*models.Organization, error) {
	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return nil, fmt.Errorf("organization not found in context")
	}

	log.Printf("📋 Retrieved organization: %s", org.ID)
	return org, nil
}

// ListDiscordIntegrations returns all Discord integrations for an organization
func (h *DashboardAPIHandler) ListDiscordIntegrations(
	ctx context.Context,
	user *models.User,
) ([]*models.DiscordIntegration, error) {
	log.Printf("📋 Listing Discord integrations for organization: %s", user.OrgID)
	integrations, err := h.discordIntegrationsService.GetDiscordIntegrationsByOrganizationID(ctx, user.OrgID)
	if err != nil {
		log.Printf("❌ Failed to get Discord integrations: %v", err)
		return nil, err
	}

	log.Printf("✅ Retrieved %d Discord integrations for organization: %s", len(integrations), user.OrgID)
	return integrations, nil
}

// CreateDiscordIntegration creates a new Discord integration for an organization
func (h *DashboardAPIHandler) CreateDiscordIntegration(
	ctx context.Context,
	discordAuthCode, guildID, redirectURL string,
	user *models.User,
) (*models.DiscordIntegration, error) {
	log.Printf("➕ Creating Discord integration for organization: %s", user.OrgID)
	integration, err := h.discordIntegrationsService.CreateDiscordIntegration(
		ctx,
		user.OrgID,
		discordAuthCode,
		guildID,
		redirectURL,
	)
	if err != nil {
		log.Printf("❌ Failed to create Discord integration: %v", err)
		return nil, err
	}

	log.Printf("✅ Discord integration created successfully: %s", integration.ID)
	return integration, nil
}

// DeleteDiscordIntegration deletes a Discord integration by ID
func (h *DashboardAPIHandler) DeleteDiscordIntegration(ctx context.Context, integrationID string) error {
	log.Printf("🗑️ Deleting Discord integration: %s", integrationID)
	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return fmt.Errorf("organization not found in context")
	}
	if err := h.discordIntegrationsService.DeleteDiscordIntegration(ctx, models.OrgID(org.ID), integrationID); err != nil {
		log.Printf("❌ Failed to delete Discord integration: %v", err)
		return err
	}

	log.Printf("✅ Discord integration deleted successfully: %s", integrationID)
	return nil
}

// GenerateCCAgentSecretKey generates a new secret key for an organization
func (h *DashboardAPIHandler) GenerateCCAgentSecretKey(ctx context.Context) (string, error) {
	org, ok := appctx.GetOrganization(ctx)
	if !ok {
		return "", fmt.Errorf("organization not found in context")
	}

	log.Printf("🔑 Generating CCAgent secret key for organization: %s", org.ID)

	secretKey, err := h.organizationsService.GenerateCCAgentSecretKey(ctx, models.OrgID(org.ID))
	if err != nil {
		log.Printf("❌ Failed to generate CCAgent secret key: %v", err)
		return "", err
	}

	// Disconnect all active agents since the API key has changed
	log.Printf("🔌 Disconnecting all active agents for organization: %s", org.ID)
	if err := h.agentsService.DisconnectAllActiveAgentsByOrganization(ctx); err != nil {
		log.Printf("❌ Failed to disconnect agents after API key regeneration: %v", err)
		return "", fmt.Errorf("API key generated but failed to disconnect agents: %w", err)
	}
	
	log.Printf("✅ All agents disconnected successfully after API key regeneration")
	log.Printf("✅ CCAgent secret key generated successfully for organization: %s", org.ID)
	return secretKey, nil
}
