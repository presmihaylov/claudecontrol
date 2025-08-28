package anthropic_integrations

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

type AnthropicIntegrationsService struct {
	anthropicRepo   *db.PostgresAnthropicIntegrationsRepository
	anthropicClient clients.AnthropicClient
}

func NewAnthropicIntegrationsService(
	repo *db.PostgresAnthropicIntegrationsRepository,
	anthropicClient clients.AnthropicClient,
) *AnthropicIntegrationsService {
	return &AnthropicIntegrationsService{
		anthropicRepo:   repo,
		anthropicClient: anthropicClient,
	}
}

func (s *AnthropicIntegrationsService) CreateAnthropicIntegration(
	ctx context.Context,
	orgID models.OrgID,
	apiKey, oauthToken, codeVerifier *string,
) (*models.AnthropicIntegration, error) {
	log.Printf("📋 Starting to create Anthropic integration for org: %s", orgID)

	if !core.IsValidULID(orgID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}

	// Validate exactly one authentication method is provided
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
		ID:              core.NewID("ai"),
		AnthropicAPIKey: apiKey,
		OrgID:           orgID,
	}

	// If OAuth token provided, exchange it for access and refresh tokens
	if oauthToken != nil {
		if codeVerifier == nil || *codeVerifier == "" {
			return nil, fmt.Errorf("code verifier is required for OAuth token exchange")
		}

		log.Printf("📋 Exchanging OAuth code for tokens")
		tokens, err := s.anthropicClient.ExchangeCodeForTokens(ctx, *oauthToken, *codeVerifier)
		if err != nil {
			return nil, fmt.Errorf("failed to exchange OAuth code for tokens: %w", err)
		}

		// Store the OAuth tokens
		integration.ClaudeCodeOAuthToken = &tokens.AccessToken
		integration.ClaudeCodeOAuthRefreshToken = &tokens.RefreshToken
		integration.ClaudeCodeOAuthTokenExpiresAt = &tokens.ExpiresAt

		log.Printf("📋 Successfully exchanged OAuth code for tokens")
	}

	if err := s.anthropicRepo.CreateAnthropicIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create Anthropic integration: %w", err)
	}

	log.Printf("📋 Completed successfully - created Anthropic integration with ID: %s", integration.ID)
	return integration, nil
}

func (s *AnthropicIntegrationsService) ListAnthropicIntegrations(
	ctx context.Context,
	orgID models.OrgID,
) ([]models.AnthropicIntegration, error) {
	log.Printf("📋 Starting to list Anthropic integrations for org: %s", orgID)
	if !core.IsValidULID(orgID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}

	integrations, err := s.anthropicRepo.GetAnthropicIntegrationsByOrganizationID(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to list Anthropic integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Anthropic integrations", len(integrations))
	return integrations, nil
}

func (s *AnthropicIntegrationsService) GetAnthropicIntegrationByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.AnthropicIntegration], error) {
	log.Printf("📋 Starting to get Anthropic integration by ID: %s for org: %s", id, orgID)
	if !core.IsValidULID(orgID) {
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("organization ID must be a valid ULID")
	}
	if !core.IsValidULID(id) {
		return mo.None[*models.AnthropicIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	maybeInt, err := s.anthropicRepo.GetAnthropicIntegrationByID(ctx, orgID, id)
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
	orgID models.OrgID,
	integrationID string,
) error {
	log.Printf("📋 Starting to delete Anthropic integration: %s for org: %s", integrationID, orgID)

	if !core.IsValidULID(orgID) {
		return fmt.Errorf("organization ID must be a valid ULID")
	}
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	if err := s.anthropicRepo.DeleteAnthropicIntegration(ctx, orgID, integrationID); err != nil {
		return fmt.Errorf("failed to delete Anthropic integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted Anthropic integration: %s", integrationID)
	return nil
}

func (s *AnthropicIntegrationsService) RefreshTokens(
	ctx context.Context,
	orgID models.OrgID,
	integrationID string,
) (*models.AnthropicIntegration, error) {
	log.Printf("📋 Starting to refresh tokens for Anthropic integration: %s in org: %s", integrationID, orgID)

	if !core.IsValidULID(orgID) {
		return nil, fmt.Errorf("organization ID must be a valid ULID")
	}
	if !core.IsValidULID(integrationID) {
		return nil, fmt.Errorf("integration ID must be a valid ULID")
	}

	// Get the current integration
	maybeIntegration, err := s.anthropicRepo.GetAnthropicIntegrationByID(ctx, orgID, integrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Anthropic integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return nil, fmt.Errorf("anthropic integration not found")
	}

	integration := maybeIntegration.MustGet()

	// Check if we have a refresh token
	if integration.ClaudeCodeOAuthRefreshToken == nil || *integration.ClaudeCodeOAuthRefreshToken == "" {
		return nil, fmt.Errorf("no refresh token available for this integration")
	}

	// Refresh the tokens
	log.Printf("📋 Refreshing OAuth tokens using refresh token")
	newTokens, err := s.anthropicClient.RefreshAccessToken(ctx, *integration.ClaudeCodeOAuthRefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh tokens: %w", err)
	}

	// Update the integration with new tokens
	integration.ClaudeCodeOAuthToken = &newTokens.AccessToken
	integration.ClaudeCodeOAuthRefreshToken = &newTokens.RefreshToken
	integration.ClaudeCodeOAuthTokenExpiresAt = &newTokens.ExpiresAt

	// Save the updated integration (we need an update method in the repository)
	if err := s.anthropicRepo.UpdateAnthropicIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to update Anthropic integration with new tokens: %w", err)
	}

	log.Printf("📋 Completed successfully - refreshed tokens for Anthropic integration: %s", integrationID)
	return integration, nil
}
