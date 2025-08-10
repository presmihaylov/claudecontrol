package discord

import (
	"context"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
	discordclient "ccbackend/clients/discord"
	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	discordintegrations "ccbackend/services/discord_integrations"
	"ccbackend/services/discordmessages"
	"ccbackend/services/jobs"
	"ccbackend/services/txmanager"
	"ccbackend/testutils"
	agentsUseCase "ccbackend/usecases/agents"
)

// discordUseCaseTestFixture encapsulates test setup and mocks
type discordUseCaseTestFixture struct {
	useCase *DiscordUseCase
	mocks   *discordUseCaseMocks
	ctx     context.Context
}

// discordUseCaseMocks contains all mock dependencies
type discordUseCaseMocks struct {
	discordClient              *discordclient.MockDiscordClient
	wsClient                   *socketio.MockSocketIOClient
	agentsService              *agents.MockAgentsService
	jobsService                *jobs.MockJobsService
	discordMessagesService     *discordmessages.MockDiscordMessagesService
	discordIntegrationsService *discordintegrations.MockDiscordIntegrationsService
	txManager                  *txmanager.MockTransactionManager
	agentsUseCase              *agentsUseCase.MockAgentsUseCase
}

// setupDiscordUseCaseTest creates a new test fixture with all mocks initialized
func setupDiscordUseCaseTest(t *testing.T) *discordUseCaseTestFixture {
	mocks := &discordUseCaseMocks{
		discordClient:              new(discordclient.MockDiscordClient),
		wsClient:                   new(socketio.MockSocketIOClient),
		agentsService:              new(agents.MockAgentsService),
		jobsService:                new(jobs.MockJobsService),
		discordMessagesService:     new(discordmessages.MockDiscordMessagesService),
		discordIntegrationsService: new(discordintegrations.MockDiscordIntegrationsService),
		txManager:                  new(txmanager.MockTransactionManager),
		agentsUseCase:              new(agentsUseCase.MockAgentsUseCase),
	}

	useCase := NewDiscordUseCase(
		mocks.discordClient,
		mocks.wsClient,
		mocks.agentsService,
		mocks.jobsService,
		mocks.discordMessagesService,
		mocks.discordIntegrationsService,
		mocks.txManager,
		mocks.agentsUseCase,
	)

	return &discordUseCaseTestFixture{
		useCase: useCase,
		mocks:   mocks,
		ctx:     context.Background(),
	}
}

// assertAllExpectations asserts expectations on all mocks
func (f *discordUseCaseTestFixture) assertAllExpectations(t *testing.T) {
	f.mocks.discordClient.AssertExpectations(t)
	f.mocks.wsClient.AssertExpectations(t)
	f.mocks.agentsService.AssertExpectations(t)
	f.mocks.jobsService.AssertExpectations(t)
	f.mocks.discordMessagesService.AssertExpectations(t)
	f.mocks.discordIntegrationsService.AssertExpectations(t)
	f.mocks.txManager.AssertExpectations(t)
	f.mocks.agentsUseCase.AssertExpectations(t)
}



// Test ProcessDiscordMessageEvent

func TestProcessDiscordMessageEvent(t *testing.T) {
	t.Run("success_new_conversation_agent_available", func(t *testing.T) {
		// Setup
		fixture := setupDiscordUseCaseTest(t)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testBotID := testutils.GenerateDiscordBotID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testJobID := testutils.GenerateJobID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testProcessedID := testutils.GenerateProcessedMessageID()

		event := models.DiscordMessageEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			Content:   "Hello bot, help me with something",
			Mentions:  []string{testBotID},
			ThreadID:  nil, // New conversation
		}

		botUser := &clients.DiscordBotUser{
			ID:       testBotID,
			Username: testutils.GenerateDiscordBotUsername(),
			Bot:      true,
		}

		threadResponse := &clients.DiscordThreadResponse{
			ThreadID:   testThreadID,
			ThreadName: "CC Sesh #1234",
		}

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testMessageID,
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID,
				IntegrationID: testIntegrationID,
			},
		}

		jobResult := &models.JobCreationResult{
			Job:    job,
			Status: models.JobCreationStatusCreated,
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             testIntegrationID,
			OrganizationID: testOrgID,
			DiscordGuildID: testGuildID,
		}

		connectedAgent := &models.ActiveAgent{
			ID:             testutils.GenerateAgentID(),
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
		}

		processedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testMessageID,
			DiscordThreadID:      testThreadID,
			TextContent:          "Hello bot, help me with something",
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
			Status:               models.ProcessedDiscordMessageStatusInProgress,
		}

		// Configure expectations
		fixture.mocks.discordClient.On("GetBotUser").Return(botUser, nil)
		fixture.mocks.discordClient.On("CreatePublicThread", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(threadResponse, nil)
		fixture.mocks.jobsService.On("GetOrCreateJobForDiscordThread", fixture.ctx, testMessageID, testChannelID, testThreadID, testUserID, testIntegrationID, testOrgID).
			Return(jobResult, nil)
		fixture.mocks.discordIntegrationsService.On("GetDiscordIntegrationByID", fixture.ctx, testIntegrationID).
			Return(mo.Some(discordIntegration), nil)
		fixture.mocks.wsClient.On("GetClientIDs").Return([]string{testWSConnectionID})
		fixture.mocks.agentsService.On("GetConnectedActiveAgents", fixture.ctx, testOrgID, []string{testWSConnectionID}).
			Return([]*models.ActiveAgent{connectedAgent}, nil)
		fixture.mocks.agentsUseCase.On("GetOrAssignAgentForJob", fixture.ctx, jobResult.Job, testThreadID, testOrgID).
			Return(testWSConnectionID, nil)
		fixture.mocks.discordMessagesService.On("CreateProcessedDiscordMessage", fixture.ctx, testJobID, testMessageID, testThreadID, "Hello bot, help me with something", testIntegrationID, testOrgID, models.ProcessedDiscordMessageStatusInProgress).
			Return(processedMessage, nil)
		fixture.mocks.discordClient.On("AddReaction", testChannelID, testMessageID, EmojiHourglass).Return(nil)
		fixture.mocks.discordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		fixture.mocks.discordClient.On("AddReaction", testChannelID, testMessageID, EmojiEyes).Return(nil)
		fixture.mocks.discordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// Expect sendStartConversationToAgent
		fixture.mocks.jobsService.On("GetJobByID", fixture.ctx, testJobID, testOrgID).
			Return(mo.Some(jobResult.Job), nil).Maybe()
		fixture.mocks.wsClient.On("SendMessage", testWSConnectionID, mock.AnythingOfType("models.BaseMessage")).
			Return(nil)

		// Execute
		err := fixture.useCase.ProcessDiscordMessageEvent(fixture.ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.assertAllExpectations(t)
	})

	t.Run("bot_not_mentioned_ignore", func(t *testing.T) {
		// Setup
		fixture := setupDiscordUseCaseTest(t)

		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testBotID := testutils.GenerateDiscordBotID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()

		event := models.DiscordMessageEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			Content:   "Just a regular message",
			Mentions:  []string{}, // Bot not mentioned
			ThreadID:  nil,
		}

		botUser := &clients.DiscordBotUser{
			ID:       testBotID,
			Username: testutils.GenerateDiscordBotUsername(),
			Bot:      true,
		}

		// Configure expectations
		fixture.mocks.discordClient.On("GetBotUser").Return(botUser, nil)

		// Execute
		err := fixture.useCase.ProcessDiscordMessageEvent(fixture.ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		fixture.assertAllExpectations(t)
	})

	t.Run("no_agents_available_queue", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testBotID := testutils.GenerateDiscordBotID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testJobID := testutils.GenerateJobID()
		testProcessedID := testutils.GenerateProcessedMessageID()
		testBotUsername := testutils.GenerateDiscordBotUsername()

		event := models.DiscordMessageEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			Content:   "Hello bot, help me with something",
			Mentions:  []string{testBotID},
			ThreadID:  nil,
		}

		botUser := &clients.DiscordBotUser{
			ID:       testBotID,
			Username: testBotUsername,
			Bot:      true,
		}

		threadResponse := &clients.DiscordThreadResponse{
			ThreadID:   testThreadID,
			ThreadName: "CC Sesh #1234",
		}

		jobResult := &models.JobCreationResult{
			Job: &models.Job{
				ID:             testJobID,
				OrganizationID: testOrgID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     testMessageID,
					ChannelID:     testChannelID,
					ThreadID:      testThreadID,
					UserID:        testUserID,
					IntegrationID: testIntegrationID,
				},
			},
			Status: models.JobCreationStatusCreated,
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             testIntegrationID,
			OrganizationID: testOrgID,
			DiscordGuildID: testGuildID,
		}

		processedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testMessageID,
			DiscordThreadID:      testThreadID,
			TextContent:          "Hello bot, help me with something",
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
			Status:               models.ProcessedDiscordMessageStatusQueued,
		}

		// Configure expectations
		mockDiscordClient.On("GetBotUser").Return(botUser, nil)
		mockDiscordClient.On("CreatePublicThread", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(threadResponse, nil)
		mockJobsService.On("GetOrCreateJobForDiscordThread", ctx, testMessageID, testChannelID, testThreadID, testUserID, testIntegrationID, testOrgID).
			Return(jobResult, nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, testIntegrationID).
			Return(mo.Some(discordIntegration), nil)
		mockWSClient.On("GetClientIDs").Return([]string{})
		mockAgentsService.On("GetConnectedActiveAgents", ctx, testOrgID, []string{}).
			Return([]*models.ActiveAgent{}, nil)
		mockDiscordMessagesService.On("CreateProcessedDiscordMessage", ctx, testJobID, testMessageID, testThreadID, "Hello bot, help me with something", testIntegrationID, testOrgID, models.ProcessedDiscordMessageStatusQueued).
			Return(processedMessage, nil)
		mockDiscordClient.On("AddReaction", testChannelID, testMessageID, EmojiHourglass).Return(nil)
		mockDiscordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		mockDiscordClient.On("AddReaction", testChannelID, testMessageID, EmojiEyes).Return(nil)
		mockDiscordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// Execute
		err := useCase.ProcessDiscordMessageEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordClient.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockDiscordMessagesService.AssertExpectations(t)
	})

	t.Run("thread_reply_no_existing_job_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testBotID := testutils.GenerateDiscordBotID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testBotUsername := testutils.GenerateDiscordBotUsername()

		event := models.DiscordMessageEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			Content:   "Reply in thread",
			Mentions:  []string{testBotID},
			ThreadID:  &testThreadID, // Thread reply
		}

		botUser := &clients.DiscordBotUser{
			ID:       testBotID,
			Username: testBotUsername,
			Bot:      true,
		}

		// Configure expectations
		mockDiscordClient.On("GetBotUser").Return(botUser, nil)
		mockJobsService.On("GetJobByDiscordThread", ctx, testThreadID, testIntegrationID, testOrgID).
			Return(mo.None[*models.Job](), nil) // No existing job
		// Expect sendSystemMessage call for error
		mockDiscordClient.On("PostMessage", testChannelID, mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.Content == EmojiGear+" Error: new jobs can only be started from top-level messages" &&
				params.ThreadID != nil && *params.ThreadID == testThreadID
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)

		// Execute
		err := useCase.ProcessDiscordMessageEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordClient.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("discord_integration_not_found", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testBotID := testutils.GenerateDiscordBotID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testJobID := testutils.GenerateJobID()
		testBotUsername := testutils.GenerateDiscordBotUsername()

		event := models.DiscordMessageEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			Content:   "Hello bot",
			Mentions:  []string{testBotID},
			ThreadID:  nil,
		}

		botUser := &clients.DiscordBotUser{
			ID:       testBotID,
			Username: testBotUsername,
			Bot:      true,
		}

		threadResponse := &clients.DiscordThreadResponse{
			ThreadID:   testThreadID,
			ThreadName: "CC Sesh #1234",
		}

		jobResult := &models.JobCreationResult{
			Job: &models.Job{
				ID:             testJobID,
				OrganizationID: testOrgID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     testMessageID,
					ChannelID:     testChannelID,
					ThreadID:      testThreadID,
					UserID:        testUserID,
					IntegrationID: testIntegrationID,
				},
			},
			Status: models.JobCreationStatusCreated,
		}

		// Configure expectations
		mockDiscordClient.On("GetBotUser").Return(botUser, nil)
		mockDiscordClient.On("CreatePublicThread", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(threadResponse, nil)
		mockJobsService.On("GetOrCreateJobForDiscordThread", ctx, testMessageID, testChannelID, testThreadID, testUserID, testIntegrationID, testOrgID).
			Return(jobResult, nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, testIntegrationID).
			Return(mo.None[*models.DiscordIntegration](), nil) // Integration not found

		// Execute
		err := useCase.ProcessDiscordMessageEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "discord integration not found")
		mockDiscordClient.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
	})
}

func TestProcessDiscordReactionEvent(t *testing.T) {
	t.Run("success_valid_completion_reaction", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testJobID := testutils.GenerateJobID()
		testAgentID := testutils.GenerateAgentID()
		testWSConnectionID := testutils.GenerateWSConnectionID()

		event := models.DiscordReactionEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			EmojiName: EmojiCheckMark,
			ThreadID:  nil,
		}

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testMessageID,
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID,
				IntegrationID: testIntegrationID,
			},
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             testIntegrationID,
			OrganizationID: testOrgID,
			DiscordGuildID: testGuildID,
		}

		agent := &models.ActiveAgent{
			ID:             testAgentID,
			WSConnectionID: testWSConnectionID,
			OrganizationID: testOrgID,
		}

		// Configure expectations
		mockJobsService.On("GetJobByDiscordThread", ctx, testMessageID, testIntegrationID, testOrgID).
			Return(mo.Some(job), nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, testIntegrationID).
			Return(mo.Some(discordIntegration), nil)
		mockAgentsService.On("GetAgentByJobID", ctx, testJobID, testOrgID).
			Return(mo.Some(agent), nil)

		// Transaction expectations
		mockTxManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				// Execute the transaction function
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(ctx) // Execute with same context for simplicity
			}).Return(nil)
		mockAgentsService.On("UnassignAgentFromJob", ctx, testAgentID, testJobID, testOrgID).Return(nil)
		mockJobsService.On("DeleteJob", ctx, testJobID, testOrgID).Return(nil)

		// Discord reaction update
		mockDiscordClient.On("AddReaction", testChannelID, testMessageID, EmojiCheckMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// System message
		mockDiscordClient.On("PostMessage", testChannelID, mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.Content == EmojiGear+" Job manually marked as complete"
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)

		// Execute
		err := useCase.ProcessDiscordReactionEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("ignore_reaction_by_different_user", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testJobID := testutils.GenerateJobID()
		testDifferentUserID := testutils.GenerateDiscordUserID() // Different user

		event := models.DiscordReactionEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testDifferentUserID, // Different from job creator
			EmojiName: EmojiCheckMark,
			ThreadID:  nil,
		}

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testMessageID,
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID, // Original job creator
				IntegrationID: testIntegrationID,
			},
		}

		// Configure expectations
		mockJobsService.On("GetJobByDiscordThread", ctx, testMessageID, testIntegrationID, testOrgID).
			Return(mo.Some(job), nil)

		// Execute
		err := useCase.ProcessDiscordReactionEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("ignore_no_job_found", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testUserID := testutils.GenerateDiscordUserID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()

		event := models.DiscordReactionEvent{
			MessageID: testMessageID,
			ChannelID: testChannelID,
			GuildID:   testGuildID,
			UserID:    testUserID,
			EmojiName: EmojiCheckMark,
			ThreadID:  nil,
		}

		// Configure expectations
		mockJobsService.On("GetJobByDiscordThread", ctx, testMessageID, testIntegrationID, testOrgID).
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := useCase.ProcessDiscordReactionEvent(ctx, event, testIntegrationID, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessProcessingMessage(t *testing.T) {
	t.Run("success_update_message_reaction", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testProcessedID := testutils.GenerateProcessedMessageID()
		testJobID := testutils.GenerateJobID()
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testUserID := testutils.GenerateDiscordUserID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testClientID := testutils.GenerateClientID()
		testTopLevelMessageID := testutils.GenerateDiscordMessageID() // Different from processed message

		payload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedID,
		}

		processedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testMessageID, // Not the top-level message
			DiscordThreadID:      testThreadID,
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
		}

		job := &models.Job{
			ID:             testJobID,
			OrganizationID: testOrgID,
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testTopLevelMessageID, // Different from processed message
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID,
				IntegrationID: testIntegrationID,
			},
		}

		// Configure expectations
		mockDiscordMessagesService.On("GetProcessedDiscordMessageByID", ctx, testProcessedID, testOrgID).
			Return(mo.Some(processedMessage), nil)
		mockJobsService.On("GetJobByID", ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		mockDiscordClient.On("AddReaction", testThreadID, testMessageID, EmojiEyes).Return(nil)
		mockDiscordClient.On("RemoveReaction", testThreadID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// Execute
		err := useCase.ProcessProcessingMessage(ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordMessagesService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("message_not_found_skip", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testProcessedID := testutils.GenerateProcessedMessageID()
		testClientID := testutils.GenerateClientID()
		testOrgID := testutils.GenerateOrganizationID()

		payload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedID,
		}

		// Configure expectations
		mockDiscordMessagesService.On("GetProcessedDiscordMessageByID", ctx, testProcessedID, testOrgID).
			Return(mo.None[*models.ProcessedDiscordMessage](), nil)

		// Execute
		err := useCase.ProcessProcessingMessage(ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordMessagesService.AssertExpectations(t)
	})

	t.Run("job_not_found_skip", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testProcessedID := testutils.GenerateProcessedMessageID()
		testJobID := testutils.GenerateJobID()
		testMessageID := testutils.GenerateDiscordMessageID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testOrgID := testutils.GenerateOrganizationID()
		testClientID := testutils.GenerateClientID()

		payload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedID,
		}

		processedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testMessageID,
			DiscordThreadID:      testThreadID,
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
		}

		// Configure expectations
		mockDiscordMessagesService.On("GetProcessedDiscordMessageByID", ctx, testProcessedID, testOrgID).
			Return(mo.Some(processedMessage), nil)
		mockJobsService.On("GetJobByID", ctx, testJobID, testOrgID).
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := useCase.ProcessProcessingMessage(ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordMessagesService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessAssistantMessage(t *testing.T) {
	t.Run("success_send_message", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testProcessedID := testutils.GenerateProcessedMessageID()
		testAgentID := testutils.GenerateAgentID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testOrgID := testutils.GenerateOrganizationID()
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testUserID := testutils.GenerateDiscordUserID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testClientID := testutils.GenerateClientID()
		testNonTopLevelMessageID := testutils.GenerateDiscordMessageID() // Different from job's top-level message

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
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testMessageID,
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID,
				IntegrationID: testIntegrationID,
			},
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             testIntegrationID,
			OrganizationID: testOrgID,
			DiscordGuildID: testGuildID,
		}

		updatedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testNonTopLevelMessageID, // Not top-level
			DiscordThreadID:      testThreadID,
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
			Status:               models.ProcessedDiscordMessageStatusCompleted,
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, testClientID, testOrgID).
			Return(mo.Some(agent), nil)
		mockJobsService.On("GetJobByID", ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		mockAgentsUseCase.On("ValidateJobBelongsToAgent", ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, testIntegrationID).
			Return(mo.Some(discordIntegration), nil)
		mockDiscordClient.On("PostMessage", testThreadID, clients.DiscordMessageParams{
			Content: "Here's my response to your question",
		}).Return(&clients.DiscordPostMessageResponse{}, nil)
		mockJobsService.On("UpdateJobTimestamp", ctx, testJobID, testOrgID).Return(nil)
		mockDiscordMessagesService.On("UpdateProcessedDiscordMessage", ctx, testProcessedID, models.ProcessedDiscordMessageStatusCompleted, testIntegrationID, testOrgID).
			Return(updatedMessage, nil)
		// Reaction update for non-top-level message
		mockDiscordClient.On("AddReaction", testThreadID, testNonTopLevelMessageID, EmojiCheckMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", testThreadID, testNonTopLevelMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		// Check if latest message
		mockDiscordMessagesService.On("GetLatestProcessedMessageForJob", ctx, testJobID, testIntegrationID, testOrgID).
			Return(mo.Some(updatedMessage), nil)
		// Add hand emoji to top-level message
		mockDiscordClient.On("AddReaction", testChannelID, testMessageID, EmojiRaisedHand).Return(nil)
		mockDiscordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// Execute
		err := useCase.ProcessAssistantMessage(ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
		mockDiscordMessagesService.AssertExpectations(t)
	})

	t.Run("handle_empty_message", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Generate consistent test data for this test case
		testJobID := testutils.GenerateJobID()
		testProcessedID := testutils.GenerateProcessedMessageID()
		testAgentID := testutils.GenerateAgentID()
		testWSConnectionID := testutils.GenerateWSConnectionID()
		testOrgID := testutils.GenerateOrganizationID()
		testMessageID := testutils.GenerateDiscordMessageID()
		testChannelID := testutils.GenerateDiscordChannelID()
		testThreadID := testutils.GenerateDiscordThreadID()
		testUserID := testutils.GenerateDiscordUserID()
		testIntegrationID := testutils.GenerateDiscordIntegrationID()
		testGuildID := testutils.GenerateDiscordGuildID()
		testClientID := testutils.GenerateClientID()
		testNonTopLevelMessageID := testutils.GenerateDiscordMessageID()

		payload := models.AssistantMessagePayload{
			JobID:              testJobID,
			Message:            "   ", // Empty/whitespace message
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
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     testMessageID,
				ChannelID:     testChannelID,
				ThreadID:      testThreadID,
				UserID:        testUserID,
				IntegrationID: testIntegrationID,
			},
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             testIntegrationID,
			OrganizationID: testOrgID,
			DiscordGuildID: testGuildID,
		}

		updatedMessage := &models.ProcessedDiscordMessage{
			ID:                   testProcessedID,
			JobID:                testJobID,
			DiscordMessageID:     testNonTopLevelMessageID,
			DiscordThreadID:      testThreadID,
			DiscordIntegrationID: testIntegrationID,
			OrganizationID:       testOrgID,
			Status:               models.ProcessedDiscordMessageStatusCompleted,
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, testClientID, testOrgID).
			Return(mo.Some(agent), nil)
		mockJobsService.On("GetJobByID", ctx, testJobID, testOrgID).
			Return(mo.Some(job), nil)
		mockAgentsUseCase.On("ValidateJobBelongsToAgent", ctx, testAgentID, testJobID, testOrgID).
			Return(nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, testIntegrationID).
			Return(mo.Some(discordIntegration), nil)
		// Expect fallback message for empty content
		mockDiscordClient.On("PostMessage", testThreadID, clients.DiscordMessageParams{
			Content: "(agent sent empty response)",
		}).Return(&clients.DiscordPostMessageResponse{}, nil)
		mockJobsService.On("UpdateJobTimestamp", ctx, testJobID, testOrgID).Return(nil)
		mockDiscordMessagesService.On("UpdateProcessedDiscordMessage", ctx, testProcessedID, models.ProcessedDiscordMessageStatusCompleted, testIntegrationID, testOrgID).
			Return(updatedMessage, nil)
		mockDiscordClient.On("AddReaction", testThreadID, testNonTopLevelMessageID, EmojiCheckMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", testThreadID, testNonTopLevelMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		mockDiscordMessagesService.On("GetLatestProcessedMessageForJob", ctx, testJobID, testIntegrationID, testOrgID).
			Return(mo.Some(updatedMessage), nil)
		mockDiscordClient.On("AddReaction", testChannelID, testMessageID, EmojiRaisedHand).Return(nil)
		mockDiscordClient.On("RemoveReaction", testChannelID, testMessageID, mock.AnythingOfType("string")).
			Return(nil).
			Maybe()

		// Execute
		err := useCase.ProcessAssistantMessage(ctx, testClientID, payload, testOrgID)

		// Assert
		assert.NoError(t, err)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("agent_not_found", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.AssistantMessagePayload{
			JobID:              "job-111",
			Message:            "Response",
			ProcessedMessageID: "processed-123",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "client-123", "org-456").
			Return(mo.None[*models.ActiveAgent](), nil)

		// Execute
		err := useCase.ProcessAssistantMessage(ctx, "client-123", payload, "org-456")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agent found for client")
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("job_not_found_skip", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.AssistantMessagePayload{
			JobID:              "job-111",
			Message:            "Response",
			ProcessedMessageID: "processed-123",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-111",
			WSConnectionID: "ws-222",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "client-123", "org-456").
			Return(mo.Some(agent), nil)
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := useCase.ProcessAssistantMessage(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessSystemMessage(t *testing.T) {
	t.Run("success_regular_system_message", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.SystemMessagePayload{
			JobID:   "job-111",
			Message: "System notification message",
		}

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
			DiscordGuildID: "guild-789",
		}

		// Configure expectations
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, "discord-int-123").
			Return(mo.Some(discordIntegration), nil)
		mockDiscordClient.On("PostMessage", "channel-456", mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.Content == EmojiGear+" System notification message" &&
				params.ThreadID != nil && *params.ThreadID == "thread-123"
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)
		mockJobsService.On("UpdateJobTimestamp", ctx, "job-111", "org-456").Return(nil)

		// Execute
		err := useCase.ProcessSystemMessage(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("success_agent_error_message_cleanup", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.SystemMessagePayload{
			JobID:   "job-111",
			Message: "ccagent encountered error: Something went wrong",
		}

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		agent := &models.ActiveAgent{
			ID:             "agent-111",
			WSConnectionID: "ws-222",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil)
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "client-123", "org-456").
			Return(mo.Some(agent), nil)
		// Expected CleanupFailedDiscordJob behavior
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, "discord-int-123").
			Return(mo.Some(&models.DiscordIntegration{
				ID:             "discord-int-123",
				OrganizationID: "org-456",
				DiscordGuildID: "guild-789",
			}), nil)
		mockDiscordClient.On("PostMessage", "channel-456", mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.ThreadID != nil && *params.ThreadID == "thread-123"
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)
		mockDiscordClient.On("AddReaction", "channel-456", "msg-123", EmojiCrossMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", "channel-456", "msg-123", mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		mockTxManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(ctx)
			}).Return(nil)
		mockAgentsService.On("UnassignAgentFromJob", ctx, "agent-111", "job-111", "org-456").Return(nil)
		mockJobsService.On("DeleteJob", ctx, "job-111", "org-456").Return(nil)

		// Execute
		err := useCase.ProcessSystemMessage(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("job_not_found_skip", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.SystemMessagePayload{
			JobID:   "job-111",
			Message: "System message",
		}

		// Configure expectations
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := useCase.ProcessSystemMessage(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessJobComplete(t *testing.T) {
	t.Run("success_agent_completes_job", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.JobCompletePayload{
			JobID:  "job-111",
			Reason: "Task completed successfully",
		}

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		agent := &models.ActiveAgent{
			ID:             "agent-111",
			WSConnectionID: "ws-222",
			OrganizationID: "org-456",
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
			DiscordGuildID: "guild-789",
		}

		// Configure expectations
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil)
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "client-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsUseCase.On("ValidateJobBelongsToAgent", ctx, "agent-111", "job-111", "org-456").
			Return(nil)
		mockDiscordClient.On("AddReaction", "channel-456", "msg-123", EmojiCheckMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", "channel-456", "msg-123", mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		mockTxManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(ctx)
			}).Return(nil)
		mockAgentsService.On("UnassignAgentFromJob", ctx, "agent-111", "job-111", "org-456").Return(nil)
		mockJobsService.On("DeleteJob", ctx, "job-111", "org-456").Return(nil)
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, "discord-int-123").
			Return(mo.Some(discordIntegration), nil)
		mockDiscordClient.On("PostMessage", "channel-456", mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.Content == EmojiGear+" Task completed successfully"
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)

		// Execute
		err := useCase.ProcessJobComplete(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
	})

	t.Run("job_not_found_skip", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		payload := models.JobCompletePayload{
			JobID:  "job-111",
			Reason: "Task completed",
		}

		// Configure expectations
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.None[*models.Job](), nil)

		// Execute
		err := useCase.ProcessJobComplete(ctx, "client-123", payload, "org-456")

		// Assert
		assert.NoError(t, err)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessQueuedJobs(t *testing.T) {
	t.Run("success_process_queued_jobs", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		integration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
		}

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		queuedMessage := &models.ProcessedDiscordMessage{
			ID:                   "processed-123",
			JobID:                "job-111",
			DiscordMessageID:     "msg-123", // Same as job message ID (new conversation)
			DiscordThreadID:      "thread-123",
			TextContent:          "Queued message content",
			DiscordIntegrationID: "discord-int-123",
			OrganizationID:       "org-456",
			Status:               models.ProcessedDiscordMessageStatusQueued,
		}

		updatedMessage := &models.ProcessedDiscordMessage{
			ID:                   "processed-123",
			JobID:                "job-111",
			DiscordMessageID:     "msg-123",
			DiscordThreadID:      "thread-123",
			TextContent:          "Queued message content",
			DiscordIntegrationID: "discord-int-123",
			OrganizationID:       "org-456",
			Status:               models.ProcessedDiscordMessageStatusInProgress,
		}

		// Configure expectations
		mockDiscordIntegrationsService.On("GetAllDiscordIntegrations", ctx).
			Return([]*models.DiscordIntegration{integration}, nil)
		mockJobsService.On("GetJobsWithQueuedMessages", ctx, models.JobTypeDiscord, "discord-int-123", "org-456").
			Return([]*models.Job{job}, nil)
		mockAgentsUseCase.On("TryAssignJobToAgent", ctx, "job-111", "org-456").
			Return("client-123", true, nil)
		mockDiscordMessagesService.On("GetProcessedMessagesByJobIDAndStatus", ctx, "job-111", models.ProcessedDiscordMessageStatusQueued, "discord-int-123", "org-456").
			Return([]*models.ProcessedDiscordMessage{queuedMessage}, nil)
		mockDiscordMessagesService.On("UpdateProcessedDiscordMessage", ctx, "processed-123", models.ProcessedDiscordMessageStatusInProgress, "discord-int-123", "org-456").
			Return(updatedMessage, nil)
		// Update reaction for new conversation (top-level message)
		mockDiscordClient.On("AddReaction", "channel-456", "msg-123", EmojiEyes).Return(nil)
		mockDiscordClient.On("RemoveReaction", "channel-456", "msg-123", mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		// Send start conversation to agent
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil).Maybe()
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, "discord-int-123").
			Return(mo.Some(integration), nil).Maybe()
		mockWSClient.On("SendMessage", "client-123", mock.AnythingOfType("models.BaseMessage")).Return(nil)

		// Execute
		err := useCase.ProcessQueuedJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockDiscordMessagesService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
	})

	t.Run("no_discord_integrations", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		// Configure expectations
		mockDiscordIntegrationsService.On("GetAllDiscordIntegrations", ctx).
			Return([]*models.DiscordIntegration{}, nil)

		// Execute
		err := useCase.ProcessQueuedJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockDiscordIntegrationsService.AssertExpectations(t)
	})

	t.Run("no_queued_jobs", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		integration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockDiscordIntegrationsService.On("GetAllDiscordIntegrations", ctx).
			Return([]*models.DiscordIntegration{integration}, nil)
		mockJobsService.On("GetJobsWithQueuedMessages", ctx, models.JobTypeDiscord, "discord-int-123", "org-456").
			Return([]*models.Job{}, nil)

		// Execute
		err := useCase.ProcessQueuedJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("still_no_agents_available", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		integration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
		}

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		// Configure expectations
		mockDiscordIntegrationsService.On("GetAllDiscordIntegrations", ctx).
			Return([]*models.DiscordIntegration{integration}, nil)
		mockJobsService.On("GetJobsWithQueuedMessages", ctx, models.JobTypeDiscord, "discord-int-123", "org-456").
			Return([]*models.Job{job}, nil)
		mockAgentsUseCase.On("TryAssignJobToAgent", ctx, "job-111", "org-456").
			Return("", false, nil) // No agent assigned

		// Execute
		err := useCase.ProcessQueuedJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
	})
}

func TestCleanupFailedDiscordJob(t *testing.T) {
	t.Run("success_cleanup_job", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: &models.DiscordJobPayload{
				MessageID:     "msg-123",
				ChannelID:     "channel-456",
				ThreadID:      "thread-123",
				UserID:        "user-abc",
				IntegrationID: "discord-int-123",
			},
		}

		discordIntegration := &models.DiscordIntegration{
			ID:             "discord-int-123",
			OrganizationID: "org-456",
			DiscordGuildID: "guild-789",
		}

		// Configure expectations
		mockDiscordIntegrationsService.On("GetDiscordIntegrationByID", ctx, "discord-int-123").
			Return(mo.Some(discordIntegration), nil)
		mockDiscordClient.On("PostMessage", "channel-456", mock.MatchedBy(func(params clients.DiscordMessageParams) bool {
			return params.ThreadID != nil && *params.ThreadID == "thread-123"
		})).
			Return(&clients.DiscordPostMessageResponse{}, nil)
		mockDiscordClient.On("AddReaction", "channel-456", "msg-123", EmojiCrossMark).Return(nil)
		mockDiscordClient.On("RemoveReaction", "channel-456", "msg-123", mock.AnythingOfType("string")).
			Return(nil).
			Maybe()
		mockTxManager.On("WithTransaction", ctx, mock.AnythingOfType("func(context.Context) error")).
			Run(func(args mock.Arguments) {
				txFunc := args.Get(1).(func(context.Context) error)
				txFunc(ctx)
			}).Return(nil)
		mockAgentsService.On("UnassignAgentFromJob", ctx, "agent-111", "job-111", "org-456").Return(nil)
		mockJobsService.On("DeleteJob", ctx, "job-111", "org-456").Return(nil)

		// Execute
		err := useCase.CleanupFailedDiscordJob(ctx, job, "agent-111", "Agent failed to process")

		// Assert
		assert.NoError(t, err)
		mockDiscordIntegrationsService.AssertExpectations(t)
		mockDiscordClient.AssertExpectations(t)
		mockTxManager.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("job_no_discord_payload", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockDiscordClient := new(discordclient.MockDiscordClient)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockAgentsService := new(agents.MockAgentsService)
		mockJobsService := new(jobs.MockJobsService)
		mockDiscordMessagesService := new(discordmessages.MockDiscordMessagesService)
		mockDiscordIntegrationsService := new(discordintegrations.MockDiscordIntegrationsService)
		mockTxManager := new(txmanager.MockTransactionManager)
		mockAgentsUseCase := new(agentsUseCase.MockAgentsUseCase)

		useCase := NewDiscordUseCase(
			mockDiscordClient,
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockDiscordMessagesService,
			mockDiscordIntegrationsService,
			mockTxManager,
			mockAgentsUseCase,
		)

		job := &models.Job{
			ID:             "job-111",
			OrganizationID: "org-456",
			DiscordPayload: nil, // No Discord payload
		}

		// Execute
		err := useCase.CleanupFailedDiscordJob(ctx, job, "agent-111", "Agent failed")

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job has no Discord payload")
	})
}
