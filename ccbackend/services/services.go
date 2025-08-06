package services

import (
	"context"

	"ccbackend/models"
)

// UsersServiceInterface defines the interface for user-related operations
type UsersServiceInterface interface {
	GetOrCreateUser(authProvider, authProviderID string) (*models.User, error)
}

// SlackIntegrationsServiceInterface defines the interface for Slack integration operations
type SlackIntegrationsServiceInterface interface {
	CreateSlackIntegration(slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error)
	GetSlackIntegrationsByUserID(userID string) ([]*models.SlackIntegration, error)
	GetAllSlackIntegrations() ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, integrationID string) error
	GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error)
	GetSlackIntegrationBySecretKey(secretKey string) (*models.SlackIntegration, error)
	GetSlackIntegrationByTeamID(teamID string) (*models.SlackIntegration, error)
	GetSlackIntegrationByID(id string) (*models.SlackIntegration, error)
}
