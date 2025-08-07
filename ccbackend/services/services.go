package services

import (
	"context"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UsersService defines the interface for user-related operations
type UsersService interface {
	GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(
		ctx context.Context,
		slackAuthCode, redirectURL string,
		userID string,
	) (*models.SlackIntegration, error)
	GetSlackIntegrationsByUserID(ctx context.Context, userID string) ([]*models.SlackIntegration, error)
	GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, integrationID string) error
	GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error)
	GetSlackIntegrationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.SlackIntegration], error)
	GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (mo.Option[*models.SlackIntegration], error)
	GetSlackIntegrationByID(ctx context.Context, id string) (mo.Option[*models.SlackIntegration], error)
}

// AgentsService defines the interface for agent-related operations
type AgentsService interface {
	UpsertActiveAgent(
		ctx context.Context,
		wsConnectionID, slackIntegrationID string,
		agentID string,
	) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(ctx context.Context, wsConnectionID, slackIntegrationID string) error
	DeleteActiveAgent(ctx context.Context, id string, slackIntegrationID string) error
	GetAgentByID(ctx context.Context, id string, slackIntegrationID string) (mo.Option[*models.ActiveAgent], error)
	GetAvailableAgents(ctx context.Context, slackIntegrationID string) ([]*models.ActiveAgent, error)
	GetConnectedActiveAgents(
		ctx context.Context,
		slackIntegrationID string,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	GetConnectedAvailableAgents(
		ctx context.Context,
		slackIntegrationID string,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool
	AssignAgentToJob(ctx context.Context, agentID, jobID string, slackIntegrationID string) error
	UnassignAgentFromJob(ctx context.Context, agentID, jobID string, slackIntegrationID string) error
	GetAgentByJobID(
		ctx context.Context,
		jobID string,
		slackIntegrationID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetAgentByWSConnectionID(
		ctx context.Context,
		wsConnectionID, slackIntegrationID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetActiveAgentJobAssignments(ctx context.Context, agentID string, slackIntegrationID string) ([]string, error)
	UpdateAgentLastActiveAt(ctx context.Context, wsConnectionID, slackIntegrationID string) error
	GetInactiveAgents(
		ctx context.Context,
		slackIntegrationID string,
		inactiveThresholdMinutes int,
	) ([]*models.ActiveAgent, error)
}

// TransactionManager handles database transactions via context
type TransactionManager interface {
	// Execute function within a transaction (recommended approach)
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error

	// Manual transaction control (for complex scenarios)
	BeginTransaction(ctx context.Context) (context.Context, error)
	CommitTransaction(ctx context.Context) error
	RollbackTransaction(ctx context.Context) error
}
