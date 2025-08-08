package services

import (
	"context"
	"time"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UsersService defines the interface for user-related operations
type UsersService interface {
	GetOrCreateUser(ctx context.Context, authProvider, authProviderID string) (*models.User, error)
}

// OrganizationsService defines the interface for organization-related operations
type OrganizationsService interface {
	CreateOrganization(ctx context.Context) (*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (mo.Option[*models.Organization], error)
	GenerateCCAgentSecretKey(ctx context.Context, organizationID string) (string, error)
	GetOrganizationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.Organization], error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(
		ctx context.Context,
		organizationID, slackAuthCode, redirectURL string,
	) (*models.SlackIntegration, error)
	GetSlackIntegrationsByOrganizationID(ctx context.Context, organizationID string) ([]*models.SlackIntegration, error)
	GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, organizationID, integrationID string) error
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

// JobsService defines the interface for job-related operations
type JobsService interface {
	GetActiveMessageCountForJobs(ctx context.Context, jobIDs []string, slackIntegrationID string) (int, error)
	CreateJob(
		ctx context.Context,
		slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
	) (*models.Job, error)
	GetJobByID(ctx context.Context, id string, slackIntegrationID string) (mo.Option[*models.Job], error)
	GetJobBySlackThread(
		ctx context.Context,
		threadTS, channelID, slackIntegrationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForSlackThread(
		ctx context.Context,
		threadTS, channelID, slackUserID, slackIntegrationID string,
	) (*models.JobCreationResult, error)
	UpdateJobTimestamp(ctx context.Context, jobID string, slackIntegrationID string) error
	GetIdleJobs(ctx context.Context, idleMinutes int) ([]*models.Job, error)
	DeleteJob(ctx context.Context, id string, slackIntegrationID string) error
	CreateProcessedSlackMessage(
		ctx context.Context,
		jobID string,
		slackChannelID, slackTS, textContent, slackIntegrationID string,
		status models.ProcessedSlackMessageStatus,
	) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(
		ctx context.Context,
		id string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		jobID string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) ([]*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(
		ctx context.Context,
		id string,
		slackIntegrationID string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	TESTS_UpdateJobUpdatedAt(ctx context.Context, id string, updatedAt time.Time, slackIntegrationID string) error
	TESTS_UpdateProcessedSlackMessageUpdatedAt(
		ctx context.Context,
		id string,
		updatedAt time.Time,
		slackIntegrationID string,
	) error
	GetJobsWithQueuedMessages(ctx context.Context, slackIntegrationID string) ([]*models.Job, error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		jobID string,
		slackIntegrationID string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
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
