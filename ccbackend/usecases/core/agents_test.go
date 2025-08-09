package core

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"ccbackend/clients"
	"ccbackend/models"
)

// RegisterAgent Tests

func TestRegisterAgent_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
		AgentID:        "agent_789",
	}

	expectedAgent := &models.ActiveAgent{
		ID:               "aa_123",
		WSConnectionID:   client.ID,
		OrganizationID:   client.OrganizationID,
		CCAgentID:        client.AgentID,
		LastActiveAt:     time.Now(),
	}

	mockAgentsService.On("UpsertActiveAgent", ctx, client.ID, client.OrganizationID, client.AgentID).Return(expectedAgent, nil)

	// Act
	err := useCase.RegisterAgent(ctx, client)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "UpsertActiveAgent", 1)
	mockAgentsService.AssertCalled(t, "UpsertActiveAgent", ctx, client.ID, client.OrganizationID, client.AgentID)
}

func TestRegisterAgent_ServiceError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
		AgentID:        "agent_789",
	}

	serviceErr := errors.New("database error")
	mockAgentsService.On("UpsertActiveAgent", ctx, client.ID, client.OrganizationID, client.AgentID).Return((*models.ActiveAgent)(nil), serviceErr)

	// Act
	err := useCase.RegisterAgent(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to register agent")
	assert.Contains(t, err.Error(), client.ID)
	assert.Contains(t, err.Error(), "database error")
	mockAgentsService.AssertNumberOfCalls(t, "UpsertActiveAgent", 1)
}

func TestRegisterAgent_NilClient(t *testing.T) {
	// Setup
	ctx := context.Background()
	useCase := &CoreUseCase{}

	// Act & Assert - should panic when accessing nil client fields
	assert.Panics(t, func() {
		_ = useCase.RegisterAgent(ctx, nil)
	})
}

// DeregisterAgent Tests

func TestDeregisterAgent_Success_NoJobs(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	agent := &models.ActiveAgent{
		ID:             "aa_123",
		WSConnectionID: client.ID,
		OrganizationID: client.OrganizationID,
	}

	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.Some(agent), nil)
	mockAgentsService.On("GetActiveAgentJobAssignments", ctx, agent.ID, client.OrganizationID).Return([]string{}, nil)
	mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, client.ID, client.OrganizationID).Return(nil)

	// Act
	err := useCase.DeregisterAgent(ctx, client)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetAgentByWSConnectionID", 1)
	mockAgentsService.AssertNumberOfCalls(t, "GetActiveAgentJobAssignments", 1)
	mockAgentsService.AssertNumberOfCalls(t, "DeleteActiveAgentByWsConnectionID", 1)
}

func TestDeregisterAgent_Success_WithSlackJobs(t *testing.T) {
	// Setup
	t.Skip("Test requires slack usecase mocking - needs refactoring for interface-based testing")
}

func TestDeregisterAgent_Success_WithUnknownJobType(t *testing.T) {
	t.Skip("Test requires slack usecase mocking - needs refactoring for interface-based testing")
}

func TestDeregisterAgent_AgentNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.None[*models.ActiveAgent](), nil)

	// Act
	err := useCase.DeregisterAgent(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent found for client")
	assert.Contains(t, err.Error(), client.ID)
	mockAgentsService.AssertNotCalled(t, "GetActiveAgentJobAssignments", mock.Anything, mock.Anything, mock.Anything)
	mockAgentsService.AssertNotCalled(t, "DeleteActiveAgentByWsConnectionID", mock.Anything, mock.Anything, mock.Anything)
}

func TestDeregisterAgent_GetAgentError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	serviceErr := errors.New("database error")
	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.None[*models.ActiveAgent](), serviceErr)

	// Act
	err := useCase.DeregisterAgent(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get agent by WS connection ID")
	assert.Contains(t, err.Error(), "database error")
}

func TestDeregisterAgent_GetJobsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	agent := &models.ActiveAgent{
		ID:             "aa_123",
		WSConnectionID: client.ID,
		OrganizationID: client.OrganizationID,
	}

	jobsErr := errors.New("failed to fetch jobs")
	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.Some(agent), nil)
	mockAgentsService.On("GetActiveAgentJobAssignments", ctx, agent.ID, client.OrganizationID).Return([]string(nil), jobsErr)

	// Act
	err := useCase.DeregisterAgent(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get jobs for cleanup")
	assert.Contains(t, err.Error(), "failed to fetch jobs")
}

func TestDeregisterAgent_JobCleanupError(t *testing.T) {
	t.Skip("Test requires slack usecase mocking - needs refactoring for interface-based testing")
}

func TestDeregisterAgent_DeleteAgentError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	agent := &models.ActiveAgent{
		ID:             "aa_123",
		WSConnectionID: client.ID,
		OrganizationID: client.OrganizationID,
	}

	deleteErr := errors.New("delete failed")
	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.Some(agent), nil)
	mockAgentsService.On("GetActiveAgentJobAssignments", ctx, agent.ID, client.OrganizationID).Return([]string{}, nil)
	mockAgentsService.On("DeleteActiveAgentByWsConnectionID", ctx, client.ID, client.OrganizationID).Return(deleteErr)

	// Act
	err := useCase.DeregisterAgent(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to deregister agent")
	assert.Contains(t, err.Error(), client.ID)
	assert.Contains(t, err.Error(), "delete failed")
}

// ProcessPing Tests

func TestProcessPing_Success(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	agent := &models.ActiveAgent{
		ID:             "aa_123",
		WSConnectionID: client.ID,
		OrganizationID: client.OrganizationID,
		LastActiveAt:   time.Now(),
	}

	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.Some(agent), nil)
	mockAgentsService.On("UpdateAgentLastActiveAt", ctx, client.ID, client.OrganizationID).Return(nil)

	// Act
	err := useCase.ProcessPing(ctx, client)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetAgentByWSConnectionID", 1)
	mockAgentsService.AssertNumberOfCalls(t, "UpdateAgentLastActiveAt", 1)
	mockAgentsService.AssertCalled(t, "UpdateAgentLastActiveAt", ctx, client.ID, client.OrganizationID)
}

func TestProcessPing_AgentNotFound(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.None[*models.ActiveAgent](), nil)

	// Act
	err := useCase.ProcessPing(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no agent found for client")
	assert.Contains(t, err.Error(), client.ID)
	mockAgentsService.AssertNotCalled(t, "UpdateAgentLastActiveAt", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessPing_GetAgentError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	serviceErr := errors.New("database error")
	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.None[*models.ActiveAgent](), serviceErr)

	// Act
	err := useCase.ProcessPing(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get agent by WS connection ID")
	assert.Contains(t, err.Error(), "database error")
	mockAgentsService.AssertNotCalled(t, "UpdateAgentLastActiveAt", mock.Anything, mock.Anything, mock.Anything)
}

func TestProcessPing_UpdateTimestampError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		agentsService: mockAgentsService,
	}

	client := &clients.Client{
		ID:             "ws_conn_123",
		OrganizationID: "org_456",
	}

	agent := &models.ActiveAgent{
		ID:             "aa_123",
		WSConnectionID: client.ID,
		OrganizationID: client.OrganizationID,
	}

	updateErr := errors.New("update failed")
	mockAgentsService.On("GetAgentByWSConnectionID", ctx, client.ID, client.OrganizationID).Return(mo.Some(agent), nil)
	mockAgentsService.On("UpdateAgentLastActiveAt", ctx, client.ID, client.OrganizationID).Return(updateErr)

	// Act
	err := useCase.ProcessPing(ctx, client)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to update agent last_active_at")
	assert.Contains(t, err.Error(), "update failed")
}

// CleanupInactiveAgents Tests

func TestCleanupInactiveAgents_Success_NoIntegrations(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return([]*models.SlackIntegration{}, nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.NoError(t, err)
	mockSlackIntegrationsService.AssertNumberOfCalls(t, "GetAllSlackIntegrations", 1)
}

func TestCleanupInactiveAgents_Success_NoInactiveAgents(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "si_2",
			OrganizationID: "org_2",
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent{}, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent{}, nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetInactiveAgents", 2)
	mockAgentsService.AssertNotCalled(t, "DeleteActiveAgent", mock.Anything, mock.Anything, mock.Anything)
}

func TestCleanupInactiveAgents_Success_WithInactiveAgents(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "si_2",
			OrganizationID: "org_2",
		},
	}

	inactiveAgents1 := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-20 * time.Minute),
		},
		{
			ID:             "aa_2",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-15 * time.Minute),
		},
	}

	inactiveAgents2 := []*models.ActiveAgent{
		{
			ID:             "aa_3",
			OrganizationID: "org_2",
			LastActiveAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return(inactiveAgents1, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes).Return(inactiveAgents2, nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_1", "org_1").Return(nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_2", "org_1").Return(nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_3", "org_2").Return(nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetInactiveAgents", 2)
	mockAgentsService.AssertNumberOfCalls(t, "DeleteActiveAgent", 3)
}

func TestCleanupInactiveAgents_GetIntegrationsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
	}

	serviceErr := errors.New("database error")
	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return([]*models.SlackIntegration(nil), serviceErr)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get slack integrations")
	assert.Contains(t, err.Error(), "database error")
}

func TestCleanupInactiveAgents_GetInactiveAgentsError(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "si_2",
			OrganizationID: "org_2",
		},
	}

	inactiveAgents2 := []*models.ActiveAgent{
		{
			ID:             "aa_3",
			OrganizationID: "org_2",
			LastActiveAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent(nil), errors.New("org_1 error"))
	mockAgentsService.On("GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes).Return(inactiveAgents2, nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_3", "org_2").Return(nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive agent cleanup encountered 1 errors")
	assert.Contains(t, err.Error(), "failed to get inactive agents for integration si_1")
	assert.Contains(t, err.Error(), "org_1 error")
	// Should still process org_2
	mockAgentsService.AssertCalled(t, "DeleteActiveAgent", ctx, "aa_3", "org_2")
}

func TestCleanupInactiveAgents_PartialDeletionFailure(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
	}

	inactiveAgents := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-20 * time.Minute),
		},
		{
			ID:             "aa_2",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-15 * time.Minute),
		},
		{
			ID:             "aa_3",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return(inactiveAgents, nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_1", "org_1").Return(nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_2", "org_1").Return(errors.New("delete error aa_2"))
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_3", "org_1").Return(nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive agent cleanup encountered 1 errors")
	assert.Contains(t, err.Error(), "failed to delete inactive agent aa_2")
	// Should still delete other agents
	mockAgentsService.AssertCalled(t, "DeleteActiveAgent", ctx, "aa_1", "org_1")
	mockAgentsService.AssertCalled(t, "DeleteActiveAgent", ctx, "aa_3", "org_1")
}

func TestCleanupInactiveAgents_MultipleOrganizations(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "si_2",
			OrganizationID: "org_2",
		},
		{
			ID:             "si_3",
			OrganizationID: "org_3",
		},
	}

	org1Agents := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-20 * time.Minute),
		},
	}

	org3Agents := []*models.ActiveAgent{
		{
			ID:             "aa_3",
			OrganizationID: "org_3",
			LastActiveAt:   time.Now().Add(-30 * time.Minute),
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return(org1Agents, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent{}, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_3", DefaultInactiveAgentTimeoutMinutes).Return(org3Agents, nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_1", "org_1").Return(nil)
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_3", "org_3").Return(nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.NoError(t, err)
	mockAgentsService.AssertNumberOfCalls(t, "GetInactiveAgents", 3)
	mockAgentsService.AssertNumberOfCalls(t, "DeleteActiveAgent", 2)
	// Verify each organization is queried with its own ID
	mockAgentsService.AssertCalled(t, "GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes)
	mockAgentsService.AssertCalled(t, "GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes)
	mockAgentsService.AssertCalled(t, "GetInactiveAgents", ctx, "org_3", DefaultInactiveAgentTimeoutMinutes)
}

func TestCleanupInactiveAgents_TimeoutThreshold(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent{}, nil)

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	require.NoError(t, err)
	// Verify the correct timeout threshold is used
	assert.Equal(t, 10, DefaultInactiveAgentTimeoutMinutes)
	mockAgentsService.AssertCalled(t, "GetInactiveAgents", ctx, "org_1", 10)
}

func TestCleanupInactiveAgents_JobNotFound(t *testing.T) {
	t.Skip("Test requires slack usecase mocking - needs refactoring for interface-based testing")
}

func TestCleanupInactiveAgents_MultipleErrors(t *testing.T) {
	// Setup
	ctx := context.Background()
	mockSlackIntegrationsService := new(MockSlackIntegrationsService)
	mockAgentsService := new(MockAgentsService)
	useCase := &CoreUseCase{
		slackIntegrationsService: mockSlackIntegrationsService,
		agentsService:            mockAgentsService,
	}

	integrations := []*models.SlackIntegration{
		{
			ID:             "si_1",
			OrganizationID: "org_1",
		},
		{
			ID:             "si_2",
			OrganizationID: "org_2",
		},
	}

	inactiveAgents1 := []*models.ActiveAgent{
		{
			ID:             "aa_1",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-20 * time.Minute),
		},
		{
			ID:             "aa_2",
			OrganizationID: "org_1",
			LastActiveAt:   time.Now().Add(-15 * time.Minute),
		},
	}

	mockSlackIntegrationsService.On("GetAllSlackIntegrations", ctx).Return(integrations, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_1", DefaultInactiveAgentTimeoutMinutes).Return(inactiveAgents1, nil)
	mockAgentsService.On("GetInactiveAgents", ctx, "org_2", DefaultInactiveAgentTimeoutMinutes).Return([]*models.ActiveAgent(nil), fmt.Errorf("org_2 error"))
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_1", "org_1").Return(fmt.Errorf("delete aa_1 error"))
	mockAgentsService.On("DeleteActiveAgent", ctx, "aa_2", "org_1").Return(fmt.Errorf("delete aa_2 error"))

	// Act
	err := useCase.CleanupInactiveAgents(ctx)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "inactive agent cleanup encountered 3 errors")
	assert.Contains(t, err.Error(), "failed to get inactive agents for integration si_2: org_2 error")
	assert.Contains(t, err.Error(), "failed to delete inactive agent aa_1: delete aa_1 error")
	assert.Contains(t, err.Error(), "failed to delete inactive agent aa_2: delete aa_2 error")
}