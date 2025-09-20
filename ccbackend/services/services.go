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
	GenerateCCAgentSecretKey(ctx context.Context, orgID models.OrgID) (string, error)
	GetOrganizationBySecretKey(ctx context.Context, secretKey string) (mo.Option[*models.Organization], error)
}

// SlackIntegrationsService defines the interface for Slack integration operations
type SlackIntegrationsService interface {
	CreateSlackIntegration(
		ctx context.Context,
		orgID models.OrgID,
		slackAuthCode, redirectURL string,
	) (*models.SlackIntegration, error)
	GetSlackIntegrationsByOrganizationID(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.SlackIntegration, error)
	GetAllSlackIntegrations(ctx context.Context) ([]models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error
	GetSlackIntegrationByTeamID(ctx context.Context, teamID string) (mo.Option[*models.SlackIntegration], error)
	GetSlackIntegrationByID(ctx context.Context, id string) (mo.Option[*models.SlackIntegration], error)
}

// DiscordIntegrationsService defines the interface for Discord integration operations
type DiscordIntegrationsService interface {
	CreateDiscordIntegration(
		ctx context.Context,
		orgID models.OrgID,
		discordAuthCode, guildID, redirectURL string,
	) (*models.DiscordIntegration, error)
	GetDiscordIntegrationsByOrganizationID(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.DiscordIntegration, error)
	GetAllDiscordIntegrations(ctx context.Context) ([]models.DiscordIntegration, error)
	DeleteDiscordIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error
	GetDiscordIntegrationByGuildID(ctx context.Context, guildID string) (mo.Option[*models.DiscordIntegration], error)
	GetDiscordIntegrationByID(ctx context.Context, id string) (mo.Option[*models.DiscordIntegration], error)
}

// GitHubIntegrationsService defines the interface for GitHub integration operations
type GitHubIntegrationsService interface {
	CreateGitHubIntegration(
		ctx context.Context,
		orgID models.OrgID,
		authCode, installationID string,
	) (*models.GitHubIntegration, error)
	ListGitHubIntegrations(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.GitHubIntegration, error)
	GetGitHubIntegrationByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.GitHubIntegration], error)
	DeleteGitHubIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error
	ListAvailableRepositories(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.GitHubRepository, error)
}

// AnthropicIntegrationsService defines the interface for Anthropic integration operations
type AnthropicIntegrationsService interface {
	CreateAnthropicIntegration(
		ctx context.Context,
		orgID models.OrgID,
		apiKey, oauthToken, codeVerifier *string,
	) (*models.AnthropicIntegration, error)
	ListAnthropicIntegrations(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.AnthropicIntegration, error)
	GetAnthropicIntegrationByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.AnthropicIntegration], error)
	DeleteAnthropicIntegration(ctx context.Context, orgID models.OrgID, integrationID string) error
	RefreshTokens(
		ctx context.Context,
		orgID models.OrgID,
		integrationID string,
	) (*models.AnthropicIntegration, error)
}

// CCAgentContainerIntegrationsService defines the interface for CCAgent container integration operations
type CCAgentContainerIntegrationsService interface {
	CreateCCAgentContainerIntegration(
		ctx context.Context,
		orgID models.OrgID,
		instancesCount int,
		repoURL string,
	) (*models.CCAgentContainerIntegration, error)
	ListCCAgentContainerIntegrations(
		ctx context.Context,
		orgID models.OrgID,
	) ([]models.CCAgentContainerIntegration, error)
	GetCCAgentContainerIntegrationByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.CCAgentContainerIntegration], error)
	DeleteCCAgentContainerIntegration(
		ctx context.Context,
		orgID models.OrgID,
		integrationID string,
	) error
	RedeployCCAgentContainer(
		ctx context.Context,
		orgID models.OrgID,
		integrationID string,
		updateConfigOnly bool,
	) error
}

// AgentsService defines the interface for agent-related operations
type AgentsService interface {
	UpsertActiveAgent(
		ctx context.Context,
		orgID models.OrgID,
		wsConnectionID string,
		agentID string,
		repoURL string,
	) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(ctx context.Context, orgID models.OrgID, wsConnectionID string) error
	DeleteActiveAgent(ctx context.Context, orgID models.OrgID, id string) error
	GetAgentByID(ctx context.Context, orgID models.OrgID, id string) (mo.Option[*models.ActiveAgent], error)
	GetAvailableAgents(ctx context.Context, orgID models.OrgID) ([]*models.ActiveAgent, error)
	GetConnectedActiveAgents(
		ctx context.Context,
		orgID models.OrgID,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	GetConnectedAvailableAgents(
		ctx context.Context,
		orgID models.OrgID,
		connectedClientIDs []string,
	) ([]*models.ActiveAgent, error)
	CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool
	AssignAgentToJob(ctx context.Context, orgID models.OrgID, agentID, jobID string) error
	UnassignAgentFromJob(ctx context.Context, orgID models.OrgID, agentID, jobID string) error
	GetAgentByJobID(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetAgentByWSConnectionID(
		ctx context.Context,
		orgID models.OrgID,
		wsConnectionID string,
	) (mo.Option[*models.ActiveAgent], error)
	GetActiveAgentJobAssignments(ctx context.Context, orgID models.OrgID, agentID string) ([]string, error)
	UpdateAgentLastActiveAt(ctx context.Context, orgID models.OrgID, wsConnectionID string) error
	GetInactiveAgents(
		ctx context.Context,
		orgID models.OrgID,
		inactiveThresholdMinutes int,
	) ([]*models.ActiveAgent, error)
	DisconnectAllActiveAgentsByOrganization(ctx context.Context, orgID models.OrgID) error
}

// SlackMessagesService defines the interface for processed slack message operations
type SlackMessagesService interface {
	CreateProcessedSlackMessage(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		slackChannelID, slackTS, textContent, slackIntegrationID string,
		status models.ProcessedSlackMessageStatus,
	) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(
		ctx context.Context,
		orgID models.OrgID,
		id string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) ([]*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		slackIntegrationID string,
	) (mo.Option[*models.ProcessedSlackMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		orgID models.OrgID,
		jobIDs []string,
		slackIntegrationID string,
	) (int, error)
	TESTS_UpdateProcessedSlackMessageUpdatedAt(
		ctx context.Context,
		orgID models.OrgID,
		id string,
		updatedAt time.Time,
		slackIntegrationID string,
	) error
	DeleteProcessedSlackMessagesByJobID(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		slackIntegrationID string,
	) error
	GetProcessedMessagesByStatus(
		ctx context.Context,
		orgID models.OrgID,
		status models.ProcessedSlackMessageStatus,
		slackIntegrationID string,
	) ([]*models.ProcessedSlackMessage, error)
}

// DiscordMessagesService defines the interface for processed discord message operations
type DiscordMessagesService interface {
	CreateProcessedDiscordMessage(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		discordMessageID, discordThreadID, textContent, discordIntegrationID string,
		status models.ProcessedDiscordMessageStatus,
	) (*models.ProcessedDiscordMessage, error)
	UpdateProcessedDiscordMessage(
		ctx context.Context,
		orgID models.OrgID,
		id string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) (*models.ProcessedDiscordMessage, error)
	GetProcessedMessagesByJobIDAndStatus(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) ([]*models.ProcessedDiscordMessage, error)
	GetProcessedDiscordMessageByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetLatestProcessedMessageForJob(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		discordIntegrationID string,
	) (mo.Option[*models.ProcessedDiscordMessage], error)
	GetActiveMessageCountForJobs(
		ctx context.Context,
		orgID models.OrgID,
		jobIDs []string,
		discordIntegrationID string,
	) (int, error)
	TESTS_UpdateProcessedDiscordMessageUpdatedAt(
		ctx context.Context,
		orgID models.OrgID,
		id string,
		updatedAt time.Time,
		discordIntegrationID string,
	) error
	DeleteProcessedDiscordMessagesByJobID(
		ctx context.Context,
		orgID models.OrgID,
		jobID string,
		discordIntegrationID string,
	) error
	GetProcessedMessagesByStatus(
		ctx context.Context,
		orgID models.OrgID,
		status models.ProcessedDiscordMessageStatus,
		discordIntegrationID string,
	) ([]*models.ProcessedDiscordMessage, error)
}

// JobsService defines the interface for job-related operations
type JobsService interface {
	GetJobByID(
		ctx context.Context,
		orgID models.OrgID,
		id string,
	) (mo.Option[*models.Job], error)
	UpdateJobTimestamp(ctx context.Context, orgID models.OrgID, jobID string) error
	GetIdleJobs(ctx context.Context, orgID models.OrgID, idleMinutes int) ([]*models.Job, error)
	DeleteJob(ctx context.Context, orgID models.OrgID, id string) error

	// Slack-specific methods
	CreateSlackJob(
		ctx context.Context,
		orgID models.OrgID,
		slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
	) (*models.Job, error)
	GetJobBySlackThread(
		ctx context.Context,
		orgID models.OrgID,
		threadTS, channelID, slackIntegrationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForSlackThread(
		ctx context.Context,
		orgID models.OrgID,
		threadTS, channelID, slackUserID, slackIntegrationID string,
	) (*models.JobCreationResult, error)

	// Discord-specific methods
	CreateDiscordJob(
		ctx context.Context,
		orgID models.OrgID,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	) (*models.Job, error)
	GetJobByDiscordThread(
		ctx context.Context,
		orgID models.OrgID,
		threadID, discordIntegrationID string,
	) (mo.Option[*models.Job], error)
	GetOrCreateJobForDiscordThread(
		ctx context.Context,
		orgID models.OrgID,
		discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	) (*models.JobCreationResult, error)
}

// SettingsService defines the interface for settings operations
type SettingsService interface {
	UpsertBooleanSetting(ctx context.Context, organizationID string, key string, value bool) error
	UpsertStringSetting(ctx context.Context, organizationID string, key string, value string) error
	UpsertStringArraySetting(ctx context.Context, organizationID string, key string, value []string) error
	GetBooleanSetting(ctx context.Context, organizationID string, key string) (bool, error)
	GetStringSetting(ctx context.Context, organizationID string, key string) (string, error)
	GetStringArraySetting(ctx context.Context, organizationID string, key string) ([]string, error)
	GetSettingByType(
		ctx context.Context,
		organizationID string,
		key string,
		settingType models.SettingType,
	) (any, error)
}

// ConnectedChannelsService defines the interface for connected channels operations
type ConnectedChannelsService interface {
	// Slack-specific methods
	UpsertSlackConnectedChannel(
		ctx context.Context,
		orgID models.OrgID,
		teamID string,
		channelID string,
	) (*models.SlackConnectedChannel, error)
	GetSlackConnectedChannel(
		ctx context.Context,
		orgID models.OrgID,
		teamID string,
		channelID string,
	) (mo.Option[*models.SlackConnectedChannel], error)
	UpdateSlackChannelDefaultRepo(
		ctx context.Context,
		orgID models.OrgID,
		teamID string,
		channelID string,
		repoURL string,
	) (*models.SlackConnectedChannel, error)

	// Discord-specific methods
	UpsertDiscordConnectedChannel(
		ctx context.Context,
		orgID models.OrgID,
		guildID string,
		channelID string,
	) (*models.DiscordConnectedChannel, error)
	GetDiscordConnectedChannel(
		ctx context.Context,
		orgID models.OrgID,
		guildID string,
		channelID string,
	) (mo.Option[*models.DiscordConnectedChannel], error)
	UpdateDiscordChannelDefaultRepo(
		ctx context.Context,
		orgID models.OrgID,
		guildID string,
		channelID string,
		repoURL string,
	) (*models.DiscordConnectedChannel, error)
}

// CommandsService defines the interface for command processing operations
type CommandsService interface {
	ProcessCommand(
		ctx context.Context,
		orgID models.OrgID,
		request models.CommandRequest,
		connectedChannel models.ConnectedChannel,
	) (*models.CommandResult, error)
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
