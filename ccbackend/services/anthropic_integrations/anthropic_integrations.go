package anthropic_integrations

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type AnthropicIntegrationsService struct {
	anthropicRepo *db.PostgresAnthropicIntegrationsRepository
}

func NewAnthropicIntegrationsService(
	repo *db.PostgresAnthropicIntegrationsRepository,
) *AnthropicIntegrationsService {
	return &AnthropicIntegrationsService{
		anthropicRepo: repo,
	}
}

func (s *AnthropicIntegrationsService) CreateAnthropicIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	apiKey, oauthToken *string,
) (*models.AnthropicIntegration, error) {
	log.Printf("📋 Starting to create Anthropic integration for org: %s", organizationID)

	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	// Validate exactly one token type is provided
	if apiKey == nil && oauthToken == nil {
		return nil, fmt.Errorf("either API key or OAuth token must be provided")
	}
	if apiKey != nil && oauthToken != nil {
		return nil, fmt.Errorf("only one of API key or OAuth token can be provided")
	}
	if apiKey != nil && *apiKey == "" {
		return nil, fmt.Errorf("API key cannot be empty")
	}
	if oauthToken != nil && *oauthToken == "" {
		return nil, fmt.Errorf("OAuth token cannot be empty")
	}

	// Create the integration
	integration := &models.AnthropicIntegration{
		ID:                   core.NewID("ai"),
		AnthropicAPIKey:      apiKey,
		ClaudeCodeOAuthToken: oauthToken,
		OrgID:                organizationID,
	}

	if err := s.anthropicRepo.CreateAnthropicIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create Anthropic integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created Anthropic integration with ID: %s", integration.ID)
	return integration, nil
}

func (s *AnthropicIntegrationsService) ListAnthropicIntegrations(
	ctx context.Context,
	organizationID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	log.Printf("📋 Starting to list Anthropic integrations for org: %s", organizationID)
	if organizationID == "" {
		return nil, fmt.Errorf("organization ID cannot be empty")
	}

	integrations, err := s.anthropicRepo.GetAnthropicIntegrationsByOrganizationID(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Anthropic integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Anthropic integrations", len(integrations))
	return integrations, nil
}

func (s *AnthropicIntegrationsService) GetAnthropicIntegrationByID(
	ctx context.Context,
	organizationID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	log.Printf("📋 Starting to get Anthropic integration by ID: %s for org: %s", id, organizationID)
	if organizationID == "" {
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("organization ID cannot be empty")
	}
	if !core.IsValidULID(id) {
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	maybeInt, err := s.anthropicRepo.GetAnthropicIntegrationByID(ctx, organizationID, id)
	if err != nil {
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("failed to get Anthropic integration: %w", err)
	}

	if maybeInt.IsPresent() {
		log.Printf("📋 Completed successfully - found Anthropic integration: %s", id)
	} else {
		log.Printf("📋 Completed successfully - Anthropic integration not found: %s", id)
	}

	return maybeInt, nil
}

func (s *AnthropicIntegrationsService) DeleteAnthropicIntegration(
	ctx context.Context,
	organizationID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete Anthropic integration: %s for org: %s", integrationID, organizationID)

	if organizationID == "" {
		return fmt.Errorf("organization ID cannot be empty")
	}
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	if err := s.anthropicRepo.DeleteAnthropicIntegration(ctx, organizationID, integrationID); err != nil {
		return fmt.Errorf("failed to delete Anthropic integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted Anthropic integration: %s", integrationID)
	return nil
}
