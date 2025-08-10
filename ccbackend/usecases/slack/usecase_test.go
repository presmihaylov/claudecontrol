package slack

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
	slackclient "ccbackend/clients/slack"
	"ccbackend/clients/socketio"
	"ccbackend/models"
	agentsservice "ccbackend/services/agents"
	"ccbackend/services/jobs"
	slackintegrations "ccbackend/services/slack_integrations"
	"ccbackend/services/slackmessages"
	"ccbackend/services/txmanager"
	"ccbackend/testutils"
	agentsusecase "ccbackend/usecases/agents"
)

// slackUseCaseTestFixture encapsulates test setup and mocks
type slackUseCaseTestFixture struct {
	useCase *SlackUseCase
	mocks   *slackUseCaseMocks
	ctx     context.Context
}

// slackUseCaseMocks contains all mock dependencies
type slackUseCaseMocks struct {
	wsClient                 *socketio.MockSocketIOClient
	agentsService            *agentsservice.MockAgentsService
	jobsService              *jobs.MockJobsService
	slackMessagesService     *slackmessages.MockSlackMessagesService
	slackIntegrationsService *slackintegrations.MockSlackIntegrationsService
	txManager                *txmanager.MockTransactionManager
	agentsUseCase            *agentsusecase.MockAgentsUseCase
	slackClient              *slackclient.MockSlackClient
}

// setupSlackUseCaseTest creates a new test fixture with all mocks initialized
func setupSlackUseCaseTest(t *testing.T) *slackUseCaseTestFixture {
	mocks := &slackUseCaseMocks{
		wsClient:                 new(socketio.MockSocketIOClient),
		agentsService:            new(agentsservice.MockAgentsService),
		jobsService:              new(jobs.MockJobsService),
		slackMessagesService:     new(slackmessages.MockSlackMessagesService),
		slackIntegrationsService: new(slackintegrations.MockSlackIntegrationsService),
		txManager:                new(txmanager.MockTransactionManager),
		agentsUseCase:            new(agentsusecase.MockAgentsUseCase),
		slackClient:              new(slackclient.MockSlackClient),
	}

	// Mock client factory that always returns the same mock client
	mockClientFactory := func(authToken string) clients.SlackClient {
		return mocks.slackClient
	}

	useCase := NewSlackUseCase(
		mocks.wsClient,
		mocks.agentsService,
		mocks.jobsService,
		mocks.slackMessagesService,
		mocks.slackIntegrationsService,
		mocks.txManager,
		mocks.agentsUseCase,
		mockClientFactory,
	)

	return &slackUseCaseTestFixture{
		useCase: useCase,
		mocks:   mocks,
		ctx:     context.Background(),
	}
}

func TestProcessSlackMessageEvent(t *testing.T) {
	t.Run("slack_integration_not_found", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testUserID := testutils.GenerateSlackUserID()
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testThreadTS := testutils.GenerateSlackThreadTS()

		event := models.SlackMessageEvent{
			User:     testUserID,
			Channel:  testChannelID,
			Text:     "Hello bot",
			TS:       testThreadTS,
			ThreadTS: "",
		}

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			SlackPayload: &models.SlackJobPayload{
				IntegrationID: testSlackIntegrationID,
				ChannelID:     testChannelID,
				ThreadTS:      testThreadTS,
				UserID:        testUserID,
			},
		}

		jobResult := &models.JobCreationResult{
			Job:    job,
			Status: models.JobCreationStatusCreated,
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetOrCreateJobForSlackThread", fixture.ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
			Return(jobResult, nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.None[*models.SlackIntegration](), nil)

		// Execute
		err := fixture.useCase.ProcessSlackMessageEvent(fixture.ctx, event, testSlackIntegrationID, testOrgID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slack integration not found")
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
	})
}

func TestProcessReactionAdded(t *testing.T) {
	t.Run("ignore_reaction_by_different_user", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testUserID := testutils.GenerateSlackUserID()
		testDifferentUserID := testutils.GenerateSlackUserID() // Different user
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testThreadTS := testutils.GenerateSlackThreadTS()

		reactionName := "white_check_mark"

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			SlackPayload: &models.SlackJobPayload{
				IntegrationID: testSlackIntegrationID,
				ChannelID:     testChannelID,
				ThreadTS:      testThreadTS,
				UserID:        testUserID, // Original job creator
			},
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetJobBySlackThread", fixture.ctx, testThreadTS, testChannelID, testSlackIntegrationID, testOrgID).
			Return(mo.Some(job), nil)

		// Execute
		err := fixture.useCase.ProcessReactionAdded(
			fixture.ctx,
			reactionName,
			testDifferentUserID,
			testChannelID,
			testThreadTS,
			testSlackIntegrationID,
			testOrgID,
		)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
	})

	t.Run("ignore_non_completion_reaction", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data
		testUserID := testutils.GenerateSlackUserID()
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testThreadTS := testutils.GenerateSlackThreadTS()

		reactionName := "thumbsup" // Not a completion emoji

		// Execute
		err := fixture.useCase.ProcessReactionAdded(
			fixture.ctx,
			reactionName,
			testUserID,
			testChannelID,
			testThreadTS,
			testSlackIntegrationID,
			testOrgID,
		)

		// Assert - should return without error and no mocks called
		assert.NoError(t, err)
	})
}

func TestProcessJobComplete(t *testing.T) {
	t.Run("job_not_found", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testOrgID := testutils.GenerateOrganizationID()
		testWSConnectionID := testutils.GenerateWSConnectionID()

		clientID := testWSConnectionID
		payload := models.JobCompletePayload{
			JobID:  testJobID,
			Reason: "Task completed successfully",
		}

		// Configure expectations - job not found
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, payload.JobID, testOrgID).
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := fixture.useCase.ProcessJobComplete(fixture.ctx, clientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
	})
}

func TestProcessAssistantMessage(t *testing.T) {
	t.Run("job_not_found", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testAgentID := testutils.GenerateAgentID()
		testOrgID := testutils.GenerateOrganizationID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testProcessedID := testutils.GenerateProcessedMessageID()

		clientID := testWSConnectionID
		payload := models.AssistantMessagePayload{
			JobID:              testJobID,
			Message:            "Here's my response",
			ProcessedMessageID: testProcessedID,
		}

		agent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: clientID,
			OrganizationID: testOrgID,
		}

		// Configure expectations
		fixture.mocks.agentsService.On("GetAgentByWSConnectionID", fixture.ctx, clientID, testOrgID).
			Return(mo.Some(agent), nil)
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, payload.JobID, testOrgID).
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := fixture.useCase.ProcessAssistantMessage(fixture.ctx, clientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.agentsService.AssertExpectations(t)
		fixture.mocks.jobsService.AssertExpectations(t)
	})
}

func TestProcessQueuedJobs(t *testing.T) {
	t.Run("no_integrations", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Configure expectations
		fixture.mocks.slackIntegrationsService.On("GetAllSlackIntegrations", fixture.ctx).
			Return([]*models.SlackIntegration{}, nil)

		// Execute
		err := fixture.useCase.ProcessQueuedJobs(fixture.ctx)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
	})

	t.Run("no_agents_available", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testUserID := testutils.GenerateSlackUserID()
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testThreadTS := testutils.GenerateSlackThreadTS()
		testSlackToken := testutils.GenerateSlackToken()

		integration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testSlackToken,
		}

		queuedJob := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			SlackPayload: &models.SlackJobPayload{
				IntegrationID: testSlackIntegrationID,
				ChannelID:     testChannelID,
				ThreadTS:      testThreadTS,
				UserID:        testUserID,
			},
		}

		// Configure expectations
		fixture.mocks.slackIntegrationsService.On("GetAllSlackIntegrations", fixture.ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		fixture.mocks.jobsService.On("GetJobsWithQueuedMessages", fixture.ctx, models.JobTypeSlack, testSlackIntegrationID, testOrgID).
			Return([]*models.Job{queuedJob}, nil)
		fixture.mocks.agentsUseCase.On("TryAssignJobToAgent", fixture.ctx, queuedJob.ID, testOrgID).
			Return("", false, nil) // No agents available

		// Execute
		err := fixture.useCase.ProcessQueuedJobs(fixture.ctx)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.agentsUseCase.AssertExpectations(t)
	})
}

func TestProcessProcessingMessage(t *testing.T) {
	t.Run("message_not_found", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testProcessedID := testutils.GenerateProcessedMessageID()
		testClientID := testutils.GenerateClientID()
		testOrgID := testutils.GenerateOrganizationID()

		payload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedID,
		}

		// Configure expectations
		fixture.mocks.slackMessagesService.On("GetProcessedSlackMessageByID", fixture.ctx, testProcessedID, testOrgID).
			Return(mo.None[*models.ProcessedSlackMessage](), nil)

		// Execute
		err := fixture.useCase.ProcessProcessingMessage(fixture.ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.slackMessagesService.AssertExpectations(t)
	})
}

func TestProcessSystemMessage(t *testing.T) {
	t.Run("job_not_found", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data
		testJobID := testutils.GenerateJobID()
		testOrgID := testutils.GenerateOrganizationID()
		testClientID := testutils.GenerateClientID()

		payload := models.SystemMessagePayload{
			JobID:   testJobID,
			Message: "System message",
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := fixture.useCase.ProcessSystemMessage(fixture.ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
	})
}