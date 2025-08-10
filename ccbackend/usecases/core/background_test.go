package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"

	"ccbackend/clients"
	"ccbackend/models"
	agentsmocks "ccbackend/services/agents"
	jobsmocks "ccbackend/services/jobs"
	organizationsmocks "ccbackend/services/organizations"
	slackintegrationsmocks "ccbackend/services/slack_integrations"
)

func TestProcessJobsInBackground(t *testing.T) {
	t.Run("no_integrations", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.NoError(t, err)
		mockSlackIntegrationsService.AssertExpectations(t)
	})

	t.Run("success_with_jobs", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		pendingJob := &models.Job{
			ID:             "job-123",
			JobType:        models.JobTypeSlack,
			OrganizationID: "org-456",
		}

		availableAgent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return([]*models.Job{pendingJob}, nil)
		mockWSClient.On("GetClientIDs").Return([]string{"ws-123"})
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-456", []string{"ws-123"}).
			Return([]*models.ActiveAgent{availableAgent}, nil)
		mockJobsService.On("AssignJobToAgent", ctx, "job-123", "agent-789", "org-456").
			Return(nil)
		mockWSClient.On("SendMessage", "ws-123", pendingJob).Return(nil)

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.NoError(t, err)
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
	})

	t.Run("no_available_agents", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		pendingJob := &models.Job{
			ID:             "job-123",
			JobType:        models.JobTypeSlack,
			OrganizationID: "org-456",
		}

		// Configure expectations - no agents available
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return([]*models.Job{pendingJob}, nil)
		mockWSClient.On("GetClientIDs").Return([]string{"ws-123"})
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-456", []string{"ws-123"}).
			Return([]*models.ActiveAgent{}, nil)
		// No job assignment should happen since no agents available

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.NoError(t, err) // Should succeed even with no agents
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("job_assignment_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		pendingJob := &models.Job{
			ID:             "job-123",
			JobType:        models.JobTypeSlack,
			OrganizationID: "org-456",
		}

		availableAgent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return([]*models.Job{pendingJob}, nil)
		mockWSClient.On("GetClientIDs").Return([]string{"ws-123"})
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-456", []string{"ws-123"}).
			Return([]*models.ActiveAgent{availableAgent}, nil)
		mockJobsService.On("AssignJobToAgent", ctx, "job-123", "agent-789", "org-456").
			Return(fmt.Errorf("assignment failed"))

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to assign job")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("message_send_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		pendingJob := &models.Job{
			ID:             "job-123",
			JobType:        models.JobTypeSlack,
			OrganizationID: "org-456",
		}

		availableAgent := &models.ActiveAgent{
			ID:             "agent-789",
			WSConnectionID: "ws-123",
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return([]*models.Job{pendingJob}, nil)
		mockWSClient.On("GetClientIDs").Return([]string{"ws-123"})
		mockAgentsService.On("GetConnectedAvailableAgents", ctx, "org-456", []string{"ws-123"}).
			Return([]*models.ActiveAgent{availableAgent}, nil)
		mockJobsService.On("AssignJobToAgent", ctx, "job-123", "agent-789", "org-456").
			Return(nil)
		mockWSClient.On("SendMessage", "ws-123", pendingJob).Return(fmt.Errorf("send failed"))

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send job message")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
		mockWSClient.AssertExpectations(t)
	})

	t.Run("get_jobs_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get jobs")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
	})

	t.Run("get_agents_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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

		pendingJob := &models.Job{
			ID:             "job-123",
			JobType:        models.JobTypeSlack,
			OrganizationID: "org-456",
		}

		// Configure expectations
		mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).
			Return([]*models.SlackIntegration{integration}, nil)
		mockJobsService.On("GetJobsByOrganizationID", ctx, "org-456").
			Return([]*models.Job{pendingJob}, nil)
		mockAgentsService.On("GetActiveAgentsByOrganizationID", ctx, "org-456").
			Return(nil, fmt.Errorf("database error"))

		// Execute
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get active agents")
		mockSlackIntegrationsService.AssertExpectations(t)
		mockJobsService.AssertExpectations(t)
		mockAgentsService.AssertExpectations(t)
	})

	t.Run("get_integrations_error", func(t *testing.T) {
		// Setup
		ctx := context.Background()
		mockAgentsService := new(agentsmocks.MockAgentsService)
		mockWSClient := new(clients.MockSocketIOClient)
		mockJobsService := new(jobsmocks.MockJobsService)
		mockSlackIntegrationsService := new(slackintegrationsmocks.MockSlackIntegrationsService)
		mockOrganizationsService := new(organizationsmocks.MockOrganizationsService)
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
		err := useCase.ProcessJobsInBackground(ctx)

		// Assert
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get slack integrations")
		mockSlackIntegrationsService.AssertExpectations(t)
	})
}