package slackintegrations

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"github.com/samber/mo"

	"ccbackend/appctx"
	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type SlackIntegrationsService struct {
	slackIntegrationsRepo *db.PostgresSlackIntegrationsRepository
	slackClient           clients.SlackClient
	slackClientID         string
	slackClientSecret     string
}

func NewSlackIntegrationsService(repo *db.PostgresSlackIntegrationsRepository, slackClient clients.SlackClient, slackClientID, slackClientSecret string) *SlackIntegrationsService {
	return &SlackIntegrationsService{
		slackIntegrationsRepo: repo,
		slackClient:           slackClient,
		slackClientID:         slackClientID,
		slackClientSecret:     slackClientSecret,
	}
}

func (s *SlackIntegrationsService) CreateSlackIntegration(ctx context.Context, slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error) {
	log.Printf("📋 Starting to create Slack integration for user: %s", userID)
	if slackAuthCode == "" {
		return nil, fmt.Errorf("slack auth code cannot be empty")
	}
	if !core.IsValidULID(userID) {
		return nil, fmt.Errorf("user ID must be a valid ULID")
	}

	// Exchange OAuth code for access token using Slack client
	oauthResponse, err := s.slackClient.GetOAuthV2Response(&http.Client{}, s.slackClientID, s.slackClientSecret, slackAuthCode, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth code with Slack: %w", err)
	}

	// Extract team information from OAuth response
	teamID := oauthResponse.TeamID
	teamName := oauthResponse.TeamName
	botAccessToken := oauthResponse.AccessToken

	if teamID == "" {
		return nil, fmt.Errorf("team ID not found in Slack OAuth response")
	}
	if teamName == "" {
		return nil, fmt.Errorf("team name not found in Slack OAuth response")
	}
	if botAccessToken == "" {
		return nil, fmt.Errorf("bot access token not found in Slack OAuth response")
	}

	integration := &models.SlackIntegration{
		ID:             core.NewID("si"),
		SlackTeamID:    teamID,
		SlackAuthToken: botAccessToken,
		SlackTeamName:  teamName,
		UserID:         userID,
	}
	if err := s.slackIntegrationsRepo.CreateSlackIntegration(ctx, integration); err != nil {
		return nil, fmt.Errorf("failed to create slack integration in database: %w", err)
	}

	log.Printf("📋 Completed successfully - created Slack integration with ID: %s for team: %s", integration.ID, teamName)
	return integration, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationsByUserID(ctx context.Context, userID string) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get Slack integrations for user: %s", userID)
	if !core.IsValidULID(userID) {
		return nil, fmt.Errorf("user ID must be a valid ULID")
	}

	integrations, err := s.slackIntegrationsRepo.GetSlackIntegrationsByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integrations for user: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Slack integrations for user: %s", len(integrations), userID)
	return integrations, nil
}

func (s *SlackIntegrationsService) GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get all Slack integrations")
	integrations, err := s.slackIntegrationsRepo.GetAllSlackIntegrations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get all slack integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Slack integrations", len(integrations))
	return integrations, nil
}

func (s *SlackIntegrationsService) DeleteSlackIntegration(ctx context.Context, integrationID string) error {
	log.Printf("📋 Starting to delete Slack integration: %s", integrationID)
	if !core.IsValidULID(integrationID) {
		return fmt.Errorf("integration ID must be a valid ULID")
	}

	user, ok := appctx.GetUser(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	deleted, err := s.slackIntegrationsRepo.DeleteSlackIntegrationByID(ctx, integrationID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete slack integration: %w", err)
	}
	if !deleted {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - deleted Slack integration: %s", integrationID)
	return nil
}

func (s *SlackIntegrationsService) GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error) {
	log.Printf("📋 Starting to generate CCAgent secret key for integration: %s", integrationID)
	if !core.IsValidULID(integrationID) {
		return "", fmt.Errorf("integration ID must be a valid ULID")
	}

	user, ok := appctx.GetUser(ctx)
	if !ok {
		return "", fmt.Errorf("user not found in context")
	}

	// Generate cryptographically secure random secret key (32 bytes = 256 bits)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return "", fmt.Errorf("failed to generate random secret key: %w", err)
	}

	// Encode as base64 for easier handling
	secretKey := base64.URLEncoding.EncodeToString(secretBytes)

	// Store the secret key in the database
	updated, err := s.slackIntegrationsRepo.GenerateCCAgentSecretKey(ctx, integrationID, user.ID, secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to store CCAgent secret key: %w", err)
	}
	if !updated {
		return "", core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - generated CCAgent secret key for integration: %s", integrationID)
	return secretKey, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.SlackIntegration], error) {
	log.Printf("📋 Starting to get slack integration by secret key")
	integrationOpt, err := s.slackIntegrationsRepo.GetSlackIntegrationBySecretKey(ctx, secretKey)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by secret key: %v", err)
		return mo.None[*models.SlackIntegration](), fmt.Errorf("failed to get slack integration by secret key: %w", err)
	}

	if !integrationOpt.IsPresent() {
		log.Printf("📋 Completed successfully - slack integration not found")
		return mo.None[*models.SlackIntegration](), nil
	}

	integration := integrationOpt.MustGet()
	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return mo.Some(integration), nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (mo.Option[*models.SlackIntegration], error) {
	log.Printf("📋 Starting to get slack integration by team ID: %s", teamID)
	if teamID == "" {
		return mo.None[*models.SlackIntegration](), fmt.Errorf("team ID cannot be empty")
	}

	integrationOpt, err := s.slackIntegrationsRepo.GetSlackIntegrationByTeamID(ctx, teamID)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by team ID: %v", err)
		return mo.None[*models.SlackIntegration](), fmt.Errorf("failed to get slack integration by team ID: %w", err)
	}

	if !integrationOpt.IsPresent() {
		log.Printf("📋 Completed successfully - slack integration not found")
		return mo.None[*models.SlackIntegration](), nil
	}

	integration := integrationOpt.MustGet()
	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return mo.Some(integration), nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationByID(ctx context.Context, id string) (mo.Option[*models.SlackIntegration], error) {
	log.Printf("📋 Starting to get slack integration by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.SlackIntegration](), fmt.Errorf("integration ID must be a valid ULID")
	}

	integrationOpt, err := s.slackIntegrationsRepo.GetSlackIntegrationByID(ctx, id)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by ID: %v", err)
		return mo.None[*models.SlackIntegration](), fmt.Errorf("failed to get slack integration by ID: %w", err)
	}

	if !integrationOpt.IsPresent() {
		log.Printf("📋 Completed successfully - slack integration not found")
		return mo.None[*models.SlackIntegration](), nil
	}

	integration := integrationOpt.MustGet()
	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return mo.Some(integration), nil
}
