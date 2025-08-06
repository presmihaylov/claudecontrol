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

// AgentsServiceInterface defines the interface for agent-related operations
type AgentsServiceInterface interface {
	UpsertActiveAgent(wsConnectionID, slackIntegrationID string, agentID string) (*models.ActiveAgent, error)
	GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error)
	GetAgentByJobID(jobID, slackIntegrationID string) (*models.ActiveAgent, error)
	DeleteActiveAgentByWsConnectionID(wsConnectionID, slackIntegrationID string) error
	DeleteActiveAgent(agentID, slackIntegrationID string) error
	GetConnectedActiveAgents(slackIntegrationID string, connectedClientIDs []string) ([]*models.ActiveAgent, error)
	CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool
	AssignAgentToJob(agentID, jobID, slackIntegrationID string) error
	UnassignAgentFromJob(agentID, jobID, slackIntegrationID string) error
	GetActiveAgentJobAssignments(agentID, slackIntegrationID string) ([]string, error)
	GetInactiveAgents(slackIntegrationID string, inactiveThresholdMinutes int) ([]*models.ActiveAgent, error)
	UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID string) error
}

// JobsServiceInterface defines the interface for job-related operations
type JobsServiceInterface interface {
	CreateJob(slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string) (*models.Job, error)
	GetJobByID(jobID, slackIntegrationID string) (*models.Job, error)
	GetJobBySlackThread(slackThreadTS, slackChannelID, slackIntegrationID string) (*models.Job, error)
	GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string) (*models.JobCreationResult, error)
	DeleteJob(jobID, slackIntegrationID string) error
	UpdateJobTimestamp(jobID, slackIntegrationID string) error
	GetJobsWithQueuedMessages(slackIntegrationID string) ([]*models.Job, error)
	GetActiveMessageCountForJobs(jobIDs []string, slackIntegrationID string) (int, error)
	CreateProcessedSlackMessage(jobID, slackChannelID, slackTS, textContent, slackIntegrationID string, status models.ProcessedSlackMessageStatus) (*models.ProcessedSlackMessage, error)
	GetProcessedSlackMessageByID(messageID, slackIntegrationID string) (*models.ProcessedSlackMessage, error)
	UpdateProcessedSlackMessage(messageID string, status models.ProcessedSlackMessageStatus, slackIntegrationID string) (*models.ProcessedSlackMessage, error)
	GetLatestProcessedMessageForJob(jobID, slackIntegrationID string) (*models.ProcessedSlackMessage, error)
	GetProcessedMessagesByJobIDAndStatus(jobID string, status models.ProcessedSlackMessageStatus, slackIntegrationID string) ([]*models.ProcessedSlackMessage, error)
}

// DashboardServicesInterface defines the interface for dashboard handler dependencies
type DashboardServicesInterface interface {
	// User operations
	GetOrCreateUser(authProvider, authProviderID string) (*models.User, error)

	// Slack integration operations
	CreateSlackIntegration(slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error)
	GetSlackIntegrationsByUserID(userID string) ([]*models.SlackIntegration, error)
	DeleteSlackIntegration(ctx context.Context, integrationID string) error
	GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error)
}

// DashboardServices implements DashboardServicesInterface by combining existing services
type DashboardServices struct {
	usersService             UsersServiceInterface
	slackIntegrationsService SlackIntegrationsServiceInterface
}

func NewDashboardServices(usersService UsersServiceInterface, slackIntegrationsService SlackIntegrationsServiceInterface) *DashboardServices {
	return &DashboardServices{
		usersService:             usersService,
		slackIntegrationsService: slackIntegrationsService,
	}
}

func (d *DashboardServices) GetOrCreateUser(authProvider, authProviderID string) (*models.User, error) {
	return d.usersService.GetOrCreateUser(authProvider, authProviderID)
}

func (d *DashboardServices) CreateSlackIntegration(slackAuthCode, redirectURL string, userID string) (*models.SlackIntegration, error) {
	return d.slackIntegrationsService.CreateSlackIntegration(slackAuthCode, redirectURL, userID)
}

func (d *DashboardServices) GetSlackIntegrationsByUserID(userID string) ([]*models.SlackIntegration, error) {
	return d.slackIntegrationsService.GetSlackIntegrationsByUserID(userID)
}

func (d *DashboardServices) DeleteSlackIntegration(ctx context.Context, integrationID string) error {
	return d.slackIntegrationsService.DeleteSlackIntegration(ctx, integrationID)
}

func (d *DashboardServices) GenerateCCAgentSecretKey(ctx context.Context, integrationID string) (string, error) {
	return d.slackIntegrationsService.GenerateCCAgentSecretKey(ctx, integrationID)
}
