package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/models"
)

func TestBroadcastCheckIdleJobs(t *testing.T) {
	t.Run("no_organizations", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{}, nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
	})

	t.Run("multiple_connected_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		org1 := &models.Organization{
			ID: "org-1",
		}
		org2 := &models.Organization{
			ID: "org-2",
		}

		agent1 := &models.ActiveAgent{
			ID:             "agent-1",
			WSConnectionID: "ws-1",
			OrganizationID: "org-1",
		}
		agent2 := &models.ActiveAgent{
			ID:             "agent-2",
			WSConnectionID: "ws-2",
			OrganizationID: "org-1",
		}
		agent3 := &models.ActiveAgent{
			ID:             "agent-3",
			WSConnectionID: "ws-3",
			OrganizationID: "org-2",
		}

		connectedClientIDs := []string{"ws-1", "ws-2", "ws-3", "ws-4"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{org1, org2}, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent1, agent2}, nil)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-2", connectedClientIDs).
			Return([]*models.ActiveAgent{agent3}, nil)

		// We need to use a custom matcher for the message since it contains a generated ID
		mockWSClient.On("SendMessage", "ws-1", mock.MatchedBy(func(msg models.BaseMessage) bool {
			return msg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)
		mockWSClient.On("SendMessage", "ws-2", mock.MatchedBy(func(msg models.BaseMessage) bool {
			return msg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)
		mockWSClient.On("SendMessage", "ws-3", mock.MatchedBy(func(msg models.BaseMessage) bool {
			return msg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("websocket_send_failure", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		org := &models.Organization{
			ID: "org-1",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-1",
			WSConnectionID: "ws-1",
			OrganizationID: "org-1",
		}

		connectedClientIDs := []string{"ws-1"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{org}, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent}, nil)
		mockWSClient.On("SendMessage", "ws-1", mock.MatchedBy(func(msg models.BaseMessage) bool {
			return msg.Type == models.MessageTypeCheckIdleJobs
		})).Return(fmt.Errorf("websocket connection closed"))

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send CheckIdleJobs message")
		assert.Contains(t, err.Error(), "websocket connection closed")
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("get_organizations_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get organizations")
		mockOrganizationsService.AssertExpectations(t)
	})

	t.Run("get_connected_agents_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		org := &models.Organization{
			ID: "org-1",
		}

		connectedClientIDs := []string{"ws-1", "ws-2"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{org}, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-1", connectedClientIDs).
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get connected agents")
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("no_connected_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		org := &models.Organization{
			ID: "org-1",
		}

		connectedClientIDs := []string{}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{org}, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{}, nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("mixed_success_and_no_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(MockAgentsService)
		mockWSClient := new(MockSocketIOClient)
		mockJobsService := new(MockJobsService)
		mockSlackIntegrationsService := new(MockSlackIntegrationsService)
		mockOrganizationsService := new(MockOrganizationsService)
		// Pass nil for use cases that aren't used in this test
		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		org1 := &models.Organization{
			ID: "org-1",
		}
		org2 := &models.Organization{
			ID: "org-2",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-1",
			WSConnectionID: "ws-1",
			OrganizationID: "org-1",
		}

		connectedClientIDs := []string{"ws-1"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{org1, org2}, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent}, nil)
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-2", connectedClientIDs).
			Return([]*models.ActiveAgent{}, nil) // No agents for org-2
		mockWSClient.On("SendMessage", "ws-1", mock.MatchedBy(func(msg models.BaseMessage) bool {
			return msg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})
}