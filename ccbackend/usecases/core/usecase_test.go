package core

import (
	"context"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"ccbackend/clients"
	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
	"ccbackend/services/jobs"
	"ccbackend/services/organizations"
	slackintegrations "ccbackend/services/slack_integrations"
)

// Agent Management Tests

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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			AgentID: "agent-789",
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("UpsertActiveAgent", ctx, models.OrgID("org-456"), "ws-123", "agent-789", "github.com/test/repo").
			Return(agent, nil)

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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			AgentID: "agent-789",
			RepoURL: "github.com/test/repo",
		}

		// Configure expectations
		mockAgentsService.On("UpsertActiveAgent", ctx, models.OrgID("org-456"), "ws-123", "agent-789", "github.com/test/repo").
			Return(nil, assert.AnError)

		// Execute
		err := useCase.RegisterAgent(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to register agent")
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, models.OrgID("org-456"), "agent-789").
			Return([]string{}, nil)
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, models.OrgID("org-456"), "ws-123").
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		job := &models.Job{
			ID:      "job-111",
			JobType: "unknown_type", // Unknown job type should be skipped during cleanup
			OrgID:   models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, models.OrgID("org-456"), "agent-789").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, models.OrgID("org-456"), "job-111").
			Return(mo.Some(job), nil)
		// Unknown job type should be skipped in cleanup - no SlackUseCase call expected
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert
		assert.NoError(t, err)
		mockAgentsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("agent_not_found", func(t *testing.T) {
		// Setup - test agent deregistration when agent is not found (should succeed gracefully)
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.None[*models.ActiveAgent](), nil)

		// Execute
		err := useCase.DeregisterAgent(ctx, client)

		// Assert - should succeed when agent not found (handles reconnection race condition)
		assert.NoError(t, err)
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, models.OrgID("org-456"), "agent-789").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, models.OrgID("org-456"), "job-111").
			Return(mo.None[*models.Job](), nil) // Job not found during cleanup
		// Job not found should be skipped - no error expected
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, models.OrgID("org-456"), "ws-123").
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		job := &models.Job{
			ID:      "job-111",
			JobType: "unknown_type",
			OrgID:   models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, models.OrgID("org-456"), "agent-789").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, models.OrgID("org-456"), "job-111").
			Return(mo.Some(job), nil)
		// Unknown job type should be skipped
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, models.OrgID("org-456"), "ws-123").
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("GetActiveAgentJobAssignments", ctx, models.OrgID("org-456"), "agent-789").
			Return([]string{"job-111"}, nil)
		mockJobsService.On("GetJobByID", ctx, models.OrgID("org-456"), "job-111").
			Return(mo.None[*models.Job](), nil)
		// Job not found should be skipped
		mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, models.OrgID("org-456"), "ws-123").
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		agent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.Some(agent), nil)
		mockAgentsService.On("UpdateAgentLastActiveAt", ctx, models.OrgID("org-456"), "ws-123").
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		client := &clients.Client{
			ID:      "ws-123",
			OrgID:   models.OrgID("org-456"),
			RepoURL: "github.com/test/repo",
		}

		// Configure expectations
		mockAgentsService.On("GetAgentByWSConnectionID", ctx, models.OrgID("org-456"), "ws-123").
			Return(mo.None[*models.ActiveAgent](), nil)

		// Execute
		err := useCase.ProcessPing(ctx, client)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agent found for client")
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{}, nil)

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		organization := &models.Organization{
			ID: "org-456",
		}

		now := time.Now()
		inactiveAgent1 := &models.ActiveAgent{
			ID:           "agent-001",
			OrgID:        models.OrgID("org-456"),
			LastActiveAt: now.Add(-20 * time.Minute),
		}
		inactiveAgent2 := &models.ActiveAgent{
			ID:           "agent-002",
			OrgID:        models.OrgID("org-456"),
			LastActiveAt: now.Add(-30 * time.Minute),
		}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{organization}, nil)
		mockAgentsService.On("GetInactiveAgents", ctx, models.OrgID("org-456"), DefaultInactiveAgentTimeoutMinutes).
			Return([]*models.ActiveAgent{inactiveAgent1, inactiveAgent2}, nil)
		mockAgentsService.On("DeleteActiveAgent", ctx, models.OrgID("org-456"), "agent-001").
			Return(nil)
		mockAgentsService.On("DeleteActiveAgent", ctx, models.OrgID("org-456"), "agent-002").
			Return(nil)

		// Execute
		err := useCase.CleanupInactiveAgents(ctx)

		// Assert
		assert.NoError(t, err)
		mockOrganizationsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})
}

// Background Processing Tests
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		organization := &models.Organization{
			ID: "org-456",
		}

		agent1 := &models.ActiveAgent{
			ID:             "agent-001",
			WSConnectionID: "ws-001",
			OrgID:          models.OrgID("org-456"),
		}

		agent2 := &models.ActiveAgent{
			ID:             "agent-002",
			WSConnectionID: "ws-002",
			OrgID:          models.OrgID("org-456"),
		}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{organization}, nil)
		mockWSClient.On("GetClientIDs").
			Return([]string{"ws-001", "ws-002"})
		mockAgentsService.On("GetConnectedActiveAgents", ctx, models.OrgID("org-456"), []string{"ws-001", "ws-002"}).
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
			nil, // slackUseCase
			nil, // discordUseCase
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
			nil, // slackUseCase
			nil, // discordUseCase
		)

		organization := &models.Organization{
			ID: "org-456",
		}

		// Configure expectations
		mockOrganizationsService.On("GetAllOrganizations", ctx).
			Return([]*models.Organization{organization}, nil)
		mockWSClient.On("GetClientIDs").
			Return([]string{"ws-001", "ws-002"})
		mockAgentsService.On("GetConnectedActiveAgents", ctx, models.OrgID("org-456"), []string{"ws-001", "ws-002"}).
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
