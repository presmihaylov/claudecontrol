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

func TestProcessQueuedJobs(t *testing.T) {
	t.Run("Success_ProcessSingleQueuedMessage", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
		
		queuedMessage := createTestProcessedMessage(testJob.ID, models.ProcessedSlackMessageStatusQueued)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)
		
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{testAgent}, nil)
		
		// Agent is available, should process the message
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, testJob.ID).
			Return(testAgent, nil)
		
		mockJobsService.On("CreateAgentJobAssignment", mock.Anything, testAgent.ID, testJob.ID).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, queuedMessage.ID, models.ProcessedSlackMessageStatusInProgress).
			Return(nil)
		
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Success_ProcessMultipleQueuedMessages", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
		
		job1 := createTestJob("1234567890.000001")
		job2 := createTestJob("1234567890.000002")
		queuedMessage1 := createTestProcessedMessage(job1.ID, models.ProcessedSlackMessageStatusQueued)
		queuedMessage2 := createTestProcessedMessage(job2.ID, models.ProcessedSlackMessageStatusQueued)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage1, queuedMessage2}, nil)
		
		// First message
		mockJobsService.On("GetJobByID", mock.Anything, job1.ID).
			Return(job1, nil)
		
		// Second message
		mockJobsService.On("GetJobByID", mock.Anything, job2.ID).
			Return(job2, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil).Times(2)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{testAgent}, nil)
		
		// Both messages should be processed
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
			Return(testAgent, nil).Times(2)
		
		mockJobsService.On("CreateAgentJobAssignment", mock.Anything, testAgent.ID, mock.AnythingOfType("string")).
			Return(nil).Times(2)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, mock.AnythingOfType("string"), models.ProcessedSlackMessageStatusInProgress).
			Return(nil).Times(2)
		
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil).Times(2)
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Success_NoQueuedMessages", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		// Mock expectations - no queued messages
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{}, nil)
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Success_NoAvailableAgents", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, _, mockAgentsUseCase := setupSlackUseCase(t)
		
		queuedMessage := createTestProcessedMessage(testJob.ID, models.ProcessedSlackMessageStatusQueued)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)
		
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		// No agents available
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{}, nil)
		
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, testJob.ID).
			Return(nil, fmt.Errorf("no agents available"))
		
		// Message should remain queued
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
	})
	
	t.Run("Success_NewConversationDetection", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
		
		// Queued message with thread TS same as message TS (new conversation)
		queuedMessage := &models.ProcessedSlackMessage{
			ID:                 core.NewID("psm"),
			SlackIntegrationID: testSlackIntegration.ID,
			JobID:              testJob.ID,
			SlackTS:            "1234567890.123456",
			SlackChannelID:     "C123456",
			TextContent:        "test message",
			Status:             models.ProcessedSlackMessageStatusQueued,
			OrganizationID:     testSlackIntegration.OrganizationID,
		}
		
		jobForNewConvo := &models.Job{
			ID: testJob.ID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "1234567890.123456", // Same as message TS
				ChannelID:     "C123456",
				IntegrationID: testSlackIntegration.ID,
				UserID:        "U789012",
			},
		}
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)
		
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(jobForNewConvo, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{testAgent}, nil)
		
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, testJob.ID).
			Return(testAgent, nil)
		
		mockJobsService.On("CreateAgentJobAssignment", mock.Anything, testAgent.ID, testJob.ID).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, queuedMessage.ID, models.ProcessedSlackMessageStatusInProgress).
			Return(nil)
		
		// Should send start conversation message
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Error_FailedToGetQueuedMessages", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return(nil, fmt.Errorf("database error"))
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get queued messages")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("ContinueOnJobNotFound", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
		
		queuedMessage1 := createTestProcessedMessage("invalid_job", models.ProcessedSlackMessageStatusQueued)
		queuedMessage2 := createTestProcessedMessage(testJob.ID, models.ProcessedSlackMessageStatusQueued)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage1, queuedMessage2}, nil)
		
		// First job not found
		mockJobsService.On("GetJobByID", mock.Anything, "invalid_job").
			Return(nil, fmt.Errorf("not found"))
		
		// Second job found and processed
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{testAgent}, nil)
		
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, testJob.ID).
			Return(testAgent, nil)
		
		mockJobsService.On("CreateAgentJobAssignment", mock.Anything, testAgent.ID, testJob.ID).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, queuedMessage2.ID, models.ProcessedSlackMessageStatusInProgress).
			Return(nil)
		
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute - should continue despite first job not found
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockAgentsUseCase.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("ContinueOnSlackIntegrationNotFound", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		
		queuedMessage := createTestProcessedMessage(testJob.ID, models.ProcessedSlackMessageStatusQueued)
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return([]*models.ProcessedSlackMessage{queuedMessage}, nil)
		
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		// Slack integration not found
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(nil, fmt.Errorf("not found"))
		
		// Execute - should continue despite integration not found
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
	})
	
	t.Run("Performance_LargeQueue", func(t *testing.T) {
		// Setup
		useCase, mockAgentsService, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, mockAgentsUseCase := setupSlackUseCase(t)
		
		// Create large queue
		var queuedMessages []*models.ProcessedSlackMessage
		for i := 0; i < 100; i++ {
			job := createTestJob(fmt.Sprintf("1234567890.%06d", i))
			job.ID = fmt.Sprintf("job_%d", i)
			msg := createTestProcessedMessage(job.ID, models.ProcessedSlackMessageStatusQueued)
			msg.ID = fmt.Sprintf("psm_%d", i)
			queuedMessages = append(queuedMessages, msg)
			
			// Mock job lookup
			mockJobsService.On("GetJobByID", mock.Anything, job.ID).
				Return(job, nil)
		}
		
		// Mock expectations
		mockJobsService.On("GetQueuedProcessedSlackMessages", mock.Anything).
			Return(queuedMessages, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationsByOrganizationID", mock.Anything, testSlackIntegration.OrganizationID).
			Return([]*models.SlackIntegration{testSlackIntegration}, nil)
		
		mockAgentsService.On("GetActiveAgentsBySlackIntegrationID", mock.Anything, testSlackIntegration.ID).
			Return([]*models.ActiveAgent{testAgent}, nil)
		
		mockAgentsUseCase.On("AssignAgentToJob", mock.Anything, testSlackIntegration.ID, mock.AnythingOfType("string")).
			Return(testAgent, nil)
		
		mockJobsService.On("CreateAgentJobAssignment", mock.Anything, testAgent.ID, mock.AnythingOfType("string")).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, mock.AnythingOfType("string"), models.ProcessedSlackMessageStatusInProgress).
			Return(nil)
		
		mockSocketClient.On("SendMessage", testAgent.WSConnectionID, mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Execute
		err := useCase.ProcessQueuedJobs(context.Background())
		
		// Assert - should complete without error
		require.NoError(t, err)
	})
}