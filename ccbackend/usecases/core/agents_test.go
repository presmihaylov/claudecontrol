package core

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/jobs"
	"ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
)

func TestRegisterAgent(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
			AgentID:        "agent-789",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("UpsertActiveAgent", ctx, "ws-123", "org-456", "agent-789").Return(agent, nil)

		// Execute
		err := useCase.RegisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
			AgentID:        "agent-789",
		}

		// Configure expectations
		mockAgentsService.On("UpsertActiveAgent", ctx, "ws-123", "org-456", "agent-789").
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.RegisterAgent(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register agent")
		assert.Contains(t, err.Error(), "database error")
		mockAgentsService.AssertExpectations(t)
	})
}

func TestDeregisterAgent(t *testing.T) {
	t.Run("success_no_jobs", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, "agent-789", "org-456").
			Return([]string{}, nil)
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("success_with_unknown_job_type", func(t *testing.T) {
		// Setup - test agent deregistration with job that has unknown type (should skip cleanup)
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		job := &models.Job{
			ID:             "job-111",
			JobType:        "unknown_type", // Unknown job type should be skipped during cleanup
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, "agent-789", "org-456").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil)
		// Unknown job type should be skipped in cleanup - no SlackUseCase call expected
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("agent_not_found", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.None[*models.ActiveAgent](), nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agent found for client")
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("job_not_found_during_cleanup", func(t *testing.T) {
		// Setup - test case where job is not found during cleanup
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, "agent-789", "org-456").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.None[*models.Job](), nil) // Job not found during cleanup
		// Job not found should be skipped - no error expected
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err) // Should succeed even if job not found
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("unknown_job_type", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		job := &models.Job{
			ID:             "job-111",
			JobType:        "unknown_type",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, "agent-789", "org-456").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.Some(job), nil)
		// Unknown job type should be skipped
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("job_not_found_skip", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, "agent-789", "org-456").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, "job-111", "org-456").
			Return(mo.None[*models.Job](), nil)
		// Job not found should be skipped
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})
}

func TestProcessPing(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("UpdateAgentLastActiveAt", ctx, "ws-123", "org-456").
			Return(nil)

		// Execute
		err := useCase.ProcessPing(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("agent_not_found", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.None[*models.ActiveAgent](), nil)

		// Execute
		err := useCase.ProcessPing(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agent found for client")
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("update_failure", func(t *testing.T) {
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

		client := &clients.Client{
			ID:             "ws-123",
			OrganizationID: "org-456",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, "ws-123", "org-456").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("UpdateAgentLastActiveAt", ctx, "ws-123", "org-456").
			Return(fmt.Errorf("database error"))

		// Execute
		err := useCase.ProcessPing(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to update agent last_active_at")
		mockAgentsService.AssertExpectations(t)
	})
}

func TestCleanupInactiveAgents(t *testing.T) {
	t.Run("no_integrations", func(t *testing.T) {
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
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{}, nil)

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.NoError(t, err)
		mockSlackIntegrationsService.AssertExpectations(t)
	})

	t.Run("multiple_inactive_agents", func(t *testing.T) {
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

		integration := &models.SlackIntegration{
			ID:             "slack-111",
			OrganizationID: "org-456",
		}

		now := time.Now()
		inactiveAgent1 := &models.ActiveAgent{
			ID:             "agent-001",
			OrganizationID: "org-456",
			LastActiveAt:   now.Add(-20 * time.Minute),
		}
		inactiveAgent2 := &models.ActiveAgent{
			ID:             "agent-002",
			OrganizationID: "org-456",
			LastActiveAt:   now.Add(-30 * time.Minute),
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockAgentsService.On("GetInactiveAgents", ctx, "org-456", DefaultInactiveAgentTimeoutMinutes).
			Return([]*models.ActiveAgent{inactiveAgent1, inactiveAgent2}, nil)
		mockAgentsService.On("DeleteActiveAgent", ctx, "agent-001", "org-456").
			Return(nil)
		mockAgentsService.On("DeleteActiveAgent", ctx, "agent-002", "org-456").
			Return(nil)

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.NoError(t, err)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("partial_cleanup_with_errors", func(t *testing.T) {
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

		integration := &models.SlackIntegration{
			ID:             "slack-111",
			OrganizationID: "org-456",
		}

		now := time.Now()
		inactiveAgent1 := &models.ActiveAgent{
			ID:             "agent-001",
			OrganizationID: "org-456",
			LastActiveAt:   now.Add(-20 * time.Minute),
		}
		inactiveAgent2 := &models.ActiveAgent{
			ID:             "agent-002",
			OrganizationID: "org-456",
			LastActiveAt:   now.Add(-30 * time.Minute),
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockAgentsService.On("GetInactiveAgents", ctx, "org-456", DefaultInactiveAgentTimeoutMinutes).
			Return([]*models.ActiveAgent{inactiveAgent1, inactiveAgent2}, nil)
		mockAgentsService.On("DeleteActiveAgent", ctx, "agent-001", "org-456").
			Return(fmt.Errorf("delete failed"))
		mockAgentsService.On("DeleteActiveAgent", ctx, "agent-002", "org-456").
			Return(nil)

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "inactive agent cleanup encountered")
		assert.Contains(t, err.Error(), "failed to delete inactive agent agent-001")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("get_inactive_agents_error", func(t *testing.T) {
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

		integration := &models.SlackIntegration{
			ID:             "slack-111",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockAgentsService.On("GetInactiveAgents", ctx, "org-456", DefaultInactiveAgentTimeoutMinutes).
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get inactive agents")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("get_integrations_error", func(t *testing.T) {
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
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get slack integrations")
		mockSlackIntegrationsService.AssertExpectations(t)
	})
}
