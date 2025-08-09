package discordintegrations

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/samber/mo"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
)

// DiscordIntegrationsRepository defines the interface for Discord integration repository operations
type DiscordIntegrationsRepository interface {
	CreateDiscordIntegration(ctx context.Context, integration *models.DiscordIntegration) error
	GetDiscordIntegrationsByOrganizationID(
		ctx context.Context,
		organizationID string,
	) ([]*models.DiscordIntegration, error)
	GetAllDiscordIntegrations(ctx context.Context) ([]*models.DiscordIntegration, error)
	DeleteDiscordIntegrationByID(ctx context.Context, integrationID, organizationID string) (bool, error)
	GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error)
	GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error)
}

type DiscordIntegrationsService struct {
	discordIntegrationsRepo DiscordIntegrationsRepository
	discordClient           clients.DiscordClient
	discordClientID         string
	discordClientSecret     string
}

func NewDiscordIntegrationsService(
	repo DiscordIntegrationsRepository,
	discordClient clients.DiscordClient,
	discordClientID, discordClientSecret string,
) *DiscordIntegrationsService {
	return &DiscordIntegrationsService{
		discordIntegrationsRepo: repo,
		discordClient:           discordClient,
		discordClientID:         discordClientID,
		discordClientSecret:     discordClientSecret,
	}
}

func (s *DiscordIntegrationsService) CreateDiscordIntegration(
	ctx context.Context,
	organizationID, discordAuthCode, guildID, redirectURL string,
) (*models.DiscordIntegration, error) {
	log.Printf("📋 Starting to create Discord integration for organization: %s", organizationID)
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}
	if discordAuthCode == "" {
		return nil, fmt.Errorf("discord auth code cannot be empty")
	}
	if guildID == "" {
		return nil, fmt.Errorf("discord guild ID cannot be empty")
	}

	// Exchange OAuth code for access token to validate the request
	oauthResponse, err := s.discordClient.ExchangeCodeForToken(
		&http.Client{},
		s.discordClientID,
		s.discordClientSecret,
		discordAuthCode,
		redirectURL,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth code with Discord: %w", err)
	}

	if oauthResponse.AccessToken == "" {
		return nil, fmt.Errorf("access token not found in Discord OAuth response")
	}

	// Fetch guild information to get the guild name
	guildInfo, err := s.discordClient.GetGuildByID(&http.Client{}, oauthResponse.AccessToken, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Discord guild information: %w", err)
	}

	if guildInfo == nil {
		return nil, fmt.Errorf("guild not found or bot not added to guild")
	}

	if guildInfo.Name == "" {
		return nil, fmt.Errorf("guild name not found in Discord API response")
	}

	integration := &models.DiscordIntegration{
		ID:               core.NewID("di"),
		DiscordGuildID:   guildID,
		DiscordAuthToken: oauthResponse.AccessToken,
		DiscordGuildName: guildInfo.Name,
		OrganizationID:   organizationID,
	}
	if err := s.discordIntegrationsRepo.CreateDiscordIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create discord integration in database: %w", err)
	}

	log.Printf(
		"📋 Completed successfully - created Discord integration with ID: %s for guild: %s",
		integration.ID,
		guildInfo.Name,
	)
	return integration, nil
}

func (s *DiscordIntegrationsService) GetDiscordIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID string,
) ([]*models.DiscordIntegration, error) {
	log.Printf("📋 Starting to get Discord integrations for organization: %s", organizationID)
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}

	integrations, err := s.discordIntegrationsRepo.GetDiscordIntegrationsByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord integrations for organization: %w", err)
	}

	log.Printf(
		"📋 Completed successfully - found %d Discord integrations for organization: %s",
		len(integrations),
		organizationID,
	)
	return integrations, nil
}

func (s *DiscordIntegrationsService) GetAllDiscordIntegrations(
	ctx context.Context,
) ([]*models.DiscordIntegration, error) {
	log.Printf("📋 Starting to get all Discord integrations")
	integrations, err := s.discordIntegrationsRepo.GetAllDiscordIntegrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all discord integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Discord integrations", len(integrations))
	return integrations, nil
}

func (s *DiscordIntegrationsService) DeleteDiscordIntegration(
	ctx context.Context,
	organizationID, integrationID string,
) error {
	log.Printf("📋 Starting to delete Discord integration: %s", integrationID)
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization ID must be a valid ULID")
	}
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	deleted, err := s.discordIntegrationsRepo.DeleteDiscordIntegrationByID(ctx, integrationID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete discord integration: %w", err)
	}
	if !deleted {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - deleted Discord integration: %s", integrationID)
	return nil
}

func (s *DiscordIntegrationsService) GetDiscordIntegrationByGuildID(
	ctx context.Context,
	guildID string,
) (mo.Option[*models.DiscordIntegration], error) {
	log.Printf("📋 Starting to get discord integration by guild ID: %s", guildID)
	if guildID == "" {
		return mo.None[*models.DiscordIntegration](), fmt.Errorf("guild ID cannot be empty")
	}

	maybeDiscordInt, err := s.discordIntegrationsRepo.GetDiscordIntegrationByGuildID(ctx, guildID)
	if err != nil {
		log.Printf("❌ Failed to get discord integration by guild ID: %v", err)
		return mo.None[*models.DiscordIntegration](), fmt.Errorf(
			"failed to get discord integration by guild ID: %w",
			err,
		)
	}

	if !maybeDiscordInt.IsPresent() {
		log.Printf("📋 Completed successfully - discord integration not found")
		return mo.None[*models.DiscordIntegration](), nil
	}

	integration := maybeDiscordInt.MustGet()
	log.Printf("📋 Completed successfully - found discord integration for guild: %s", integration.DiscordGuildName)
	return mo.Some(integration), nil
}

func (s *DiscordIntegrationsService) GetDiscordIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.DiscordIntegration], error) {
	log.Printf("📋 Starting to get discord integration by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.DiscordIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	maybeDiscordInt, err := s.discordIntegrationsRepo.GetDiscordIntegrationByID(ctx, id)
	if err != nil {
		log.Printf("❌ Failed to get discord integration by ID: %v", err)
		return mo.None[*models.DiscordIntegration](), fmt.Errorf("failed to get discord integration by ID: %w", err)
	}

	if !maybeDiscordInt.IsPresent() {
		log.Printf("📋 Completed successfully - discord integration not found")
		return mo.None[*models.DiscordIntegration](), nil
	}

	integration := maybeDiscordInt.MustGet()
	log.Printf("📋 Completed successfully - found discord integration for guild: %s", integration.DiscordGuildName)
	return mo.Some(integration), nil
}
