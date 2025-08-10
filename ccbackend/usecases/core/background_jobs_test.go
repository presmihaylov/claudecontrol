package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/jobs"
	"ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
)

func TestGetActiveOrganizations(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization1 := &models.Organization{ID: "org-1"}
		organization2 := &models.Organization{ID: "org-2"}
		organizations := []*models.Organization{organization1, organization2}

		connectedClientIDs := []string{"client-1", "client-2"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return(organizations, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{{ID: "agent-1", OrganizationID: "org-1"}}, nil)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-2", connectedClientIDs).
			Return([]*models.ActiveAgent{{ID: "agent-2", OrganizationID: "org-2"}}, nil)

		// Execute
		result, err := useCase.GetActiveOrganizations(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, "org-1", result[0].ID)
		assert.Equal(t, "org-2", result[1].ID)
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("no_available_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization := &models.Organization{ID: "org-1"}
		organizations := []*models.Organization{organization}

		connectedClientIDs := []string{"client-1"}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return(organizations, nil)
		mockWSClient.On("GetClientIDs").
			Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{}, nil) // No available agents

		// Execute
		result, err := useCase.GetActiveOrganizations(ctx)

		// Assert
		assert.NoError(t, err)
		assert.Len(t, result, 0) // Should be filtered out
		mockOrganizationsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("service_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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
		result, err := useCase.GetActiveOrganizations(ctx)

		// Assert
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "failed to get organizations")
		mockOrganizationsService.AssertExpectations(t)
	})
}

func TestAssignJobs(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization := &models.Organization{ID: "org-1"}
		agent := &models.ActiveAgent{ID: "agent-1", OrganizationID: "org-1"}
		job := &models.Job{ID: "job-1", OrganizationID: "org-1"}

		connectedClientIDs := []string{"client-1"}

		// Configure expectations
		mockWSClient.On("GetClientIDs").Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent}, nil)
		mockJobsService.On("GetIdleJobs", ctx, DefaultIdleJobTimeoutMinutes, "org-1").
			Return([]*models.Job{job}, nil)
		mockAgentsService.On("AssignAgentToJob", ctx, "agent-1", "job-1", "org-1").
			Return(nil)

		// Execute
		err := useCase.AssignJobs(ctx, organization)

		// Assert
		assert.NoError(t, err)
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("no_available_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization := &models.Organization{ID: "org-1"}
		connectedClientIDs := []string{"client-1"}

		// Configure expectations
		mockWSClient.On("GetClientIDs").Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{}, nil) // No available agents

		// Execute
		err := useCase.AssignJobs(ctx, organization)

		// Assert
		assert.NoError(t, err) // Should succeed even with no agents
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		// JobsService should not be called since there are no agents
	})

	t.Run("assignment_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization := &models.Organization{ID: "org-1"}
		agent := &models.ActiveAgent{ID: "agent-1", OrganizationID: "org-1"}
		job := &models.Job{ID: "job-1", OrganizationID: "org-1"}

		connectedClientIDs := []string{"client-1"}

		// Configure expectations
		mockWSClient.On("GetClientIDs").Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent}, nil)
		mockJobsService.On("GetIdleJobs", ctx, DefaultIdleJobTimeoutMinutes, "org-1").
			Return([]*models.Job{job}, nil)
		mockAgentsService.On("AssignAgentToJob", ctx, "agent-1", "job-1", "org-1").
			Return(fmt.Errorf("assignment failed"))

		// Execute
		err := useCase.AssignJobs(ctx, organization)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to assign agent")
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("get_idle_jobs_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agents.MockAgentsService)
		mockWSClient := new(socketio.MockSocketIOClient)
		mockJobsService := new(jobs.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrations.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizations.MockOrganizationsService)
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

		organization := &models.Organization{ID: "org-1"}
		agent := &models.ActiveAgent{ID: "agent-1", OrganizationID: "org-1"}

		connectedClientIDs := []string{"client-1"}

		// Configure expectations
		mockWSClient.On("GetClientIDs").Return(connectedClientIDs)
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-1", connectedClientIDs).
			Return([]*models.ActiveAgent{agent}, nil)
		mockJobsService.On("GetIdleJobs", ctx, DefaultIdleJobTimeoutMinutes, "org-1").
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.AssignJobs(ctx, organization)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get idle jobs")
		mockWSClient.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})
}