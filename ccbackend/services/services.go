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
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GenerateCCAgentSecretKey(ctx context.Context, organizationID models.OrgID) (string, error)
	GetOrganizationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.Organization], error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(
		ctx context.Context,
		organizationID models.OrgID, slackAuthCode, redirectURL string,
	) (*models.SlackIntegration, error)
	GetSlackIntegrationsByOrganizationID(
		ctx context.Context,
		organizationID models.OrgID,
	) ([]*models.SlackIntegration, error)
	GetAllSlackIntegrations(ctx context.Context) ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, organizationID models.OrgID, integrationID string) error
	GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (mo.Option[*models.SlackIntegration], error)
	GetSlackIntegrationByID(ctx context.Context, id string) (mo.Option[*models.SlackIntegration], error)
}

// DiscordIntegrationsService defines the interface for Discord integration operations
type DiscordIntegrationsService interface {
	CreateDiscordIntegration(
		ctx context.Context,
		organizationID models.OrgID, discordAuthCode, guildID, redirectURL string,
	) (*models.DiscordIntegration, error)
	GetDiscordIntegrationsByOrganizationID(
		ctx context.Context,
		organizationID models.OrgID,
	) ([]*models.DiscordIntegration, error)
	GetAllDiscordIntegrations(ctx context.Context) ([]*models.DiscordIntegration, error)
	DeleteDiscordIntegration(ctx context.Context, organizationID models.OrgID, integrationID string) error
	GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error)
	GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error)
}

// AgentsService defines the interface for agent-related operations
type AgentsService interface {
	UpsertActiveAgent(
		ctx context.Context,
		wsConnectionID string, organizationID models.OrgID,
		agentID string,
	) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(ctx context.Context, wsConnectionID string, organizationID models.OrgID) error
	DeleteActiveAgent(ctx context.Context, id string, organizationID models.OrgID) error
	GetAgentByID(ctx context.Context, id string, organizationID models.OrgID) (mo.Option[*models.ActiveAgent], error)
	GetAvailableAgents(ctx context.Context, organizationID models.OrgID) ([]*models.ActiveAgent, error)
	GetConnectedActiveAgents(
		ctx context.Context,
		organizationID models.OrgID,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	GetConnectedAvailableAgents(
		ctx context.Context,
		organizationID models.OrgID,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool
	AssignAgentToJob(ctx context.Context, agentID, jobID string, organizationID models.OrgID) error
	UnassignAgentFromJob(ctx context.Context, agentID, jobID string, organizationID models.OrgID) error
	GetAgentByJobID(
		ctx context.Context,
		jobID string,
		organizationID models.OrgID,
	) (mo.Option[*models.ActiveAgent], error)
	GetAgentByWSConnectionID(
		ctx context.Context,
		wsConnectionID string, organizationID models.OrgID,
	) (mo.Option[*models.ActiveAgent], error)
	GetActiveAgentJobAssignments(ctx context.Context, agentID string, organizationID models.OrgID) ([]string, error)
	UpdateAgentLastActiveAt(ctx context.Context, wsConnectionID string, organizationID models.OrgID) error
	GetInactiveAgents(
		ctx context.Context,
		organizationID models.OrgID,
		inactiveThresholdMinutes int,
	) ([]*models.ActiveAgent, error)
}

// SlackMessagesService defines the interface for processed slack message operations
type SlackMessagesService interface {
	CreateProcessedSlackMessage(
		ctx context.Context,
		jobID string,
		slackChannelID, slackTS, textContent, slackIntegrationID string,
		organizationID models.OrgID,
		status models.ProcessedSlackMessageStatus,
	) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(
		ctx context.Context,
		id string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
		organizationID models.OrgID,
	) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		jobID string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
		organizationID models.OrgID,
	) ([]*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(
		ctx context.Context,
		id string,
		organizationID models.OrgID,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		jobID string,
		slackIntegrationID string,
		organizationID models.OrgID,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		jobIDs []string,
		slackIntegrationID string,
		organizationID models.OrgID,
	) (int, error)
	TESTS_UpdateProcessedSlackMessageUpdatedAt(
		ctx context.Context,
		id string,
		updatedAt time.Time,
		slackIntegrationID string,
		organizationID models.OrgID,
	) error
	DeleteProcessedSlackMessagesByJobID(
		ctx context.Context,
		jobID string,
		slackIntegrationID string,
		organizationID models.OrgID,
	) error
}

// DiscordMessagesService defines the interface for processed discord message operations
type DiscordMessagesService interface {
	CreateProcessedDiscordMessage(
		ctx context.Context,
		jobID string,
		discordMessageID, discordThreadID, textContent, discordIntegrationID string,
		organizationID models.OrgID,
		status models.ProcessedDiscordMessageStatus,
	) (*models.ProcessedDiscordMessage, error)
	UpdateProcessedDiscordMessage(
		ctx context.Context,
		id string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
		organizationID models.OrgID,
	) (*models.ProcessedDiscordMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		jobID string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
		organizationID models.OrgID,
	) ([]*models.ProcessedDiscordMessage, error)
	GetProcessedDiscordMessageByID(
		ctx context.Context,
		id string,
		organizationID models.OrgID,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		jobID string,
		discordIntegrationID string,
		organizationID models.OrgID,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		jobIDs []string,
		discordIntegrationID string,
		organizationID models.OrgID,
	) (int, error)
	TESTS_UpdateProcessedDiscordMessageUpdatedAt(
		ctx context.Context,
		id string,
		updatedAt time.Time,
		discordIntegrationID string,
		organizationID models.OrgID,
	) error
	DeleteProcessedDiscordMessagesByJobID(
		ctx context.Context,
		jobID string,
		discordIntegrationID string,
		organizationID models.OrgID,
	) error
}

// JobsService defines the interface for job-related operations
type JobsService interface {
	GetJobByID(
		ctx context.Context,
		id string,
		organizationID models.OrgID,
	) (mo.Option[*models.Job], error)
	UpdateJobTimestamp(ctx context.Context, jobID string, organizationID models.OrgID) error
	GetIdleJobs(ctx context.Context, idleMinutes int, organizationID models.OrgID) ([]*models.Job, error)
	DeleteJob(ctx context.Context, id string, organizationID models.OrgID) error
	GetJobsWithQueuedMessages(
		ctx context.Context,
		jobType models.JobType,
		integrationID string,
		organizationID models.OrgID,
	) ([]*models.Job, error)

	// Slack-specific methods
	CreateSlackJob(
		ctx context.Context,
		slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
		organizationID models.OrgID,
	) (*models.Job, error)
	GetJobBySlackThread(
		ctx context.Context,
		threadTS, channelID, slackIntegrationID string,
		organizationID models.OrgID,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForSlackThread(
		ctx context.Context,
		threadTS, channelID, slackUserID, slackIntegrationID string,
		organizationID models.OrgID,
	) (*models.JobCreationResult, error)

	// Discord-specific methods
	CreateDiscordJob(
		ctx context.Context,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
		organizationID models.OrgID,
	) (*models.Job, error)
	GetJobByDiscordThread(
		ctx context.Context,
		threadID, discordIntegrationID string,
		organizationID models.OrgID,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForDiscordThread(
		ctx context.Context,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
		organizationID models.OrgID,
	) (*models.JobCreationResult, error)
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
