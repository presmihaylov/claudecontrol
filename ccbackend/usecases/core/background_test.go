package core

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/jobs"
	"ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
)

func TestBroadcastCheckIdleJobs(t *testing.T) {
	t.Run("success_single_organization", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)

		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		organization := &models.Organization{
			ID: "org-456",
		}

		agent1 := &models.ActiveAgent{
			ID:             "agent-001",
			WSConnectionID: "ws-001",
			OrganizationID: "org-456",
		}

		agent2 := &models.ActiveAgent{
			ID:             "agent-002",
			WSConnectionID: "ws-002",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{organization}, nil)
		mockWSClient.On("GetClientIDs").
			Return([]string{"ws-001", "ws-002"})
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-456", []string{"ws-001", "ws-002"}).
			Return([]*models.ActiveAgent{agent1, agent2}, nil)

		// Expect SendMessage to be called for each agent
		mockWSClient.On("SendMessage", "ws-001", mock.MatchedBy(func(msg any) bool {
			// Verify it's a CheckIdleJobs message
			baseMsg, ok := msg.(models.BaseMessage)
			return ok && baseMsg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)

		mockWSClient.On("SendMessage", "ws-002", mock.MatchedBy(func(msg any) bool {
			// Verify it's a CheckIdleJobs message
			baseMsg, ok := msg.(models.BaseMessage)
			return ok && baseMsg.Type == models.MessageTypeCheckIdleJobs
		})).Return(nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("no_organizations", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)

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

	t.Run("no_connected_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)

		useCase := NewCoreUseCase(
			mockWSClient,
			mockAgentsService,
			mockJobsService,
			mockSlackIntegrationsService,
			mockOrganizationsService,
			nil, // agentsUseCase
			nil, // slackUseCase
		)

		organization := &models.Organization{
			ID: "org-456",
		}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{organization}, nil)
		mockWSClient.On("GetClientIDs").
			Return([]string{"ws-001", "ws-002"})
		mockAgentsService.On("GetConnectedActiveAgents", ctx, "org-456", []string{"ws-001", "ws-002"}).
			Return([]*models.ActiveAgent{}, nil)

		// Execute
		err := useCase.BroadcastCheckIdleJobs(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})
}
