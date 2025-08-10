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

// OrganizationsService defines the interface for organization-related operations
type OrganizationsService interface {
	CreateOrganization(ctx context.Context) (*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (mo.Option[*models.Organization], error)
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
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

// DiscordIntegrationsService defines the interface for Discord integration operations
type DiscordIntegrationsService interface {
	CreateDiscordIntegration(
		ctx context.Context,
		organizationID, discordAuthCode, guildID, redirectURL string,
	) (*models.DiscordIntegration, error)
	GetDiscordIntegrationsByOrganizationID(
		ctx context.Context,
		organizationID string,
	) ([]*models.DiscordIntegration, error)
	GetAllDiscordIntegrations(ctx context.Context) ([]*models.DiscordIntegration, error)
	DeleteDiscordIntegration(ctx context.Context, organizationID, integrationID string) error
	GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error)
	GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error)
}

// AgentsService defines the interface for agent-related operations
type AgentsService interface {
	UpsertActiveAgent(
		ctx context.Context,
		wsConnectionID, organizationID string,
		agentID string,
	) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(ctx context.Context, wsConnectionID, organizationID string) error
	DeleteActiveAgent(ctx context.Context, id string, organizationID string) error
	GetAgentByID(ctx context.Context, id string, organizationID string) (mo.Option[*models.ActiveAgent], error)
	GetAvailableAgents(ctx context.Context, organizationID string) ([]*models.ActiveAgent, error)
	GetConnectedActiveAgents(
		ctx context.Context,
		organizationID string,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	GetConnectedAvailableAgents(
		ctx context.Context,
		organizationID string,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool
	AssignAgentToJob(ctx context.Context, agentID, jobID string, organizationID string) error
	UnassignAgentFromJob(ctx context.Context, agentID, jobID string, organizationID string) error
	GetAgentByJobID(
		ctx context.Context,
		jobID string,
		organizationID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetAgentByWSConnectionID(
		ctx context.Context,
		wsConnectionID, organizationID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetActiveAgentJobAssignments(ctx context.Context, agentID string, organizationID string) ([]string, error)
	UpdateAgentLastActiveAt(ctx context.Context, wsConnectionID, organizationID string) error
	GetInactiveAgents(
		ctx context.Context,
		organizationID string,
		inactiveThresholdMinutes int,
	) ([]*models.ActiveAgent, error)
}

// JobsService defines the interface for job-related operations
type JobsService interface {
	GetActiveMessageCountForJobs(
		ctx context.Context,
		jobIDs []string,
		slackIntegrationID string,
		organizationID string,
	) (int, error)
	CreateJob(
		ctx context.Context,
		slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID string,
	) (*models.Job, error)
	GetJobByID(
		ctx context.Context,
		id string,
		organizationID string,
	) (mo.Option[*models.Job], error)
	GetJobBySlackThread(
		ctx context.Context,
		threadTS, channelID, slackIntegrationID, organizationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForSlackThread(
		ctx context.Context,
		threadTS, channelID, slackUserID, slackIntegrationID, organizationID string,
	) (*models.JobCreationResult, error)
	UpdateJobTimestamp(ctx context.Context, jobID string, slackIntegrationID string, organizationID string) error
	GetIdleJobs(ctx context.Context, idleMinutes int, organizationID string) ([]*models.Job, error)
	DeleteJob(ctx context.Context, id string, slackIntegrationID string, organizationID string) error
	GetJobsWithQueuedMessages(
		ctx context.Context,
		slackIntegrationID string,
		organizationID string,
	) ([]*models.Job, error)
	CreateProcessedSlackMessage(
		ctx context.Context,
		jobID string,
		slackChannelID, slackTS, textContent, slackIntegrationID, organizationID string,
		status models.ProcessedSlackMessageStatus,
	) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(
		ctx context.Context,
		id string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
		organizationID string,
	) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		jobID string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
		organizationID string,
	) ([]*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(
		ctx context.Context,
		id string,
		organizationID string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		jobID string,
		slackIntegrationID string,
		organizationID string,
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
