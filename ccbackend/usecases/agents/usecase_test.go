package agents

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/mo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients/socketio"
	"ccbackend/models"
	"ccbackend/services/agents"
)

// Test helper functions
func createTestAgent(id, wsConnectionID, organizationID string) *models.ActiveAgent {
	return &models.ActiveAgent{
		ID:             id,
		WSConnectionID: wsConnectionID,
		OrganizationID: organizationID,
		CCAgentID:      "ccaid_test123",
	}
}

func createTestJob(id, slackThreadTS, slackChannelID, organizationID string) *models.Job {
	return &models.Job{
		ID:             id,
		JobType:        models.JobTypeSlack,
		OrganizationID: organizationID,
		SlackPayload: &models.SlackJobPayload{
			ThreadTS:  slackThreadTS,
			ChannelID: slackChannelID,
		},
	}
}

// Constructor Tests
func TestNewAgentsUseCase(t *testing.T) {
	t.Run("Valid initialization", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		useCase := NewAgentsUseCase(mockWS, mockAgents)

		assert.NotNil(t, useCase)
		assert.Equal(t, mockWS, useCase.wsClient)
		assert.Equal(t, mockAgents, useCase.agentsService)
	})
}

// GetOrAssignAgentForJob Tests
func TestGetOrAssignAgentForJob(t *testing.T) {
	ctx := context.Background()
	organizationID := "org_test123"
	threadTS := "1234567890.123456"
	job := createTestJob("job_123", threadTS, "C123456", organizationID)

	t.Run("Job already assigned with active connection", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.Some(agent), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_123", "ws_conn_456"})
		mockAgents.On("CheckAgentHasActiveConnection", agent, []string{"ws_conn_123", "ws_conn_456"}).
			Return(true)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, err := useCase.GetOrAssignAgentForJob(ctx, job, threadTS, organizationID)

		assert.NoError(t, err)
		assert.Equal(t, "ws_conn_123", clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("Job assigned but agent disconnected", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.Some(agent), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_456"}) // Different connection ID
		mockAgents.On("CheckAgentHasActiveConnection", agent, []string{"ws_conn_456"}).
			Return(false)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, err := useCase.GetOrAssignAgentForJob(ctx, job, threadTS, organizationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no active agents available")
		assert.Empty(t, clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("New job needing assignment", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations for no existing assignment
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil).Once()

		// Setup expectations for assignment flow
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil).Once()
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_123"})
		mockAgents.On("GetConnectedActiveAgents", ctx, organizationID, []string{"ws_conn_123"}).
			Return([]*models.ActiveAgent{agent}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent.ID, organizationID).
			Return([]string{}, nil)
		mockAgents.On("AssignAgentToJob", ctx, agent.ID, job.ID, organizationID).
			Return(nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, err := useCase.GetOrAssignAgentForJob(ctx, job, threadTS, organizationID)

		assert.NoError(t, err)
		assert.Equal(t, "ws_conn_123", clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})
}

// AssignJobToAvailableAgent Tests
func TestAssignJobToAvailableAgent(t *testing.T) {
	ctx := context.Background()
	organizationID := "org_test123"
	threadTS := "1234567890.123456"
	job := createTestJob("job_123", threadTS, "C123456", organizationID)

	t.Run("Successful assignment", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_123"})
		mockAgents.On("GetConnectedActiveAgents", ctx, organizationID, []string{"ws_conn_123"}).
			Return([]*models.ActiveAgent{agent}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent.ID, organizationID).
			Return([]string{}, nil)
		mockAgents.On("AssignAgentToJob", ctx, agent.ID, job.ID, organizationID).
			Return(nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, err := useCase.AssignJobToAvailableAgent(ctx, job, threadTS, organizationID)

		assert.NoError(t, err)
		assert.Equal(t, "ws_conn_123", clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("No agents available", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, job.ID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil)
		mockWS.On("GetClientIDs").Return([]string{})
		mockAgents.On("GetConnectedActiveAgents", ctx, organizationID, []string{}).
			Return([]*models.ActiveAgent{}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, err := useCase.AssignJobToAvailableAgent(ctx, job, threadTS, organizationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no agents with active WebSocket connections")
		assert.Empty(t, clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})
}

// TryAssignJobToAgent Tests
func TestTryAssignJobToAgent(t *testing.T) {
	ctx := context.Background()
	organizationID := "org_test123"
	jobID := "job_123"

	t.Run("Job already assigned with active connection", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, jobID, organizationID).
			Return(mo.Some(agent), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_123"})
		mockAgents.On("CheckAgentHasActiveConnection", agent, []string{"ws_conn_123"}).
			Return(true)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, wasAssigned, err := useCase.TryAssignJobToAgent(ctx, jobID, organizationID)

		assert.NoError(t, err)
		assert.True(t, wasAssigned)
		assert.Equal(t, "ws_conn_123", clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("Job assigned but agent disconnected", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_123", "ws_conn_123", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, jobID, organizationID).
			Return(mo.Some(agent), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_456"})
		mockAgents.On("CheckAgentHasActiveConnection", agent, []string{"ws_conn_456"}).
			Return(false)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, wasAssigned, err := useCase.TryAssignJobToAgent(ctx, jobID, organizationID)

		assert.NoError(t, err)
		assert.False(t, wasAssigned)
		assert.Empty(t, clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("New assignment to least loaded agent", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent1 := createTestAgent("agent_1", "ws_conn_1", organizationID)
		agent2 := createTestAgent("agent_2", "ws_conn_2", organizationID)

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, jobID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil)
		mockWS.On("GetClientIDs").Return([]string{"ws_conn_1", "ws_conn_2"})
		mockAgents.On("GetConnectedActiveAgents", ctx, organizationID, []string{"ws_conn_1", "ws_conn_2"}).
			Return([]*models.ActiveAgent{agent1, agent2}, nil)
		// Agent 1 has 2 jobs, Agent 2 has 1 job - should select agent 2
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent1.ID, organizationID).
			Return([]string{"job_a", "job_b"}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent2.ID, organizationID).
			Return([]string{"job_c"}, nil)
		mockAgents.On("AssignAgentToJob", ctx, agent2.ID, jobID, organizationID).
			Return(nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, wasAssigned, err := useCase.TryAssignJobToAgent(ctx, jobID, organizationID)

		assert.NoError(t, err)
		assert.True(t, wasAssigned)
		assert.Equal(t, "ws_conn_2", clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})

	t.Run("No connected agents", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		// Setup expectations
		mockAgents.On("GetAgentByJobID", ctx, jobID, organizationID).
			Return(mo.None[*models.ActiveAgent](), nil)
		mockWS.On("GetClientIDs").Return([]string{})
		mockAgents.On("GetConnectedActiveAgents", ctx, organizationID, []string{}).
			Return([]*models.ActiveAgent{}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		clientID, wasAssigned, err := useCase.TryAssignJobToAgent(ctx, jobID, organizationID)

		assert.NoError(t, err)
		assert.False(t, wasAssigned)
		assert.Empty(t, clientID)
		mockWS.AssertExpectations(t)
		mockAgents.AssertExpectations(t)
	})
}

// ValidateJobBelongsToAgent Tests
func TestValidateJobBelongsToAgent(t *testing.T) {
	ctx := context.Background()
	organizationID := "org_test123"
	agentID := "agent_123"
	jobID := "job_123"

	t.Run("Valid agent-job relationship", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		mockAgents.On("GetActiveAgentJobAssignments", ctx, agentID, organizationID).
			Return([]string{"job_456", jobID, "job_789"}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		err := useCase.ValidateJobBelongsToAgent(ctx, agentID, jobID, organizationID)

		assert.NoError(t, err)
		mockAgents.AssertExpectations(t)
	})

	t.Run("Invalid agent-job relationship", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		mockAgents.On("GetActiveAgentJobAssignments", ctx, agentID, organizationID).
			Return([]string{"job_456", "job_789"}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		err := useCase.ValidateJobBelongsToAgent(ctx, agentID, jobID, organizationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("agent %s is not assigned to job %s", agentID, jobID))
		mockAgents.AssertExpectations(t)
	})

	t.Run("Agent with no jobs", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		mockAgents.On("GetActiveAgentJobAssignments", ctx, agentID, organizationID).
			Return([]string{}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		err := useCase.ValidateJobBelongsToAgent(ctx, agentID, jobID, organizationID)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), fmt.Sprintf("agent %s is not assigned to job %s", agentID, jobID))
		mockAgents.AssertExpectations(t)
	})
}

// sortAgentsByLoad Tests
func TestSortAgentsByLoad(t *testing.T) {
	ctx := context.Background()
	organizationID := "org_test123"

	t.Run("Correct load calculation and sort order", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent1 := createTestAgent("agent_1", "ws_1", organizationID)
		agent2 := createTestAgent("agent_2", "ws_2", organizationID)
		agent3 := createTestAgent("agent_3", "ws_3", organizationID)

		// Agent 1: 3 jobs, Agent 2: 1 job, Agent 3: 2 jobs
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent1.ID, organizationID).
			Return([]string{"job_a", "job_b", "job_c"}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent2.ID, organizationID).
			Return([]string{"job_d"}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent3.ID, organizationID).
			Return([]string{"job_e", "job_f"}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		sorted, err := useCase.sortAgentsByLoad(ctx, []*models.ActiveAgent{agent1, agent2, agent3}, organizationID)

		require.NoError(t, err)
		require.Len(t, sorted, 3)

		// Should be sorted: agent2 (1 job), agent3 (2 jobs), agent1 (3 jobs)
		assert.Equal(t, agent2.ID, sorted[0].agent.ID)
		assert.Equal(t, 1, sorted[0].load)
		assert.Equal(t, agent3.ID, sorted[1].agent.ID)
		assert.Equal(t, 2, sorted[1].load)
		assert.Equal(t, agent1.ID, sorted[2].agent.ID)
		assert.Equal(t, 3, sorted[2].load)

		mockAgents.AssertExpectations(t)
	})

	t.Run("Empty agent list", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		sorted, err := useCase.sortAgentsByLoad(ctx, []*models.ActiveAgent{}, organizationID)

		assert.NoError(t, err)
		assert.Empty(t, sorted)
	})

	t.Run("Single agent", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent := createTestAgent("agent_1", "ws_1", organizationID)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent.ID, organizationID).
			Return([]string{"job_a", "job_b"}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		sorted, err := useCase.sortAgentsByLoad(ctx, []*models.ActiveAgent{agent}, organizationID)

		require.NoError(t, err)
		require.Len(t, sorted, 1)
		assert.Equal(t, agent.ID, sorted[0].agent.ID)
		assert.Equal(t, 2, sorted[0].load)
		mockAgents.AssertExpectations(t)
	})

	t.Run("Agents with equal load", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent1 := createTestAgent("agent_1", "ws_1", organizationID)
		agent2 := createTestAgent("agent_2", "ws_2", organizationID)

		// Both agents have 2 jobs
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent1.ID, organizationID).
			Return([]string{"job_a", "job_b"}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent2.ID, organizationID).
			Return([]string{"job_c", "job_d"}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		sorted, err := useCase.sortAgentsByLoad(ctx, []*models.ActiveAgent{agent1, agent2}, organizationID)

		require.NoError(t, err)
		require.Len(t, sorted, 2)
		assert.Equal(t, 2, sorted[0].load)
		assert.Equal(t, 2, sorted[1].load)
		mockAgents.AssertExpectations(t)
	})

	t.Run("Agents with no jobs", func(t *testing.T) {
		mockWS := &socketio.MockSocketIOClient{}
		mockAgents := &agents.MockAgentsService{}

		agent1 := createTestAgent("agent_1", "ws_1", organizationID)
		agent2 := createTestAgent("agent_2", "ws_2", organizationID)

		// Both agents have no jobs
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent1.ID, organizationID).
			Return([]string{}, nil)
		mockAgents.On("GetActiveAgentJobAssignments", ctx, agent2.ID, organizationID).
			Return([]string{}, nil)

		useCase := NewAgentsUseCase(mockWS, mockAgents)
		sorted, err := useCase.sortAgentsByLoad(ctx, []*models.ActiveAgent{agent1, agent2}, organizationID)

		require.NoError(t, err)
		require.Len(t, sorted, 2)
		assert.Equal(t, 0, sorted[0].load)
		assert.Equal(t, 0, sorted[1].load)
		mockAgents.AssertExpectations(t)
	})
}
