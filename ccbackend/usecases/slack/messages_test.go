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

func TestProcessAssistantMessage(t *testing.T) {
	t.Run("Success_SendToSlack", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              testJob.ID,
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "Here's the solution to your problem",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		mockSlackClient.On("RemoveReaction", "hourglass_flowing_sand", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", "white_check_mark", mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("Success_UpdateReactions", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              testJob.ID,
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "Solution provided",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackClient.On("PostMessage", mock.AnythingOfType("string"), mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.NoError(t, err)
		
		// Verify reaction updates were called
		mockSlackClient.AssertCalled(t, "RemoveReaction", "hourglass_flowing_sand", mock.AnythingOfType("slack.ItemRef"))
		mockSlackClient.AssertCalled(t, "AddReaction", "white_check_mark", mock.AnythingOfType("slack.ItemRef"))
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("Error_InvalidJobID", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              "invalid_job",
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "Test message",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, "invalid_job").
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job not found")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("Error_MissingSlackIntegration", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              testJob.ID,
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "Test message",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "slack integration not found")
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
	})
	
	t.Run("Error_SlackAPIFailure", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              testJob.ID,
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "Test message",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackClient.On("PostMessage", mock.AnythingOfType("string"), mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "", fmt.Errorf("rate_limited"))
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message to Slack")
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("EmptyMessage", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		assistantPayload := &models.AssistantMessagePayload{
			JobID:              testJob.ID,
			ProcessedMessageID: testProcessedMessage.ID,
			Message:            "",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockJobsService.On("GetProcessedSlackMessageByID", mock.Anything, testProcessedMessage.ID).
			Return(testProcessedMessage, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Should still send empty message
		mockSlackClient.On("PostMessage", mock.AnythingOfType("string"), mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		mockSlackClient.On("RemoveReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockSlackClient.On("AddReaction", mock.AnythingOfType("string"), mock.AnythingOfType("slack.ItemRef")).
			Return(nil)
		
		mockJobsService.On("UpdateProcessedSlackMessageStatus", mock.Anything, testProcessedMessage.ID, models.ProcessedSlackMessageStatusCompleted).
			Return(nil)
		
		// Execute
		err := useCase.ProcessAssistantMessage(context.Background(), assistantPayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
}

func TestProcessSystemMessage(t *testing.T) {
	t.Run("Success_NormalSystemMessage", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		systemPayload := &models.SystemMessagePayload{
			JobID:   testJob.ID,
			Message: "Processing your request...",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		// Execute
		err := useCase.ProcessSystemMessage(context.Background(), systemPayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
	
	t.Run("Success_ErrorMessage_Cleanup", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, mockSocketClient, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		systemPayload := &models.SystemMessagePayload{
			JobID:   testJob.ID,
			Message: "ccagent encountered error: Failed to process request",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Error message should trigger cleanup
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
			Return([]*models.AgentJobAssignment{{AgentID: testAgent.ID, JobID: testJob.ID}}, nil)
		
		mockSocketClient.On("SendMessage", mock.AnythingOfType("string"), mock.AnythingOfType("map[string]interface {}")).
			Return(nil)
		
		mockJobsService.On("DeleteAgentJobAssignmentsByJobID", mock.Anything, testJob.ID).
			Return(nil)
		
		// Execute
		err := useCase.ProcessSystemMessage(context.Background(), systemPayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
		mockSocketClient.AssertExpectations(t)
	})
	
	t.Run("Error_InvalidJobID", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, _, _, _, _ := setupSlackUseCase(t)
		
		systemPayload := &models.SystemMessagePayload{
			JobID:   "invalid_job",
			Message: "Test message",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, "invalid_job").
			Return(nil, fmt.Errorf("not found"))
		
		// Execute
		err := useCase.ProcessSystemMessage(context.Background(), systemPayload)
		
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
		
		systemPayload := &models.SystemMessagePayload{
			JobID:   jobWithoutPayload.ID,
			Message: "Test message",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, jobWithoutPayload.ID).
			Return(jobWithoutPayload, nil)
		
		// Execute
		err := useCase.ProcessSystemMessage(context.Background(), systemPayload)
		
		// Assert
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job has no slack payload")
		
		mockJobsService.AssertExpectations(t)
	})
	
	t.Run("EmptySystemMessage", func(t *testing.T) {
		// Setup
		useCase, _, mockJobsService, mockSlackIntegrationsService, _, _, _ := setupSlackUseCase(t)
		useCase.slackClients = make(map[string]SlackClientInterface)
		mockSlackClient := new(MockSlackClient)
		useCase.slackClients[testSlackIntegration.ID] = mockSlackClient
		
		systemPayload := &models.SystemMessagePayload{
			JobID:   testJob.ID,
			Message: "",
		}
		
		// Mock expectations
		mockJobsService.On("GetJobByID", mock.Anything, testJob.ID).
			Return(testJob, nil)
		
		mockSlackIntegrationsService.On("GetSlackIntegrationByID", mock.Anything, testSlackIntegration.ID).
			Return(testSlackIntegration, nil)
		
		// Should still send empty message
		mockSlackClient.On("PostMessage", testJob.SlackPayload.ChannelID, mock.AnythingOfType("slack.MsgOption"), mock.AnythingOfType("slack.MsgOption")).
			Return("", "1234567890.123457", nil)
		
		// Execute
		err := useCase.ProcessSystemMessage(context.Background(), systemPayload)
		
		// Assert
		require.NoError(t, err)
		
		mockJobsService.AssertExpectations(t)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockSlackClient.AssertExpectations(t)
	})
}

func TestIsAgentErrorMessage(t *testing.T) {
	tests := []struct {
		name     string
		message  string
		expected bool
	}{
		{
			name:     "StandardErrorMessage",
			message:  "ccagent encountered error: Failed to process",
			expected: true,
		},
		{
			name:     "ErrorMessageWithDetails",
			message:  "ccagent encountered error: Connection timeout after 30 seconds",
			expected: true,
		},
		{
			name:     "NormalMessage",
			message:  "Processing your request",
			expected: false,
		},
		{
			name:     "PartialMatch",
			message:  "The ccagent is working",
			expected: false,
		},
		{
			name:     "EmptyMessage",
			message:  "",
			expected: false,
		},
		{
			name:     "CaseSensitive",
			message:  "CCAGENT ENCOUNTERED ERROR: test",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useCase, _, _, _, _, _, _ := setupSlackUseCase(t)
			result := useCase.IsAgentErrorMessage(tt.message)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Mock Slack Client for testing
type MockSlackClient struct {
	mock.Mock
}

func (m *MockSlackClient) PostMessage(channelID string, options ...any) (string, string, error) {
	args := m.Called(channelID, options[0], options[1])
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockSlackClient) AddReaction(name string, item any) error {
	args := m.Called(name, item)
	return args.Error(0)
}

func (m *MockSlackClient) RemoveReaction(name string, item any) error {
	args := m.Called(name, item)
	return args.Error(0)
}

func (m *MockSlackClient) GetReactions(item any, params any) ([]any, error) {
	args := m.Called(item, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]any), args.Error(1)
}

func (m *MockSlackClient) GetPermalink(params any) (string, error) {
	args := m.Called(params)
	return args.String(0), args.Error(1)
}

func (m *MockSlackClient) GetUsersInfo(users ...string) (*any, error) {
	args := m.Called(users)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	userInfo := args.Get(0).(any)
	return &userInfo, args.Error(1)
}