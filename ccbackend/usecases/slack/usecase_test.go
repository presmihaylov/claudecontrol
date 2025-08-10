package slack

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

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
	t.Run("success_new_conversation_agent_available", func(t *testing.T) {
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
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testProcessedID := testutils.GenerateProcessedMessageID()
		testAgentID := testutils.GenerateAgentID()

		event := models.SlackMessageEvent{
			User:     testUserID,
			Channel:  testChannelID,
			Text:     "Hello bot, help me with something",
			TS:       testThreadTS,
			ThreadTS: "", // New conversation
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

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testSlackToken,
		}

		connectedAgent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
		}

		processedMessage := &models.ProcessedSlackMessage{
			ID:                 testProcessedID,
			JobID:              testJobID,
			SlackTS:            testThreadTS,
			SlackChannelID:     testChannelID,
			TextContent:        "Hello bot, help me with something",
			SlackIntegrationID: testSlackIntegrationID,
			OrganizationID:     testOrgID,
			Status:             models.ProcessedSlackMessageStatusInProgress,
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetOrCreateJobForSlackThread", fixture.ctx, event.TS, event.Channel, event.User, testSlackIntegrationID, testOrgID).
			Return(jobResult, nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)
		fixture.mocks.wsClient.On("GetClientIDs").Return([]string{testWSConnectionID})
		fixture.mocks.agentsService.On("GetConnectedActiveAgents", fixture.ctx, testOrgID, []string{testWSConnectionID}).
			Return([]*models.ActiveAgent{connectedAgent}, nil)
		fixture.mocks.agentsUseCase.On("GetOrAssignAgentForJob", fixture.ctx, jobResult.Job, testThreadTS, testOrgID).
			Return(testWSConnectionID, nil)
		fixture.mocks.slackMessagesService.On("CreateProcessedSlackMessage", fixture.ctx, testJobID, testChannelID, testThreadTS, "Hello bot, help me with something", testSlackIntegrationID, testOrgID, models.ProcessedSlackMessageStatusInProgress).
			Return(processedMessage, nil)

		// Mock Slack client expectations for updating reaction
		fixture.mocks.slackClient.MockAddReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockRemoveReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockGetReactions = func(item clients.SlackItemRef, params clients.SlackGetReactionsParameters) ([]clients.SlackItemReaction, error) {
			return []clients.SlackItemReaction{}, nil
		}
		fixture.mocks.slackClient.MockAuthTest = func() (*clients.SlackAuthTestResponse, error) {
			return &clients.SlackAuthTestResponse{
				UserID: "B123456789",
				TeamID: testSlackIntegrationID,
			}, nil
		}

		// Mock expectations for sendStartConversationToAgent
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		fixture.mocks.slackClient.MockGetPermalink = func(params *clients.SlackPermalinkParameters) (string, error) {
			return "https://workspace.slack.com/archives/" + params.Channel + "/p" + params.TS, nil
		}
		fixture.mocks.slackClient.MockResolveMentionsInMessage = func(ctx context.Context, message string) string {
			return message // Return unchanged for simplicity
		}
		fixture.mocks.wsClient.On("SendMessage", testWSConnectionID, mock.AnythingOfType("models.BaseMessage")).
			Return(nil)

		// Execute
		err := fixture.useCase.ProcessSlackMessageEvent(fixture.ctx, event, testSlackIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
		fixture.mocks.wsClient.AssertExpectations(t)
		fixture.mocks.agentsService.AssertExpectations(t)
		fixture.mocks.agentsUseCase.AssertExpectations(t)
		fixture.mocks.slackMessagesService.AssertExpectations(t)
	})

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
	t.Run("success_valid_completion_reaction", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testUserID := testutils.GenerateSlackUserID()
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testThreadTS := testutils.GenerateSlackThreadTS()
		testAgentID := testutils.GenerateAgentID()
		testWSConnectionID := testutils.GenerateWSConnectionID()

		reactionName := "white_check_mark"

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

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testutils.GenerateSlackToken(),
		}

		agent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetJobBySlackThread", fixture.ctx, testThreadTS, testChannelID, testSlackIntegrationID, testOrgID).
			Return(mo.Some(job), nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)
		fixture.mocks.agentsService.On("GetAgentByJobID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(agent), nil)

		// Transaction expectations
		fixture.mocks.txManager.On("WithTransaction", fixture.ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				// Execute the transaction function
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(fixture.ctx) // Execute with same context for simplicity
			}).Return(nil)
		fixture.mocks.agentsService.On("UnassignAgentFromJob", fixture.ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		fixture.mocks.jobsService.On("DeleteJob", fixture.ctx, testJobID, testOrgID).Return(nil)

		// Mock Slack client for sending system message
		fixture.mocks.slackClient.MockPostMessage = func(channelID string, params clients.SlackMessageParams) (*clients.SlackPostMessageResponse, error) {
			return &clients.SlackPostMessageResponse{
				Channel:   channelID,
				Timestamp: "1234567890.123456",
			}, nil
		}

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

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
		fixture.mocks.agentsService.AssertExpectations(t)
		fixture.mocks.txManager.AssertExpectations(t)
	})

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
	t.Run("success_agent_completes_job", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testOrgID := testutils.GenerateOrganizationID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testAgentID := testutils.GenerateAgentID()
		testChannelID := testutils.GenerateSlackChannelID()
		testThreadTS := testutils.GenerateSlackThreadTS()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testUserID := testutils.GenerateSlackUserID()

		clientID := testWSConnectionID
		payload := models.JobCompletePayload{
			JobID:  testJobID,
			Reason: "Task completed successfully",
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

		agent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
		}

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testutils.GenerateSlackToken(),
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, payload.JobID, testOrgID).
			Return(mo.Some(job), nil)
		fixture.mocks.agentsService.On("GetAgentByWSConnectionID", fixture.ctx, clientID, testOrgID).
			Return(mo.Some(agent), nil)
		fixture.mocks.agentsUseCase.On("ValidateJobBelongsToAgent", fixture.ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)

		// Mock Slack client for updating reaction
		fixture.mocks.slackClient.MockAddReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockRemoveReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockGetReactions = func(item clients.SlackItemRef, params clients.SlackGetReactionsParameters) ([]clients.SlackItemReaction, error) {
			return []clients.SlackItemReaction{}, nil
		}
		fixture.mocks.slackClient.MockAuthTest = func() (*clients.SlackAuthTestResponse, error) {
			return &clients.SlackAuthTestResponse{
				UserID: "B123456789",
				TeamID: testSlackIntegrationID,
			}, nil
		}

		// Transaction expectations
		fixture.mocks.txManager.On("WithTransaction", fixture.ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				// Execute the transaction function
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(fixture.ctx) // Execute with same context for simplicity
			}).Return(nil)
		fixture.mocks.agentsService.On("UnassignAgentFromJob", fixture.ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		fixture.mocks.jobsService.On("DeleteJob", fixture.ctx, testJobID, testOrgID).Return(nil)

		// Mock system message sending
		fixture.mocks.slackClient.MockPostMessage = func(channelID string, params clients.SlackMessageParams) (*clients.SlackPostMessageResponse, error) {
			return &clients.SlackPostMessageResponse{
				Channel:   channelID,
				Timestamp: "1234567890.123456",
			}, nil
		}

		// Execute
		err := fixture.useCase.ProcessJobComplete(fixture.ctx, clientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.agentsService.AssertExpectations(t)
		fixture.mocks.agentsUseCase.AssertExpectations(t)
		fixture.mocks.txManager.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
	})

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
	t.Run("success_send_message", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testAgentID := testutils.GenerateAgentID()
		testOrgID := testutils.GenerateOrganizationID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testProcessedID := testutils.GenerateProcessedMessageID()
		testChannelID := testutils.GenerateSlackChannelID()
		testThreadTS := testutils.GenerateSlackThreadTS()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testUserID := testutils.GenerateSlackUserID()

		clientID := testWSConnectionID
		payload := models.AssistantMessagePayload{
			JobID:              testJobID,
			Message:            "Here's my response to your question",
			ProcessedMessageID: testProcessedID,
		}

		agent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
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

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testutils.GenerateSlackToken(),
		}

		updatedMessage := &models.ProcessedSlackMessage{
			ID:                 testProcessedID,
			JobID:              testJobID,
			SlackTS:            testutils.GenerateSlackMessageID(),
			SlackChannelID:     testChannelID,
			SlackIntegrationID: testSlackIntegrationID,
			OrganizationID:     testOrgID,
			Status:             models.ProcessedSlackMessageStatusCompleted,
		}

		// Configure expectations
		fixture.mocks.agentsService.On("GetAgentByWSConnectionID", fixture.ctx, clientID, testOrgID).
			Return(mo.Some(agent), nil)
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		fixture.mocks.agentsUseCase.On("ValidateJobBelongsToAgent", fixture.ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)
		fixture.mocks.jobsService.On("UpdateJobTimestamp", fixture.ctx, testJobID, testOrgID).Return(nil)
		fixture.mocks.slackMessagesService.On("UpdateProcessedSlackMessage", fixture.ctx, testProcessedID, models.ProcessedSlackMessageStatusCompleted, testSlackIntegrationID, testOrgID).
			Return(updatedMessage, nil)

		// Mock Slack client for posting message
		fixture.mocks.slackClient.MockPostMessage = func(channelID string, params clients.SlackMessageParams) (*clients.SlackPostMessageResponse, error) {
			return &clients.SlackPostMessageResponse{
				Channel:   channelID,
				Timestamp: "1234567890.123456",
			}, nil
		}

		// Mock reaction and bot user methods for updating message status
		fixture.mocks.slackClient.MockAddReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockRemoveReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockGetReactions = func(item clients.SlackItemRef, params clients.SlackGetReactionsParameters) ([]clients.SlackItemReaction, error) {
			return []clients.SlackItemReaction{}, nil
		}
		fixture.mocks.slackClient.MockAuthTest = func() (*clients.SlackAuthTestResponse, error) {
			return &clients.SlackAuthTestResponse{
				UserID: "B123456789",
				TeamID: testSlackIntegrationID,
			}, nil
		}

		// Check if latest message
		fixture.mocks.slackMessagesService.On("GetLatestProcessedMessageForJob", fixture.ctx, testJobID, testSlackIntegrationID, testOrgID).
			Return(mo.Some(updatedMessage), nil)

		// Execute
		err := fixture.useCase.ProcessAssistantMessage(fixture.ctx, clientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.agentsService.AssertExpectations(t)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.agentsUseCase.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
		fixture.mocks.slackMessagesService.AssertExpectations(t)
	})

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
	t.Run("success_process_queued_jobs", func(t *testing.T) {
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
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testProcessedID := testutils.GenerateProcessedMessageID()

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

		queuedMessage := &models.ProcessedSlackMessage{
			ID:                 testProcessedID,
			JobID:              testJobID,
			SlackTS:            testThreadTS,
			SlackChannelID:     testChannelID,
			TextContent:        "Queued message content",
			SlackIntegrationID: testSlackIntegrationID,
			OrganizationID:     testOrgID,
			Status:             models.ProcessedSlackMessageStatusQueued,
		}

		updatedMessage := &models.ProcessedSlackMessage{
			ID:                 testProcessedID,
			JobID:              testJobID,
			SlackTS:            testThreadTS,
			SlackChannelID:     testChannelID,
			TextContent:        "Queued message content",
			SlackIntegrationID: testSlackIntegrationID,
			OrganizationID:     testOrgID,
			Status:             models.ProcessedSlackMessageStatusInProgress,
		}

		// Configure expectations
		fixture.mocks.slackIntegrationsService.On("GetAllSlackIntegrations", fixture.ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		fixture.mocks.jobsService.On("GetJobsWithQueuedMessages", fixture.ctx, models.JobTypeSlack, testSlackIntegrationID, testOrgID).
			Return([]*models.Job{queuedJob}, nil)
		fixture.mocks.agentsUseCase.On("TryAssignJobToAgent", fixture.ctx, queuedJob.ID, testOrgID).
			Return(testWSConnectionID, true, nil)
		fixture.mocks.slackMessagesService.On("GetProcessedMessagesByJobIDAndStatus", fixture.ctx, testJobID, models.ProcessedSlackMessageStatusQueued, testSlackIntegrationID, testOrgID).
			Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)
		fixture.mocks.slackMessagesService.On("UpdateProcessedSlackMessage", fixture.ctx, testProcessedID, models.ProcessedSlackMessageStatusInProgress, testSlackIntegrationID, testOrgID).
			Return(updatedMessage, nil)

		// Mock Slack client for updating reaction
		fixture.mocks.slackClient.MockAddReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockRemoveReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockGetReactions = func(item clients.SlackItemRef, params clients.SlackGetReactionsParameters) ([]clients.SlackItemReaction, error) {
			return []clients.SlackItemReaction{}, nil
		}
		fixture.mocks.slackClient.MockAuthTest = func() (*clients.SlackAuthTestResponse, error) {
			return &clients.SlackAuthTestResponse{
				UserID: "B123456789",
				TeamID: testSlackIntegrationID,
			}, nil
		}

		// Mock expectations for sendStartConversationToAgent
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(queuedJob), nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(integration), nil)
		fixture.mocks.slackClient.MockGetPermalink = func(params *clients.SlackPermalinkParameters) (string, error) {
			return "https://workspace.slack.com/archives/" + params.Channel + "/p" + params.TS, nil
		}
		fixture.mocks.slackClient.MockResolveMentionsInMessage = func(ctx context.Context, message string) string {
			return message // Return unchanged for simplicity
		}
		fixture.mocks.wsClient.On("SendMessage", testWSConnectionID, mock.AnythingOfType("models.BaseMessage")).
			Return(nil)

		// Execute
		err := fixture.useCase.ProcessQueuedJobs(fixture.ctx)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.agentsUseCase.AssertExpectations(t)
		fixture.mocks.slackMessagesService.AssertExpectations(t)
		fixture.mocks.wsClient.AssertExpectations(t)
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
	t.Run("success_update_message_reaction", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data for this test case
		testProcessedID := testutils.GenerateProcessedMessageID()
		testJobID := testutils.GenerateJobID()
		testClientID := testutils.GenerateClientID()
		testOrgID := testutils.GenerateOrganizationID()
		testChannelID := testutils.GenerateSlackChannelID()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testMessageID := testutils.GenerateSlackMessageID()

		payload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedID,
		}

		processedMessage := &models.ProcessedSlackMessage{
			ID:                 testProcessedID,
			JobID:              testJobID,
			SlackTS:            testMessageID,
			SlackChannelID:     testChannelID,
			SlackIntegrationID: testSlackIntegrationID,
			OrganizationID:     testOrgID,
		}

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testutils.GenerateSlackToken(),
		}

		// Configure expectations
		fixture.mocks.slackMessagesService.On("GetProcessedSlackMessageByID", fixture.ctx, testProcessedID, testOrgID).
			Return(mo.Some(processedMessage), nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)

		// Mock Slack client for updating reaction
		fixture.mocks.slackClient.MockAddReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockRemoveReaction = func(name string, item clients.SlackItemRef) error {
			return nil
		}
		fixture.mocks.slackClient.MockGetReactions = func(item clients.SlackItemRef, params clients.SlackGetReactionsParameters) ([]clients.SlackItemReaction, error) {
			return []clients.SlackItemReaction{}, nil
		}
		fixture.mocks.slackClient.MockAuthTest = func() (*clients.SlackAuthTestResponse, error) {
			return &clients.SlackAuthTestResponse{
				UserID: "B123456789",
				TeamID: testSlackIntegrationID,
			}, nil
		}

		// Execute
		err := fixture.useCase.ProcessProcessingMessage(fixture.ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.slackMessagesService.AssertExpectations(t)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
	})

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
	t.Run("success_regular_system_message", func(t *testing.T) {
		// Setup
		fixture := setupSlackUseCaseTest(t)

		// Generate consistent test data
		testJobID := testutils.GenerateJobID()
		testOrgID := testutils.GenerateOrganizationID()
		testClientID := testutils.GenerateClientID()
		testChannelID := testutils.GenerateSlackChannelID()
		testThreadTS := testutils.GenerateSlackThreadTS()
		testSlackIntegrationID := testutils.GenerateSlackIntegrationID()
		testUserID := testutils.GenerateSlackUserID()

		payload := models.SystemMessagePayload{
			JobID:   testJobID,
			Message: "System notification message",
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

		slackIntegration := &models.SlackIntegration{
			ID:             testSlackIntegrationID,
			OrganizationID: testOrgID,
			SlackAuthToken: testutils.GenerateSlackToken(),
		}

		// Configure expectations
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		fixture.mocks.slackIntegrationsService.On("GetSlackIntegrationByID", fixture.ctx, testSlackIntegrationID).
			Return(mo.Some(slackIntegration), nil)
		fixture.mocks.jobsService.On("UpdateJobTimestamp", fixture.ctx, testJobID, testOrgID).Return(nil)

		// Mock Slack client for posting message
		fixture.mocks.slackClient.MockPostMessage = func(channelID string, params clients.SlackMessageParams) (*clients.SlackPostMessageResponse, error) {
			return &clients.SlackPostMessageResponse{
				Channel:   channelID,
				Timestamp: "1234567890.123456",
			}, nil
		}

		// Execute
		err := fixture.useCase.ProcessSystemMessage(fixture.ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.mocks.jobsService.AssertExpectations(t)
		fixture.mocks.slackIntegrationsService.AssertExpectations(t)
	})

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
