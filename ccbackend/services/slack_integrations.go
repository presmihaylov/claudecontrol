package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"

	"ccbackend/appctx"
	"ccbackend/clients"
	"ccbackend/db"
	"ccbackend/models"

	"github.com/google/uuid"
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

func (s *SlackIntegrationsService) CreateSlackIntegration(slackAuthCode, redirectURL string, userID uuid.UUID) (*models.SlackIntegration, error) {
	log.Printf("📋 Starting to create Slack integration for user: %s", userID)

	if slackAuthCode == "" {
		return nil, fmt.Errorf("slack auth code cannot be empty")
	}

	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be nil")
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

	// Create slack integration record
	integration := &models.SlackIntegration{
		ID:             uuid.New(),
		SlackTeamID:    teamID,
		SlackAuthToken: botAccessToken,
		SlackTeamName:  teamName,
		UserID:         userID,
	}

	if err := s.slackIntegrationsRepo.CreateSlackIntegration(integration); err != nil {
		return nil, fmt.Errorf("failed to create slack integration in database: %w", err)
	}

	log.Printf("📋 Completed successfully - created Slack integration with ID: %s for team: %s", integration.ID, teamName)
	return integration, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationsByUserID(userID uuid.UUID) ([]*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get Slack integrations for user: %s", userID)

	if userID == uuid.Nil {
		return nil, fmt.Errorf("user ID cannot be nil")
	}

	integrations, err := s.slackIntegrationsRepo.GetSlackIntegrationsByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integrations for user: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Slack integrations for user: %s", len(integrations), userID)
	return integrations, nil
}

func (s *SlackIntegrationsService) GetAllSlackIntegrations() ([]*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get all Slack integrations")

	integrations, err := s.slackIntegrationsRepo.GetAllSlackIntegrations()
	if err != nil {
		return nil, fmt.Errorf("failed to get all slack integrations: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d Slack integrations", len(integrations))
	return integrations, nil
}

func (s *SlackIntegrationsService) DeleteSlackIntegration(ctx context.Context, integrationID uuid.UUID) error {
	log.Printf("📋 Starting to delete Slack integration: %s", integrationID)

	if integrationID == uuid.Nil {
		return fmt.Errorf("integration ID cannot be nil")
	}

	// Get user from context
	user, ok := appctx.GetUser(ctx)
	if !ok {
		return fmt.Errorf("user not found in context")
	}

	if err := s.slackIntegrationsRepo.DeleteSlackIntegrationByID(integrationID, user.ID); err != nil {
		return fmt.Errorf("failed to delete slack integration: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted Slack integration: %s", integrationID)
	return nil
}

func (s *SlackIntegrationsService) GenerateCCAgentSecretKey(ctx context.Context, integrationID uuid.UUID) (string, error) {
	log.Printf("📋 Starting to generate CCAgent secret key for integration: %s", integrationID)

	if integrationID == uuid.Nil {
		return "", fmt.Errorf("integration ID cannot be nil")
	}

	// Get user from context
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
	if err := s.slackIntegrationsRepo.GenerateCCAgentSecretKey(ctx, integrationID, user.ID, secretKey); err != nil {
		return "", fmt.Errorf("failed to store CCAgent secret key: %w", err)
	}

	log.Printf("📋 Completed successfully - generated CCAgent secret key for integration: %s", integrationID)
	return secretKey, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationBySecretKey(secretKey string) (*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get slack integration by secret key")
	
	integration, err := s.slackIntegrationsRepo.GetSlackIntegrationBySecretKey(secretKey)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by secret key: %v", err)
		return nil, fmt.Errorf("failed to get slack integration by secret key: %w", err)
	}

	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return integration, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationByTeamID(teamID string) (*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get slack integration by team ID: %s", teamID)
	
	if teamID == "" {
		return nil, fmt.Errorf("team ID cannot be empty")
	}
	
	integration, err := s.slackIntegrationsRepo.GetSlackIntegrationByTeamID(teamID)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by team ID: %v", err)
		return nil, fmt.Errorf("failed to get slack integration by team ID: %w", err)
	}

	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return integration, nil
}

func (s *SlackIntegrationsService) GetSlackIntegrationByID(id uuid.UUID) (*models.SlackIntegration, error) {
	log.Printf("📋 Starting to get slack integration by ID: %s", id)
	
	if id == uuid.Nil {
		return nil, fmt.Errorf("integration ID cannot be nil")
	}
	
	integration, err := s.slackIntegrationsRepo.GetSlackIntegrationByID(id)
	if err != nil {
		log.Printf("❌ Failed to get slack integration by ID: %v", err)
		return nil, fmt.Errorf("failed to get slack integration by ID: %w", err)
	}

	log.Printf("📋 Completed successfully - found slack integration for team: %s", integration.SlackTeamName)
	return integration, nil
}