package services

import (
	"context"
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