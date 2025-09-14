package discordintegrations

import (
	"context"
	"fmt"

	"github.com/samber/mo"

	"ccbackend/models"
)

// OptionalDiscordIntegrationsService returns errors for all operations when Discord is not configured
type OptionalDiscordIntegrationsService struct{}

// NewOptionalDiscordIntegrationsService creates a new optional Discord integrations service
func NewOptionalDiscordIntegrationsService() *OptionalDiscordIntegrationsService {
	return &OptionalDiscordIntegrationsService{}
}

func (s *OptionalDiscordIntegrationsService) CreateDiscordIntegration(
	ctx context.Context,
	orgID models.OrgID,
	discordAuthCode, guildID, redirectURL string,
) (*models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *OptionalDiscordIntegrationsService) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *OptionalDiscordIntegrationsService) GetAllDiscordIntegrations(ctx context.Context) ([]models.DiscordIntegration, error) {
	return nil, fmt.Errorf("Service Discord is not configured")
}

func (s *OptionalDiscordIntegrationsService) DeleteDiscordIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error {
	return fmt.Errorf("Service Discord is not configured")
}

func (s *OptionalDiscordIntegrationsService) GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error) {
	return mo.None[*models.DiscordIntegration](), fmt.Errorf("Service Discord is not configured")
}

func (s *OptionalDiscordIntegrationsService) GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error) {
	return mo.None[*models.DiscordIntegration](), fmt.Errorf("Service Discord is not configured")
}