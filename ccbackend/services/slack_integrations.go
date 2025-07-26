package services

import (
	"fmt"
	"log"
	"net/http"

	"ccbackend/db"
	"ccbackend/models"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

type SlackIntegrationsService struct {
	slackIntegrationsRepo *db.PostgresSlackIntegrationsRepository
	slackClientID         string
	slackClientSecret     string
}

func NewSlackIntegrationsService(repo *db.PostgresSlackIntegrationsRepository, slackClientID, slackClientSecret string) *SlackIntegrationsService {
	return &SlackIntegrationsService{
		slackIntegrationsRepo: repo,
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

	// Exchange OAuth code for access token using Slack SDK
	oauthResponse, err := slack.GetOAuthV2Response(&http.Client{}, s.slackClientID, s.slackClientSecret, slackAuthCode, redirectURL)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange OAuth code with Slack: %w", err)
	}

	// Extract team information from OAuth response
	teamID := oauthResponse.Team.ID
	teamName := oauthResponse.Team.Name
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