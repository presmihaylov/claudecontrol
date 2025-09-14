package discordintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UnconfiguredDiscordIntegrationsService returns errors for all operations when Discord is not configured
type UnconfiguredDiscordIntegrationsService struct{}

// NewUnconfiguredDiscordIntegrationsService creates a new unconfigured Discord integrations service
func NewUnconfiguredDiscordIntegrationsService() *UnconfiguredDiscordIntegrationsService {
	return &UnconfiguredDiscordIntegrationsService{}
}

func (s *UnconfiguredDiscordIntegrationsService) CreateDiscordIntegration(
	ctx context.Context,
	orgID models.OrgID,
	discordAuthCode, guildID, redirectURL string,
) (*models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *UnconfiguredDiscordIntegrationsService) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *UnconfiguredDiscordIntegrationsService) GetAllDiscordIntegrations(ctx context.Context) ([]models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *UnconfiguredDiscordIntegrationsService) DeleteDiscordIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error {
	return fmt.Errorf("Service Discord is not configured")
}

func (s *UnconfiguredDiscordIntegrationsService) GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error) {
	return mo.None[*models.DiscordIntegration](), fmt.Errorf("Service Discord is not configured")
}

func (s *UnconfiguredDiscordIntegrationsService) GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error) {
	return mo.None[*models.DiscordIntegration](), fmt.Errorf("Service Discord is not configured")
}