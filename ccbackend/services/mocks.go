package services

import (
	"context"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

// MockAgentsService is a mock implementation of AgentsService
type MockAgentsService struct {
	mock.Mock
}

func (m *MockAgentsService) UpsertActiveAgent(
	ctx context.Context,
	wsConnectionID, organizationID string,
	agentID string,
) (*models.ActiveAgent, error) {
	args := m.Called(ctx, wsConnectionID, organizationID, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) DeleteActiveAgentByWsConnectionID(
	ctx context.Context,
	wsConnectionID, organizationID string,
) error {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) DeleteActiveAgent(ctx context.Context, id string, organizationID string) error {
	args := m.Called(ctx, id, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAvailableAgents(
	ctx context.Context,
	organizationID string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedActiveAgents(
	ctx context.Context,
	organizationID string,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, connectedClientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetConnectedAvailableAgents(
	ctx context.Context,
	organizationID string,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, connectedClientIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool {
	args := m.Called(agent, connectedClientIDs)
	return args.Bool(0)
}

func (m *MockAgentsService) AssignAgentToJob(ctx context.Context, agentID, jobID string, organizationID string) error {
	args := m.Called(ctx, agentID, jobID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) UnassignAgentFromJob(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	args := m.Called(ctx, agentID, jobID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetAgentByJobID(
	ctx context.Context,
	jobID string,
	organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, jobID, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetAgentByWSConnectionID(
	ctx context.Context,
	wsConnectionID, organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Get(0).(mo.Option[*models.ActiveAgent]), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentJobAssignments(
	ctx context.Context,
	agentID string,
	organizationID string,
) ([]string, error) {
	args := m.Called(ctx, agentID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockAgentsService) UpdateAgentLastActiveAt(ctx context.Context, wsConnectionID, organizationID string) error {
	args := m.Called(ctx, wsConnectionID, organizationID)
	return args.Error(0)
}

func (m *MockAgentsService) GetInactiveAgents(
	ctx context.Context,
	organizationID string,
	inactiveThresholdMinutes int,
) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID, inactiveThresholdMinutes)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

// MockJobsService is a mock implementation of JobsService
type MockJobsService struct {
	mock.Mock
}

func (m *MockJobsService) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	slackIntegrationID string,
	organizationID string,
) (int, error) {
	args := m.Called(ctx, jobIDs, slackIntegrationID, organizationID)
	return args.Int(0), args.Error(1)
}

func (m *MockJobsService) CreateJob(
	ctx context.Context,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID string,
) (*models.Job, error) {
	args := m.Called(ctx, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID, organizationID string,
) (mo.Option[*models.Job], error) {
	args := m.Called(ctx, threadTS, channelID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.Job]), args.Error(1)
}

func (m *MockJobsService) GetOrCreateJobForSlackThread(
	ctx context.Context,
	threadTS, channelID, slackUserID, slackIntegrationID, organizationID string,
) (*models.JobCreationResult, error) {
	args := m.Called(ctx, threadTS, channelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.JobCreationResult), args.Error(1)
}

func (m *MockJobsService) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetIdleJobs(
	ctx context.Context,
	idleMinutes int,
	organizationID string,
) ([]*models.Job, error) {
	args := m.Called(ctx, idleMinutes, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) DeleteJob(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) CreateProcessedSlackMessage(
	ctx context.Context,
	jobID string,
	slackChannelID, slackTS, textContent, slackIntegrationID, organizationID string,
	status models.ProcessedSlackMessageStatus,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, slackChannelID, slackTS, textContent, slackIntegrationID, organizationID, status)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) UpdateProcessedSlackMessage(
	ctx context.Context,
	id string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, id, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, jobID, status, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, id, organizationID)
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockJobsService) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

func (m *MockJobsService) GetJobsWithQueuedMessages(
	ctx context.Context,
	slackIntegrationID string,
	organizationID string,
) ([]*models.Job, error) {
	args := m.Called(ctx, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	args := m.Called(ctx, jobID, slackIntegrationID, organizationID)
	return args.Get(0).(mo.Option[*models.ProcessedSlackMessage]), args.Error(1)
}

func (m *MockJobsService) TESTS_UpdateProcessedSlackMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	args := m.Called(ctx, id, updatedAt, slackIntegrationID, organizationID)
	return args.Error(0)
}

// MockSlackIntegrationsService is a mock implementation of SlackIntegrationsService
type MockSlackIntegrationsService struct {
	mock.Mock
}

func (m *MockSlackIntegrationsService) CreateSlackIntegration(
	ctx context.Context,
	organizationID, slackAuthCode, redirectURL string,
) (*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID, slackAuthCode, redirectURL)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(
	ctx context.Context,
	organizationID string,
) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetAllSlackIntegrations(
	ctx context.Context,
) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) DeleteSlackIntegration(
	ctx context.Context,
	organizationID, integrationID string,
) error {
	args := m.Called(ctx, organizationID, integrationID)
	return args.Error(0)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByTeamID(
	ctx context.Context,
	teamID string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, teamID)
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.SlackIntegration], error) {
	args := m.Called(ctx, id)
	return args.Get(0).(mo.Option[*models.SlackIntegration]), args.Error(1)
}

// MockOrganizationsService is a mock implementation of OrganizationsService
type MockOrganizationsService struct {
	mock.Mock
}

func (m *MockOrganizationsService) CreateOrganization(ctx context.Context) (*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Organization), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationByID(
	ctx context.Context,
	id string,
) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, id)
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}

func (m *MockOrganizationsService) GetAllOrganizations(ctx context.Context) ([]*models.Organization, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Organization), args.Error(1)
}

func (m *MockOrganizationsService) GenerateCCAgentSecretKey(
	ctx context.Context,
	organizationID string,
) (string, error) {
	args := m.Called(ctx, organizationID)
	return args.String(0), args.Error(1)
}

func (m *MockOrganizationsService) GetOrganizationBySecretKey(
	ctx context.Context,
	secretKey string,
) (mo.Option[*models.Organization], error) {
	args := m.Called(ctx, secretKey)
	return args.Get(0).(mo.Option[*models.Organization]), args.Error(1)
}
