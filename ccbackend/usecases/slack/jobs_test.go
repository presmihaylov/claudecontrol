package slack

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/models"
)

func TestProcessJobComplete(t *testing.T) {
	t.Run("Success_UpdateReactionAndNotifyAgents", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   testJob.ID,
			Message: "Job completed successfully",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Update reaction to hand
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		// Update processed message status
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// Get agent assignments for notification
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{
				{AgentID: testAgent.ID, JobID: testJob.ID},
				{AgentID: "aa_another", JobID: testJob.ID},
			}, nil)
		
		// Notify other agents
		mockSocketClient.On("SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
			Return(nil).Times(2)
		
		// Delete agent assignments
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Success_NoOtherAgentsToNotify", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   testJob.ID,
			Message: "Job completed",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// No agent assignments
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{}, nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("Error_JobNotFound", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   "invalid_job",
			Message: "Job completed",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, "invalid_job").
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job not found")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_MissingJobPayload", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		jobWithoutPayload := &models.Job{
			ID:           "job_test",
			SlackPayload: nil,
		}
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   jobWithoutPayload.ID,
			Message: "Job completed",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, jobWithoutPayload.ID).
			Return(jobWithoutPayload, nil)
		
		// Execute
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job has no slack payload")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_SlackIntegrationNotFound", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   testJob.ID,
			Message: "Job completed",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slack integration not found")
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
	})
	
	t.Run("ContinueOnReactionError", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		jobCompletePayload := &models.JobCompletePayload{
			JobID:   testJob.ID,
			Message: "Job completed",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Reaction operations fail but should continue
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(fmt.Errorf("reaction not found"))
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(fmt.Errorf("already_reacted"))
		
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{}, nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute - should not error despite reaction failures
		err := useCase.ProcessJobComplete(context.Background(), jobCompletePayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
}

func TestCleanupFailedSlackJob(t *testing.T) {
	t.Run("Success_SendErrorNotification", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		errorMessage := "Failed to process request: timeout"
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Send error message to Slack
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		// Update reaction to hand
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		// Update processed message status
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// Get and notify agents
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{{AgentID: testAgent.ID, JobID: testJob.ID}}, nil)
		
		mockSocketClient.On("SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		// Delete agent assignments
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute
		err := useCase.CleanupFailedSlackJob(context.Background(), testJob.ID, errorMessage)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Success_EmptyErrorMessage", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Should send default error message
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{}, nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute with empty error message
		err := useCase.CleanupFailedSlackJob(context.Background(), testJob.ID, "")
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("Error_JobNotFound", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, "invalid_job").
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.CleanupFailedSlackJob(context.Background(), "invalid_job", "error")
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job not found")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_MissingJobPayload", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		jobWithoutPayload := &models.Job{
			ID:           "job_test",
			SlackPayload: nil,
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, jobWithoutPayload.ID).
			Return(jobWithoutPayload, nil)
		
		// Execute
		err := useCase.CleanupFailedSlackJob(context.Background(), jobWithoutPayload.ID, "error")
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job has no slack payload")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_SlackIntegrationNotFound", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.CleanupFailedSlackJob(context.Background(), testJob.ID, "error")
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slack integration not found")
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
	})
	
	t.Run("ContinueOnSlackAPIError", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Slack API call fails
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "", fmt.Errorf("rate_limited"))
		
		// Should still try to update reactions and clean up
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "hand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("GetProcessedSlackMessage", mock.Anything, testJob.SlackPayload.ThreadTS, testJob.SlackPayload.ChannelID, testSlackIntegration.ID).
			Return(testProcessedMessage, nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		mockJobsService.On("GetAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return([]*models.AgentJobAssignment{}, nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute - should continue despite Slack API error
		err := useCase.CleanupFailedSlackJob(context.Background(), testJob.ID, "error")
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
}