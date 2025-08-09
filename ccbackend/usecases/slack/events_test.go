package slack

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/models"
)

func TestProcessSlackMessageEvent(t *testing.T) {
	t.Run("NewConversation", func(t *testing.T) {
		t.Run("Success_WithAvailableAgent", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, mockTxManager, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
			
			event := createTestSlackEvent("<@U123456> help me with code", "")
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
				Return(testAgent, nil)
			
			mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
				Return(nil)
			
			mockJobsService.On("CreateJobWithTransaction", mock.Anything, mock.AnythingOfType("*models.Job"), mock.AnythingOfType("*models.ProcessedSlackMessage"), mock.AnythingOfType("*models.AgentJobAssignment")).
				Return(nil)
			
			mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
				Return(nil)
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.NoError(t, err)
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
			mockAgentsUseCase.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
			mockSocketClient.AssertExpectations(t)
		})
		
		t.Run("Success_NoAgentsAvailable_Queued", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, mockTxManager, _, mockAgentsUseCase := setupSlackUseCase(t)
			
			event := createTestSlackEvent("<@U123456> help me with code", "")
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
				Return(nil, fmt.Errorf("no agents available"))
			
			mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
				Return(nil)
			
			mockJobsService.On("CreateJobWithTransaction", mock.Anything, mock.AnythingOfType("*models.Job"), mock.AnythingOfType("*models.ProcessedSlackMessage"), nil).
				Return(nil)
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.NoError(t, err)
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
			mockAgentsUseCase.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
		
		t.Run("Error_InvalidSlackIntegration", func(t *testing.T) {
			// Setup
			useCase, _, _, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
			
			event := createTestSlackEvent("<@U123456> help", "")
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, "invalid_id").
				Return(nil, fmt.Errorf("not found"))
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, "invalid_id", testSlackIntegration.OrganizationID)
			
			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack integration not found")
			
			mockSlackIntegrationsService.AssertExpectations(t)
		})
		
		t.Run("Error_TransactionFailure", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, mockTxManager, _, mockAgentsUseCase := setupSlackUseCase(t)
			
			event := createTestSlackEvent("<@U123456> help", "")
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
				Return(testAgent, nil)
			
			mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
				Return(fmt.Errorf("transaction failed"))
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "transaction failed")
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
			mockAgentsUseCase.AssertExpectations(t)
			mockTxManager.AssertExpectations(t)
		})
	})
	
	t.Run("ThreadReply", func(t *testing.T) {
		t.Run("Success_ExistingJob_AgentAssigned", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, _ := setupSlackUseCase(t)
			
			threadTS := "1234567890.000001"
			event := createTestSlackEvent("<@U123456> follow up question", threadTS)
			job := createTestJob(threadTS)
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, threadTS, event.Channel, testSlackIntegration.ID).
				Return(job, nil)
			
			mockJobsService.On("GetProcessedSlackMessage", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, job.ID).
				Return([]*models.AgentJobAssignment{{AgentID: testAgent.ID, JobID: job.ID}}, nil)
			
			processedMsg := createTestProcessedMessage(job.ID, models.ProcessedSlackMessageStatusInProgress)
			mockJobsService.On("CreateProcessedSlackMessage", mock.Anything, mock.AnythingOfType("*models.ProcessedSlackMessage")).
				Run(func(args mock.Arguments) {
					msg := args.Get(1).(*models.ProcessedSlackMessage)
					msg.ID = processedMsg.ID
				}).
				Return(nil)
			
			mockSocketClient.On("SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
				Return(nil)
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.NoError(t, err)
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
			mockSocketClient.AssertExpectations(t)
		})
		
		t.Run("Error_NoExistingJob", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
			
			threadTS := "1234567890.000001"
			event := createTestSlackEvent("<@U123456> follow up", threadTS)
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, threadTS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no job found for thread")
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
		})
		
		t.Run("Error_NoAgentsAssigned", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
			
			threadTS := "1234567890.000001"
			event := createTestSlackEvent("<@U123456> follow up", threadTS)
			job := createTestJob(threadTS)
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, threadTS, event.Channel, testSlackIntegration.ID).
				Return(job, nil)
			
			mockJobsService.On("GetProcessedSlackMessage", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, job.ID).
				Return([]*models.AgentJobAssignment{}, nil)
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.Error(t, err)
			assert.Contains(t, err.Error(), "no agents assigned to job")
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
		})
	})
	
	t.Run("EdgeCases", func(t *testing.T) {
		t.Run("EmptyMessageText", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, mockTxManager, _, mockAgentsUseCase := setupSlackUseCase(t)
			
			event := createTestSlackEvent("", "")
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(nil, fmt.Errorf("not found"))
			
			mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
				Return(testAgent, nil)
			
			mockTxManager.On("WithTransaction", mock.Anything, mock.AnythingOfType("func(context.Context) error")).
				Return(nil)
			
			mockJobsService.On("CreateJobWithTransaction", mock.Anything, mock.AnythingOfType("*models.Job"), mock.AnythingOfType("*models.ProcessedSlackMessage"), mock.AnythingOfType("*models.AgentJobAssignment")).
				Return(nil)
			
			// Execute - should still process even with empty text
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.NoError(t, err)
		})
		
		t.Run("AlreadyProcessedMessage", func(t *testing.T) {
			// Setup
			useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
			
			threadTS := "1234567890.000001"
			event := createTestSlackEvent("<@U123456> test", threadTS)
			job := createTestJob(threadTS)
			processedMsg := createTestProcessedMessage(job.ID, models.ProcessedSlackMessageStatusCompleted)
			
			// Mock expectations
			mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
				Return(testSlackIntegration, nil)
			
			mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, threadTS, event.Channel, testSlackIntegration.ID).
				Return(job, nil)
			
			mockJobsService.On("GetProcessedSlackMessage", mock.Anything, event.TS, event.Channel, testSlackIntegration.ID).
				Return(processedMsg, nil)
			
			// Execute
			err := useCase.ProcessSlackMessageEvent(context.Background(), *event, testSlackIntegration.ID, testSlackIntegration.OrganizationID)
			
			// Assert
			require.NoError(t, err)
			
			mockSlackIntegrationsService.AssertExpectations(t)
			mockJobsService.AssertExpectations(t)
		})
	})
}

func TestProcessReactionAdded(t *testing.T) {
	t.Run("Success_CheckmarkReaction", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, mockSocketClient, _ := setupSlackUseCase(t)
		
		job := createTestJob("1234567890.000001")
		
		// Mock expectations
		mockJobsService.On("GetJobBySlackThreadTS", mock.Anything, job.SlackPayload.ThreadTS, job.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(job, nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, job.ID).
			Return(nil)
		
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, job.ID).
			Return([]*models.AgentJobAssignment{{AgentID: testAgent.ID, JobID: job.ID}}, nil)
		
		mockSocketClient.On("SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.ProcessReactionAdded(
			context.Background(),
			"white_check_mark",
			job.SlackPayload.UserID,
			job.SlackPayload.ChannelID,
			job.SlackPayload.ThreadTS,
			testSlackIntegration.ID,
			testSlackIntegration.OrganizationID,
		)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Ignore_NonCompletionReaction", func(t *testing.T) {
		// Setup
		useCase, _, _, _, _, _, _ := setupSlackUseCase(t)
		
		// Execute - should ignore non-completion reactions
		err := useCase.ProcessReactionAdded(
			context.Background(),
			"thumbsup",
			"U789012",
			"C123456",
			"1234567890.000001",
			testSlackIntegration.ID,
			testSlackIntegration.OrganizationID,
		)
		
		// Assert
		require.NoError(t, err)
	})
}

func TestProcessProcessingMessage(t *testing.T) {
	t.Run("Success_UpdateToInProgress", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		processingPayload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedMessage.ID,
		}
		
		// Mock expectations
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusInProgress).
			Return(nil)
		
		// Execute
		err := useCase.ProcessProcessingMessage(context.Background(), "client_123", processingPayload, testSlackIntegration.OrganizationID)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_InvalidMessageID", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		processingPayload := models.ProcessingMessagePayload{
			ProcessedMessageID: "invalid_id",
		}
		
		// Mock expectations
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, "invalid_id").
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessProcessingMessage(context.Background(), "client_123", processingPayload, testSlackIntegration.OrganizationID)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "processed message not found")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Success_AlreadyCompleted", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		completedMessage := createTestProcessedMessage(testJob.ID, models.ProcessedSlackMessageStatusCompleted)
		processingPayload := models.ProcessingMessagePayload{
			ProcessedMessageID: completedMessage.ID,
		}
		
		// Mock expectations
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, completedMessage.ID).
			Return(completedMessage, nil)
		
		// Should not update if already completed
		
		// Execute
		err := useCase.ProcessProcessingMessage(context.Background(), "client_123", processingPayload, testSlackIntegration.OrganizationID)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockJobsService.AssertNotCalled(t, "UpdateProcessedSlackMessageStatus")
	})
	
	t.Run("Error_UpdateStatusFailure", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		processingPayload := models.ProcessingMessagePayload{
			ProcessedMessageID: testProcessedMessage.ID,
		}
		
		// Mock expectations
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusInProgress).
			Return(fmt.Errorf("database error"))
		
		// Execute
		err := useCase.ProcessProcessingMessage(context.Background(), "client_123", processingPayload, testSlackIntegration.OrganizationID)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update processed message status")
		
		mockJobsService.AssertExpectations(t)
	})
}