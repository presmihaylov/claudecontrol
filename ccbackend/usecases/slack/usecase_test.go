package slack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/agents"
)

// Test data fixtures
var (
	testOrg = &models.Organization{
		ID: "org_01234567890123456789012345",
	}

	testUser = &models.User{
		ID:             "u_01234567890123456789012345",
		AuthProvider:   "clerk",
		AuthProviderID: "user_test_123",
		OrganizationID: testOrg.ID,
	}

	testSlackIntegration = &models.SlackIntegration{
		ID:             "si_01234567890123456789012345",
		SlackTeamID:    "T123456",
		SlackAuthToken: "xoxb-test-token",
		SlackTeamName:  "Test Team",
		OrganizationID: testOrg.ID,
	}

	testAgent = &models.ActiveAgent{
		ID:             "aa_01234567890123456789012345",
		WSConnectionID: "conn_123",
		OrganizationID: testOrg.ID,
		CCAgentID:      "ccagent_test_123",
	}

	testJob = &models.Job{
		ID: "job_01234567890123456789012345",
		SlackPayload: &models.SlackJobPayload{
			ThreadTS:      "1234567890.123456",
			ChannelID:     "C123456",
			IntegrationID: testSlackIntegration.ID,
			UserID:        "U789012",
		},
	}

	testProcessedMessage = &models.ProcessedSlackMessage{
		ID:                 "psm_01234567890123456789012345",
		SlackIntegrationID: testSlackIntegration.ID,
		JobID:              testJob.ID,
		SlackTS:            "1234567890.123456",
		SlackChannelID:     "C123456",
		TextContent:        "test message",
		Status:             models.ProcessedSlackMessageStatusQueued,
		OrganizationID:     testOrg.ID,
	}

	testSlackEvent = &models.SlackMessageEvent{
		Text:     "<@U123456> test message",
		Channel:  "C123456",
		User:     "U789012",
		TS:       "1234567890.123456",
		ThreadTS: "",
	}
)

// Mocks
type MockAgentsService struct {
	mock.Mock
}

func (m *MockAgentsService) CreateActiveAgent(ctx context.Context, slackIntegrationID string, connectionID string) (*models.ActiveAgent, error) {
	args := m.Called(ctx, slackIntegrationID, connectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentsBySlackIntegrationID(ctx context.Context, slackIntegrationID string) ([]*models.ActiveAgent, error) {
	args := m.Called(ctx, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentByID(ctx context.Context, id string) (*models.ActiveAgent, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) GetActiveAgentByConnectionID(ctx context.Context, connectionID string) (*models.ActiveAgent, error) {
	args := m.Called(ctx, connectionID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsService) DeleteActiveAgent(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockAgentsService) DeleteActiveAgentByConnectionID(ctx context.Context, connectionID string) error {
	args := m.Called(ctx, connectionID)
	return args.Error(0)
}

type MockJobsService struct {
	mock.Mock
}

func (m *MockJobsService) CreateJob(ctx context.Context, slackThreadTS string, slackChannelID string, slackUserID string, slackIntegrationID string, organizationID string) (*models.Job, error) {
	args := m.Called(ctx, slackThreadTS, slackChannelID, slackUserID, slackIntegrationID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobBySlackThreadTS(ctx context.Context, slackThreadTS string, slackChannelID string, slackIntegrationID string) (*models.Job, error) {
	args := m.Called(ctx, slackThreadTS, slackChannelID, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetJobByID(ctx context.Context, id string) (*models.Job, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Job), args.Error(1)
}

func (m *MockJobsService) GetIdleJobsByOrganizationID(ctx context.Context, organizationID string) ([]*models.Job, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.Job), args.Error(1)
}

func (m *MockJobsService) CreateProcessedSlackMessage(ctx context.Context, msg *models.ProcessedSlackMessage) error {
	args := m.Called(ctx, msg)
	return args.Error(0)
}

func (m *MockJobsService) GetProcessedSlackMessage(ctx context.Context, slackMessageTS string, slackChannelID string, slackIntegrationID string) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, slackMessageTS, slackChannelID, slackIntegrationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) GetProcessedSlackMessageByID(ctx context.Context, id string) (*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) UpdateProcessedSlackMessageStatus(ctx context.Context, id string, status models.ProcessedSlackMessageStatus) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockJobsService) GetQueuedProcessedSlackMessages(ctx context.Context) ([]*models.ProcessedSlackMessage, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.ProcessedSlackMessage), args.Error(1)
}

func (m *MockJobsService) CreateAgentJobAssignment(ctx context.Context, agentID string, jobID string) error {
	args := m.Called(ctx, agentID, jobID)
	return args.Error(0)
}

func (m *MockJobsService) GetAgentJobAssignments(ctx context.Context, agentID string) ([]*models.AgentJobAssignment, error) {
	args := m.Called(ctx, agentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AgentJobAssignment), args.Error(1)
}

func (m *MockJobsService) DeleteAgentJobAssignmentsByJobID(ctx context.Context, jobID string) error {
	args := m.Called(ctx, jobID)
	return args.Error(0)
}

func (m *MockJobsService) GetAgentJobAssignmentsByJobID(ctx context.Context, jobID string) ([]*models.AgentJobAssignment, error) {
	args := m.Called(ctx, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.AgentJobAssignment), args.Error(1)
}

func (m *MockJobsService) IsAgentAssignedToJob(ctx context.Context, agentID string, jobID string) (bool, error) {
	args := m.Called(ctx, agentID, jobID)
	return args.Bool(0), args.Error(1)
}

func (m *MockJobsService) CreateJobWithTransaction(ctx context.Context, job *models.Job, processedMessage *models.ProcessedSlackMessage, agentJobAssignment *models.AgentJobAssignment) error {
	args := m.Called(ctx, job, processedMessage, agentJobAssignment)
	return args.Error(0)
}

type MockSlackIntegrationsService struct {
	mock.Mock
}

func (m *MockSlackIntegrationsService) CreateSlackIntegration(ctx context.Context, slackTeamID string, slackAuthToken string, slackTeamName string, slackChannelID string, organizationID string) (*models.SlackIntegration, error) {
	args := m.Called(ctx, slackTeamID, slackAuthToken, slackTeamName, slackChannelID, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationByID(ctx context.Context, id string) (*models.SlackIntegration, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) GetSlackIntegrationsByOrganizationID(ctx context.Context, organizationID string) ([]*models.SlackIntegration, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.SlackIntegration), args.Error(1)
}

func (m *MockSlackIntegrationsService) DeleteSlackIntegrationByID(ctx context.Context, id string, userID string) (bool, error) {
	args := m.Called(ctx, id, userID)
	return args.Bool(0), args.Error(1)
}

type MockTransactionManager struct {
	mock.Mock
}

func (m *MockTransactionManager) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	args := m.Called(ctx, fn)
	if args.Get(0) != nil {
		return args.Error(0)
	}
	// Execute the function for testing
	return fn(ctx)
}

type MockSocketIOClient struct {
	mock.Mock
}

func (m *MockSocketIOClient) SendMessage(connectionID string, message any) error {
	args := m.Called(connectionID, message)
	return args.Error(0)
}

func (m *MockSocketIOClient) BroadcastMessage(message any) error {
	args := m.Called(message)
	return args.Error(0)
}

type MockAgentsUseCase struct {
	mock.Mock
}

func (m *MockAgentsUseCase) AssignAgentToJob(ctx context.Context, slackIntegrationID string, jobID string) (*models.ActiveAgent, error) {
	args := m.Called(ctx, slackIntegrationID, jobID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

func (m *MockAgentsUseCase) GetActiveAgentForOrganization(ctx context.Context, organizationID string) (*models.ActiveAgent, error) {
	args := m.Called(ctx, organizationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.ActiveAgent), args.Error(1)
}

// Helper function to create a SlackUseCase with mocked dependencies
func setupSlackUseCase(t *testing.T) (*SlackUseCase, *MockAgentsService, *MockJobsService, *MockSlackIntegrationsService, *MockTransactionManager, *MockSocketIOClient, *MockAgentsUseCase) {
	mockAgentsService := new(MockAgentsService)
	mockJobsService := new(MockJobsService)
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockTxManager := new(MockTransactionManager)
	mockSocketClient := new(MockSocketIOClient)
	mockAgentsUseCase := new(MockAgentsUseCase)

	useCase := &SlackUseCase{
		agentsService:            mockAgentsService,
		jobsService:              mockJobsService,
		slackIntegrationsService: mockSlackIntegrationsService,
		txManager:                mockTxManager,
		socketClient:             mockSocketClient,
		agentsUseCase:            mockAgentsUseCase,
	}

	return useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, mockTxManager, mockSocketClient, mockAgentsUseCase
}

// Helper function to create test Slack event
func createTestSlackEvent(text string, threadTS string) *models.SlackMessageEvent {
	event := &models.SlackMessageEvent{
		Type:    "app_mention",
		Text:    text,
		Channel: "C123456",
		User:    "U789012",
		TS:      "1234567890.123456",
	}
	if threadTS != "" {
		event.ThreadTS = threadTS
	}
	return event
}

// Helper function to create test job
func createTestJob(threadTS string) *models.Job {
	return &models.Job{
		ID: core.NewID("job"),
		SlackPayload: &models.SlackJobPayload{
			ThreadTS:      threadTS,
			ChannelID:     "C123456",
			IntegrationID: testSlackIntegration.ID,
			UserID:        "U789012",
		},
	}
}

// Helper function to create test processed message
func createTestProcessedMessage(jobID string, status models.ProcessedSlackMessageStatus) *models.ProcessedSlackMessage {
	return &models.ProcessedSlackMessage{
		ID:                 core.NewID("psm"),
		SlackIntegrationID: testSlackIntegration.ID,
		JobID:              jobID,
		SlackMessageTS:     "1234567890.123456",
		SlackChannelID:     "C123456",
		SlackUserID:        "U789012",
		Status:             status,
	}
}

// Helper function to assert that a Slack message was sent
func assertSlackMessageSent(t *testing.T, mockSocketClient *MockSocketIOClient, connectionID string, messageType string) {
	require.NotNil(t, mockSocketClient)
	
	// Find the call with the expected message type
	for _, call := range mockSocketClient.Calls {
		if call.Method == "SendMessage" {
			if callConnID, ok := call.Arguments[0].(string); ok && callConnID == connectionID {
				if msg, ok := call.Arguments[1].(map[string]any); ok {
					if msgType, ok := msg["type"].(string); ok && msgType == messageType {
						return
					}
				}
			}
		}
	}
	
	t.Errorf("Expected SendMessage to be called with connection ID %s and message type %s", connectionID, messageType)
}

// Helper function to assert reaction update
func assertReactionUpdated(t *testing.T, mockJobsService *MockJobsService, expectedStatus models.ProcessedSlackMessageStatus) {
	require.NotNil(t, mockJobsService)
	
	// Check if UpdateProcessedSlackMessageStatus was called with expected status
	for _, call := range mockJobsService.Calls {
		if call.Method == "UpdateProcessedSlackMessageStatus" {
			if status, ok := call.Arguments[1].(models.ProcessedSlackMessageStatus); ok && status == expectedStatus {
				return
			}
		}
	}
	
	t.Errorf("Expected UpdateProcessedSlackMessageStatus to be called with status %v", expectedStatus)
}