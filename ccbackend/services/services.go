package services

import (
	"context"
	"time"

	"github.com/samber/mo"

	"ccbackend/models"
)

// UsersService defines the interface for user-related operations
type UsersService interface {
	GetOrCreateUser(ctx context.Context, authProvider, authProviderID, email string) (*models.User, error)
}

// OrganizationsService defines the interface for organization-related operations
type OrganizationsService interface {
	CreateOrganization(ctx context.Context) (*models.Organization, error)
	GetOrganizationByID(ctx context.Context, id string) (mo.Option[*models.Organization], error)
	GetAllOrganizations(ctx context.Context) ([]*models.Organization, error)
	GenerateCCAgentSecretKey(ctx context.Context, organizationID models.OrgID) (string, error)
	GetOrganizationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.Organization], error)
	GetOrganizationBySystemSecretKey(
		ctx context.Context,
		systemSecretKey string,
	) (mo.Option[*models.Organization], error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(
		ctx context.Context,
		organizationID models.OrgID,
		slackAuthCode, redirectURL string,
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
		organizationID models.OrgID,
		discordAuthCode, guildID, redirectURL string,
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
		organizationID models.OrgID,
		wsConnectionID string,
		agentID string,
	) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(ctx context.Context, organizationID models.OrgID, wsConnectionID string) error
	DeleteActiveAgent(ctx context.Context, organizationID models.OrgID, id string) error
	GetAgentByID(ctx context.Context, organizationID models.OrgID, id string) (mo.Option[*models.ActiveAgent], error)
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
	AssignAgentToJob(ctx context.Context, organizationID models.OrgID, agentID, jobID string) error
	UnassignAgentFromJob(ctx context.Context, organizationID models.OrgID, agentID, jobID string) error
	GetAgentByJobID(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetAgentByWSConnectionID(
		ctx context.Context,
		organizationID models.OrgID,
		wsConnectionID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetActiveAgentJobAssignments(ctx context.Context, organizationID models.OrgID, agentID string) ([]string, error)
	UpdateAgentLastActiveAt(ctx context.Context, organizationID models.OrgID, wsConnectionID string) error
	GetInactiveAgents(
		ctx context.Context,
		organizationID models.OrgID,
		inactiveThresholdMinutes int,
	) ([]*models.ActiveAgent, error)
	DisconnectAllActiveAgentsByOrganization(ctx context.Context, organizationID models.OrgID) error
}

// SlackMessagesService defines the interface for processed slack message operations
type SlackMessagesService interface {
	CreateProcessedSlackMessage(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		slackChannelID, slackTS, textContent, slackIntegrationID string,
		status models.ProcessedSlackMessageStatus,
	) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) ([]*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		slackIntegrationID string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		organizationID models.OrgID,
		jobIDs []string,
		slackIntegrationID string,
	) (int, error)
	TESTS_UpdateProcessedSlackMessageUpdatedAt(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
		updatedAt time.Time,
		slackIntegrationID string,
	) error
	DeleteProcessedSlackMessagesByJobID(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		slackIntegrationID string,
	) error
	GetProcessedMessagesByStatus(
		ctx context.Context,
		organizationID models.OrgID,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) ([]*models.ProcessedSlackMessage, error)
}

// DiscordMessagesService defines the interface for processed discord message operations
type DiscordMessagesService interface {
	CreateProcessedDiscordMessage(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		discordMessageID, discordThreadID, textContent, discordIntegrationID string,
		status models.ProcessedDiscordMessageStatus,
	) (*models.ProcessedDiscordMessage, error)
	UpdateProcessedDiscordMessage(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) (*models.ProcessedDiscordMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) ([]*models.ProcessedDiscordMessage, error)
	GetProcessedDiscordMessageByID(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		discordIntegrationID string,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		organizationID models.OrgID,
		jobIDs []string,
		discordIntegrationID string,
	) (int, error)
	TESTS_UpdateProcessedDiscordMessageUpdatedAt(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
		updatedAt time.Time,
		discordIntegrationID string,
	) error
	DeleteProcessedDiscordMessagesByJobID(
		ctx context.Context,
		organizationID models.OrgID,
		jobID string,
		discordIntegrationID string,
	) error
	GetProcessedMessagesByStatus(
		ctx context.Context,
		organizationID models.OrgID,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) ([]*models.ProcessedDiscordMessage, error)
}

// JobsService defines the interface for job-related operations
type JobsService interface {
	GetJobByID(
		ctx context.Context,
		organizationID models.OrgID,
		id string,
	) (mo.Option[*models.Job], error)
	UpdateJobTimestamp(ctx context.Context, organizationID models.OrgID, jobID string) error
	GetIdleJobs(ctx context.Context, organizationID models.OrgID, idleMinutes int) ([]*models.Job, error)
	DeleteJob(ctx context.Context, organizationID models.OrgID, id string) error

	// Slack-specific methods
	CreateSlackJob(
		ctx context.Context,
		organizationID models.OrgID,
		slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
	) (*models.Job, error)
	GetJobBySlackThread(
		ctx context.Context,
		organizationID models.OrgID,
		threadTS, channelID, slackIntegrationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForSlackThread(
		ctx context.Context,
		organizationID models.OrgID,
		threadTS, channelID, slackUserID, slackIntegrationID string,
	) (*models.JobCreationResult, error)

	// Discord-specific methods
	CreateDiscordJob(
		ctx context.Context,
		organizationID models.OrgID,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	) (*models.Job, error)
	GetJobByDiscordThread(
		ctx context.Context,
		organizationID models.OrgID,
		threadID, discordIntegrationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForDiscordThread(
		ctx context.Context,
		organizationID models.OrgID,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
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
