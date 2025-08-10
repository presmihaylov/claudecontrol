package slack

import (
	"context"
	"errors"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/clients/socketio"
	"ccbackend/models"
	agentsservice "ccbackend/services/agents"
	"ccbackend/services/jobs"
	slackintegrations "ccbackend/services/slack_integrations"
	"ccbackend/services/slackmessages"
	"ccbackend/services/txmanager"
	agentsusecase "ccbackend/usecases/agents"
)

// Test data constants
const (
	testJobID              = "job_123"
	testAgentID            = "agent_123"
	testUserID             = "user123"
	testOrgID              = "org_123"
	testChannelID          = "channel123"
	testSlackIntegrationID = "slack_int_123"
	testMessageID          = "msg_123"
	testWSConnectionID     = "ws_123"
	testThreadTS           = "1234567890.123"
	testSlackToken         = "token123"
)

// Helper functions for test data creation
func newTestJob(orgID, jobID string) *models.Job {
	return &models.Job{
		ID:             jobID,
		OrganizationID: orgID,
		SlackPayload: &models.SlackJobPayload{
			IntegrationID: testSlackIntegrationID,
			ChannelID:     testChannelID,
			ThreadTS:      testThreadTS,
			UserID:        testUserID,
		},
	}
}

func newTestSlackIntegration(integrationID, orgID string) *models.SlackIntegration {
	return &models.SlackIntegration{
		ID:             integrationID,
		OrganizationID: orgID,
		SlackAuthToken: testSlackToken,
	}
}

func newTestAgent(agentID, wsConnectionID, orgID string) *models.ActiveAgent {
	return &models.ActiveAgent{
		ID:               agentID,
		WSConnectionID:   wsConnectionID,
		OrganizationID:   orgID,
	}
}

func newTestProcessedMessage(messageID, jobID, orgID string, status models.ProcessedSlackMessageStatus) *models.ProcessedSlackMessage {
	return &models.ProcessedSlackMessage{
		ID:                 messageID,
		JobID:              jobID,
		SlackChannelID:     testChannelID,
		SlackTS:            testThreadTS,
		TextContent:        "Test message",
		SlackIntegrationID: testSlackIntegrationID,
		OrganizationID:     orgID,
		Status:             status,
	}
}

func setupBasicMocks() (*socketio.MockSocketIOClient, *agentsservice.MockAgentsService, *jobs.MockJobsService, *slackmessages.MockSlackMessagesService, *slackintegrations.MockSlackIntegrationsService, *txmanager.MockTransactionManager, *agentsusecase.MockAgentsUseCase) {
	wsClient := &socketio.MockSocketIOClient{}
	agentsService := &agentsservice.MockAgentsService{}
	jobsService := &jobs.MockJobsService{}
	slackMessagesService := &slackmessages.MockSlackMessagesService{}
	slackIntegrationsService := &slackintegrations.MockSlackIntegrationsService{}
	txManager := &txmanager.MockTransactionManager{}
	agentsUseCase := &agentsusecase.MockAgentsUseCase{}

	return wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase
}

func TestSlackUseCase_ProcessSlackMessageEvent_NewThreadWithAgents(t *testing.T) {
	// Setup mocks
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Hello bot",
		TS:       testThreadTS,
		ThreadTS: "", // New thread
	}

	// Test data objects
	job := newTestJob(testOrgID, testJobID)
	slackIntegration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)
	connectedAgents := []*models.ActiveAgent{newTestAgent(testAgentID, testWSConnectionID, testOrgID)}
	processedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusInProgress)

	jobResult := &models.JobCreationResult{
		Job:    job,
		Status: models.JobCreationStatusCreated,
	}

	// Setup mock expectations
	jobsService.On("GetOrCreateJobForSlackThread", ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
		Return(jobResult, nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.Some(slackIntegration), nil)

	wsClient.On("GetClientIDs").Return([]string{testWSConnectionID})
	agentsService.On("GetConnectedActiveAgents", ctx, testOrgID, []string{testWSConnectionID}).
		Return(connectedAgents, nil)

	agentsUseCase.On("GetOrAssignAgentForJob", ctx, job, event.TS, testOrgID).
		Return(testWSConnectionID, nil)

	slackMessagesService.On("CreateProcessedSlackMessage", ctx, job.ID, event.Channel, event.TS, event.Text, testSlackIntegrationID, testOrgID, models.ProcessedSlackMessageStatusInProgress).
		Return(processedMessage, nil)

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert
	assert.NoError(t, err)
	wsClient.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
	slackIntegrationsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSlackMessageEvent_SlackIntegrationNotFound(t *testing.T) {
	// Test critical error path: Slack integration not found
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Hello bot",
		TS:       testThreadTS,
		ThreadTS: "",
	}

	job := newTestJob(testOrgID, testJobID)
	jobResult := &models.JobCreationResult{
		Job:    job,
		Status: models.JobCreationStatusCreated,
	}

	// Setup mock expectations
	jobsService.On("GetOrCreateJobForSlackThread", ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
		Return(jobResult, nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.None[*models.SlackIntegration](), nil)

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert - should return error for missing integration
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "slack integration not found")
	jobsService.AssertExpectations(t)
	slackIntegrationsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSlackMessageEvent_JobServiceError(t *testing.T) {
	// Test critical error path: Job service failure
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Hello bot",
		TS:       testThreadTS,
		ThreadTS: "",
	}

	// Setup mock expectations - job service fails
	jobsService.On("GetOrCreateJobForSlackThread", ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
		Return((*models.JobCreationResult)(nil), errors.New("database connection failed"))

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert - should propagate error from job service
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get or create job for slack thread")
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSlackMessageEvent_ThreadReplyWithExistingJob(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Follow up message",
		TS:       "1234567890.124",
		ThreadTS: testThreadTS, // Thread reply
	}

	// Existing job
	existingJob := newTestJob(testOrgID, testJobID)
	jobResult := &models.JobCreationResult{
		Job:    existingJob,
		Status: models.JobCreationStatusNA,
	}

	slackIntegration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)
	connectedAgents := []*models.ActiveAgent{newTestAgent(testAgentID, testWSConnectionID, testOrgID)}
	processedMessage := newTestProcessedMessage("msg_124", testJobID, testOrgID, models.ProcessedSlackMessageStatusInProgress)

	// Setup mock expectations
	// First, check if job exists for thread reply
	jobsService.On("GetJobBySlackThread", ctx, event.ThreadTS, event.Channel, testSlackIntegrationID, testOrgID).
		Return(mo.Some(existingJob), nil)

	jobsService.On("GetOrCreateJobForSlackThread", ctx, event.ThreadTS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
		Return(jobResult, nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.Some(slackIntegration), nil)

	wsClient.On("GetClientIDs").Return([]string{testWSConnectionID})
	agentsService.On("GetConnectedActiveAgents", ctx, testOrgID, []string{testWSConnectionID}).
		Return(connectedAgents, nil)

	agentsUseCase.On("GetOrAssignAgentForJob", ctx, existingJob, event.ThreadTS, testOrgID).
		Return(testWSConnectionID, nil)

	slackMessagesService.On("CreateProcessedSlackMessage", ctx, existingJob.ID, event.Channel, event.TS, event.Text, testSlackIntegrationID, testOrgID, models.ProcessedSlackMessageStatusInProgress).
		Return(processedMessage, nil)

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
	slackIntegrationsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSlackMessageEvent_ThreadReplyWithNoJob(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Reply without job",
		TS:       "1234567890.124",
		ThreadTS: testThreadTS, // Thread reply
	}

	slackIntegration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)

	// Setup mock expectations - job not found for thread
	jobsService.On("GetJobBySlackThread", ctx, event.ThreadTS, event.Channel, testSlackIntegrationID, testOrgID).
		Return(mo.None[*models.Job](), nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.Some(slackIntegration), nil)

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert - should complete without error but send error message to user
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSlackMessageEvent_NoAgentsAvailable(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	event := models.SlackMessageEvent{
		User:     testUserID,
		Channel:  testChannelID,
		Text:     "Hello bot",
		TS:       testThreadTS,
		ThreadTS: "", // New thread
	}

	job := newTestJob(testOrgID, testJobID)
	jobResult := &models.JobCreationResult{
		Job:    job,
		Status: models.JobCreationStatusCreated,
	}

	slackIntegration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)
	processedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusQueued)

	// Setup mock expectations
	jobsService.On("GetOrCreateJobForSlackThread", ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
		Return(jobResult, nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.Some(slackIntegration), nil)

	wsClient.On("GetClientIDs").Return([]string{})
	agentsService.On("GetConnectedActiveAgents", ctx, testOrgID, []string{}).
		Return([]*models.ActiveAgent{}, nil) // No agents available

	slackMessagesService.On("CreateProcessedSlackMessage", ctx, job.ID, event.Channel, event.TS, event.Text, testSlackIntegrationID, testOrgID, models.ProcessedSlackMessageStatusQueued).
		Return(processedMessage, nil)

	// Act
	err := useCase.ProcessSlackMessageEvent(ctx, event, testSlackIntegrationID, testOrgID)

	// Assert
	assert.NoError(t, err)
	agentsService.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	slackIntegrationsService.AssertExpectations(t)
	wsClient.AssertExpectations(t)
}

func TestSlackUseCase_ProcessReactionAdded_ValidCompletionByJobCreator(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	reactionName := "white_check_mark"

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, testWSConnectionID, testOrgID)
	slackIntegration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)

	// Setup mock expectations
	jobsService.On("GetJobBySlackThread", ctx, testThreadTS, testChannelID, testSlackIntegrationID, testOrgID).
		Return(mo.Some(job), nil)

	slackIntegrationsService.On("GetSlackIntegrationByID", ctx, testSlackIntegrationID).
		Return(mo.Some(slackIntegration), nil)

	agentsService.On("GetAgentByJobID", ctx, job.ID, testOrgID).
		Return(mo.Some(agent), nil)

	// Transaction expectations
	txManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			// Execute the transaction function
			_ = fn(ctx)
		}).Return(nil)

	agentsService.On("UnassignAgentFromJob", ctx, agent.ID, job.ID, testOrgID).
		Return(nil)

	jobsService.On("DeleteJob", ctx, job.ID, testOrgID).
		Return(nil)

	// Act
	err := useCase.ProcessReactionAdded(ctx, reactionName, testUserID, testChannelID, testThreadTS, testSlackIntegrationID, testOrgID)

	// Assert
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	slackIntegrationsService.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestSlackUseCase_ProcessReactionAdded_IgnoredReactionType(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	reactionName := "thumbsup" // Not a completion emoji

	// Act
	err := useCase.ProcessReactionAdded(ctx, reactionName, testUserID, testChannelID, testThreadTS, testSlackIntegrationID, testOrgID)

	// Assert - should return without error and no mocks called
	assert.NoError(t, err)
}

func TestSlackUseCase_ProcessReactionAdded_WrongUser(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	reactionName := "white_check_mark"
	wrongUserID := "other_user" // Different user

	job := newTestJob(testOrgID, testJobID)
	job.SlackPayload.UserID = "original_user" // Different from wrongUserID

	// Setup mock expectations
	jobsService.On("GetJobBySlackThread", ctx, testThreadTS, testChannelID, testSlackIntegrationID, testOrgID).
		Return(mo.Some(job), nil)

	// Act
	err := useCase.ProcessReactionAdded(ctx, reactionName, wrongUserID, testChannelID, testThreadTS, testSlackIntegrationID, testOrgID)

	// Assert - should return without error but not process the reaction
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessJobComplete_Success(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.JobCompletePayload{
		JobID:  testJobID,
		Reason: "Task completed successfully",
	}

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, clientID, testOrgID)

	// Setup mock expectations
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	agentsUseCase.On("ValidateJobBelongsToAgent", ctx, agent.ID, payload.JobID, testOrgID).
		Return(nil)

	// Transaction expectations
	txManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(ctx)
		}).Return(nil)

	agentsService.On("UnassignAgentFromJob", ctx, agent.ID, payload.JobID, testOrgID).
		Return(nil)

	jobsService.On("DeleteJob", ctx, payload.JobID, testOrgID).
		Return(nil)

	// Act
	err := useCase.ProcessJobComplete(ctx, clientID, payload, testOrgID)

	// Assert
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestSlackUseCase_ProcessJobComplete_JobNotFound(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.JobCompletePayload{
		JobID:  testJobID,
		Reason: "Task completed successfully",
	}

	// Setup mock expectations - job not found
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.None[*models.Job](), nil)

	// Act
	err := useCase.ProcessJobComplete(ctx, clientID, payload, testOrgID)

	// Assert - should return without error (job already completed)
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessJobComplete_AgentNotFound(t *testing.T) {
	// Test critical error path: Agent not found for WebSocket connection
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.JobCompletePayload{
		JobID:  testJobID,
		Reason: "Task completed successfully",
	}

	job := newTestJob(testOrgID, testJobID)

	// Setup mock expectations
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.None[*models.ActiveAgent](), nil)

	// Act
	err := useCase.ProcessJobComplete(ctx, clientID, payload, testOrgID)

	// Assert - should return error for missing agent
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent found for client")
	jobsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessJobComplete_AgentNotAssignedToJob(t *testing.T) {
	// Test critical error path: Agent validation fails
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.JobCompletePayload{
		JobID:  testJobID,
		Reason: "Task completed successfully",
	}

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, clientID, testOrgID)

	// Setup mock expectations
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	agentsUseCase.On("ValidateJobBelongsToAgent", ctx, agent.ID, payload.JobID, testOrgID).
		Return(errors.New("agent not assigned to job"))

	// Act
	err := useCase.ProcessJobComplete(ctx, clientID, payload, testOrgID)

	// Assert - should return error for validation failure
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent not assigned to job")
	jobsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
}

func TestSlackUseCase_ProcessAssistantMessage_Success(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.AssistantMessagePayload{
		JobID:              testJobID,
		Message:            "Here's my response",
		ProcessedMessageID: testMessageID,
	}

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, clientID, testOrgID)
	updatedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusCompleted)

	// Setup mock expectations
	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsUseCase.On("ValidateJobBelongsToAgent", ctx, agent.ID, payload.JobID, testOrgID).
		Return(nil)

	jobsService.On("UpdateJobTimestamp", ctx, job.ID, testOrgID).
		Return(nil)

	slackMessagesService.On("UpdateProcessedSlackMessage", ctx, payload.ProcessedMessageID, models.ProcessedSlackMessageStatusCompleted, job.SlackPayload.IntegrationID, testOrgID).
		Return(updatedMessage, nil)

	slackMessagesService.On("GetLatestProcessedMessageForJob", ctx, job.ID, job.SlackPayload.IntegrationID, testOrgID).
		Return(mo.Some(updatedMessage), nil)

	// Act
	err := useCase.ProcessAssistantMessage(ctx, clientID, payload, testOrgID)

	// Assert
	assert.NoError(t, err)
	agentsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessAssistantMessage_EmptyMessage(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.AssistantMessagePayload{
		JobID:              testJobID,
		Message:            "   ", // Empty/whitespace only
		ProcessedMessageID: testMessageID,
	}

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, clientID, testOrgID)
	updatedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusCompleted)

	// Setup mock expectations
	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsUseCase.On("ValidateJobBelongsToAgent", ctx, agent.ID, payload.JobID, testOrgID).
		Return(nil)

	jobsService.On("UpdateJobTimestamp", ctx, job.ID, testOrgID).
		Return(nil)

	slackMessagesService.On("UpdateProcessedSlackMessage", ctx, payload.ProcessedMessageID, models.ProcessedSlackMessageStatusCompleted, job.SlackPayload.IntegrationID, testOrgID).
		Return(updatedMessage, nil)

	slackMessagesService.On("GetLatestProcessedMessageForJob", ctx, job.ID, job.SlackPayload.IntegrationID, testOrgID).
		Return(mo.Some(updatedMessage), nil)

	// Act
	err := useCase.ProcessAssistantMessage(ctx, clientID, payload, testOrgID)

	// Assert - should handle empty message gracefully
	assert.NoError(t, err)
	agentsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessAssistantMessage_JobNotFound(t *testing.T) {
	// Test critical edge case: Job completed while agent was processing
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.AssistantMessagePayload{
		JobID:              testJobID,
		Message:            "Here's my response",
		ProcessedMessageID: testMessageID,
	}

	agent := newTestAgent(testAgentID, clientID, testOrgID)

	// Setup mock expectations
	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.None[*models.Job](), nil) // Job not found

	// Act
	err := useCase.ProcessAssistantMessage(ctx, clientID, payload, testOrgID)

	// Assert - should return without error (job already completed)
	assert.NoError(t, err)
	agentsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSystemMessage_ErrorMessage(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.SystemMessagePayload{
		Message: "ccagent encountered error: Something went wrong",
		JobID:   testJobID,
	}

	job := newTestJob(testOrgID, testJobID)
	agent := newTestAgent(testAgentID, clientID, testOrgID)

	// Setup mock expectations
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	agentsService.On("GetAgentByWSConnectionID", ctx, clientID, testOrgID).
		Return(mo.Some(agent), nil)

	// Transaction expectations for cleanup
	txManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(context.Context) error)
			_ = fn(ctx)
		}).Return(nil)

	agentsService.On("UnassignAgentFromJob", ctx, agent.ID, job.ID, testOrgID).
		Return(nil)

	jobsService.On("DeleteJob", ctx, job.ID, testOrgID).
		Return(nil)

	// Act
	err := useCase.ProcessSystemMessage(ctx, clientID, payload, testOrgID)

	// Assert
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
	agentsService.AssertExpectations(t)
	txManager.AssertExpectations(t)
}

func TestSlackUseCase_ProcessSystemMessage_RegularMessage(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.SystemMessagePayload{
		Message: "Processing your request...",
		JobID:   testJobID,
	}

	job := newTestJob(testOrgID, testJobID)

	// Setup mock expectations
	jobsService.On("GetJobByID", ctx, payload.JobID, testOrgID).
		Return(mo.Some(job), nil)

	jobsService.On("UpdateJobTimestamp", ctx, job.ID, testOrgID).
		Return(nil)

	// Act
	err := useCase.ProcessSystemMessage(ctx, clientID, payload, testOrgID)

	// Assert
	assert.NoError(t, err)
	jobsService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessQueuedJobs_Success(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()

	integration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)
	queuedJob := newTestJob(testOrgID, testJobID)
	queuedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusQueued)
	updatedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusInProgress)

	// Setup mock expectations
	slackIntegrationsService.On("GetAllSlackIntegrations", ctx).
		Return([]*models.SlackIntegration{integration}, nil)

	jobsService.On("GetJobsWithQueuedMessages", ctx, models.JobTypeSlack, testSlackIntegrationID, testOrgID).
		Return([]*models.Job{queuedJob}, nil)

	agentsUseCase.On("TryAssignJobToAgent", ctx, queuedJob.ID, testOrgID).
		Return(testWSConnectionID, true, nil) // Successfully assigned

	slackMessagesService.On("GetProcessedMessagesByJobIDAndStatus", ctx, queuedJob.ID, models.ProcessedSlackMessageStatusQueued, testSlackIntegrationID, testOrgID).
		Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)

	slackMessagesService.On("UpdateProcessedSlackMessage", ctx, queuedMessage.ID, models.ProcessedSlackMessageStatusInProgress, testSlackIntegrationID, testOrgID).
		Return(updatedMessage, nil)

	// Act
	err := useCase.ProcessQueuedJobs(ctx)

	// Assert
	assert.NoError(t, err)
	slackIntegrationsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
	slackMessagesService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessQueuedJobs_NoAgentsAvailable(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()

	integration := newTestSlackIntegration(testSlackIntegrationID, testOrgID)
	queuedJob := newTestJob(testOrgID, testJobID)

	// Setup mock expectations
	slackIntegrationsService.On("GetAllSlackIntegrations", ctx).
		Return([]*models.SlackIntegration{integration}, nil)

	jobsService.On("GetJobsWithQueuedMessages", ctx, models.JobTypeSlack, testSlackIntegrationID, testOrgID).
		Return([]*models.Job{queuedJob}, nil)

	agentsUseCase.On("TryAssignJobToAgent", ctx, queuedJob.ID, testOrgID).
		Return("", false, nil) // No agents available

	// Act
	err := useCase.ProcessQueuedJobs(ctx)

	// Assert
	assert.NoError(t, err)
	slackIntegrationsService.AssertExpectations(t)
	jobsService.AssertExpectations(t)
	agentsUseCase.AssertExpectations(t)
	// Should not process messages if no agent assigned
}

func TestSlackUseCase_ProcessProcessingMessage_Success(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.ProcessingMessagePayload{
		ProcessedMessageID: testMessageID,
	}

	processedMessage := newTestProcessedMessage(testMessageID, testJobID, testOrgID, models.ProcessedSlackMessageStatusInProgress)

	// Setup mock expectations
	slackMessagesService.On("GetProcessedSlackMessageByID", ctx, payload.ProcessedMessageID, testOrgID).
		Return(mo.Some(processedMessage), nil)

	// Act
	err := useCase.ProcessProcessingMessage(ctx, clientID, payload, testOrgID)

	// Assert
	assert.NoError(t, err)
	slackMessagesService.AssertExpectations(t)
}

func TestSlackUseCase_ProcessProcessingMessage_MessageNotFound(t *testing.T) {
	wsClient, agentsService, jobsService, slackMessagesService, slackIntegrationsService, txManager, agentsUseCase := setupBasicMocks()

	useCase := NewSlackUseCase(
		wsClient,
		agentsService,
		jobsService,
		slackMessagesService,
		slackIntegrationsService,
		txManager,
		agentsUseCase,
	)

	// Test data
	ctx := context.Background()
	clientID := testWSConnectionID
	payload := models.ProcessingMessagePayload{
		ProcessedMessageID: testMessageID,
	}

	// Setup mock expectations - message not found
	slackMessagesService.On("GetProcessedSlackMessageByID", ctx, payload.ProcessedMessageID, testOrgID).
		Return(mo.None[*models.ProcessedSlackMessage](), nil)

	// Act
	err := useCase.ProcessProcessingMessage(ctx, clientID, payload, testOrgID)

	// Assert - should return without error (job may have been completed)
	assert.NoError(t, err)
	slackMessagesService.AssertExpectations(t)
}