package services

import (
	"testing"

	"ccbackend/db"
	"ccbackend/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*AgentsService, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	repo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	service := NewAgentsService(repo)

	cleanup := func() {
		dbConn.Close()
	}

	return service, cleanup
}

func TestAgentsService(t *testing.T) {
	service, cleanup := setupTestService(t)
	defer cleanup()

	t.Run("CreateActiveAgent", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-1"
			agent, err := service.CreateActiveAgent(wsConnectionID, nil)

			require.NoError(t, err)
			defer func() {
				// Cleanup: delete the agent we created
				_ = service.DeleteActiveAgent(agent.ID)
			}()

			assert.NotEqual(t, uuid.Nil, agent.ID)
			assert.Nil(t, agent.AssignedJobID)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			assert.False(t, agent.CreatedAt.IsZero())
			assert.False(t, agent.UpdatedAt.IsZero())

			// Verify agent exists in database
			fetchedAgent, err := service.GetAgentByID(agent.ID)
			require.NoError(t, err)
			assert.Equal(t, agent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
		})

		t.Run("WithAssignedJobID", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-2"
			jobID := uuid.New()

			agent, err := service.CreateActiveAgent(wsConnectionID, &jobID)

			require.NoError(t, err)
			defer func() {
				// Cleanup: delete the agent we created
				_ = service.DeleteActiveAgent(agent.ID)
			}()

			assert.NotEqual(t, uuid.Nil, agent.ID)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			require.NotNil(t, agent.AssignedJobID)
			assert.Equal(t, jobID, *agent.AssignedJobID)

			// Verify agent exists in database with correct job ID
			fetchedAgent, err := service.GetAgentByID(agent.ID)
			require.NoError(t, err)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			require.NotNil(t, fetchedAgent.AssignedJobID)
			assert.Equal(t, jobID, *fetchedAgent.AssignedJobID)
		})

		t.Run("EmptyWSConnectionID", func(t *testing.T) {
			_, err := service.CreateActiveAgent("", nil)

			require.Error(t, err)
			assert.Equal(t, "ws_connection_id cannot be empty", err.Error())
		})

	})

	t.Run("DeleteActiveAgent", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-3"
			agent, err := service.CreateActiveAgent(wsConnectionID, nil)
			require.NoError(t, err)

			err = service.DeleteActiveAgent(agent.ID)
			require.NoError(t, err)

			// Verify agent no longer exists
			_, err = service.GetAgentByID(agent.ID)
			assert.Error(t, err)
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := service.DeleteActiveAgent(uuid.Nil)

			require.Error(t, err)
			assert.Equal(t, "agent ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			err := service.DeleteActiveAgent(id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetAgentByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-4"
			jobID := uuid.New()

			createdAgent, err := service.CreateActiveAgent(wsConnectionID, &jobID)
			require.NoError(t, err)
			defer func() {
				// Cleanup: delete the agent we created
				_ = service.DeleteActiveAgent(createdAgent.ID)
			}()

			fetchedAgent, err := service.GetAgentByID(createdAgent.ID)
			require.NoError(t, err)

			assert.Equal(t, createdAgent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			require.NotNil(t, fetchedAgent.AssignedJobID)
			require.NotNil(t, createdAgent.AssignedJobID)
			assert.Equal(t, *createdAgent.AssignedJobID, *fetchedAgent.AssignedJobID)
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := service.GetAgentByID(uuid.Nil)

			require.Error(t, err)
			assert.Equal(t, "agent ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := service.GetAgentByID(id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetAvailableAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple agents - some with jobs, some without
			agent1, err := service.CreateActiveAgent("test-ws-1", nil)
			require.NoError(t, err)
			defer func() { _ = service.DeleteActiveAgent(agent1.ID) }()

			jobID := uuid.New()
			agent2, err := service.CreateActiveAgent("test-ws-2", &jobID)
			require.NoError(t, err)
			defer func() { _ = service.DeleteActiveAgent(agent2.ID) }()

			agent3, err := service.CreateActiveAgent("test-ws-3", nil)
			require.NoError(t, err)
			defer func() { _ = service.DeleteActiveAgent(agent3.ID) }()

			// Get available agents (should only return agent1 and agent3)
			availableAgents, err := service.GetAvailableAgents()
			require.NoError(t, err)

			// Should have at least 2 available agents (the ones we created without jobs)
			foundAgent1 := false
			foundAgent3 := false
			for _, agent := range availableAgents {
				assert.Nil(t, agent.AssignedJobID)
				if agent.ID == agent1.ID {
					foundAgent1 = true
				}
				if agent.ID == agent3.ID {
					foundAgent3 = true
				}
			}

			assert.True(t, foundAgent1, "Should find agent1 in available agents")
			assert.True(t, foundAgent3, "Should find agent3 in available agents")
		})

		t.Run("EmptyResult", func(t *testing.T) {
			// Clear all agents first
			err := service.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Create only agents with jobs
			jobID1 := uuid.New()
			agent1, err := service.CreateActiveAgent("test-ws-busy-1", &jobID1)
			require.NoError(t, err)
			defer func() { _ = service.DeleteActiveAgent(agent1.ID) }()

			jobID2 := uuid.New()
			agent2, err := service.CreateActiveAgent("test-ws-busy-2", &jobID2)
			require.NoError(t, err)
			defer func() { _ = service.DeleteActiveAgent(agent2.ID) }()

			// Get available agents (should be empty)
			availableAgents, err := service.GetAvailableAgents()
			require.NoError(t, err)
			assert.Empty(t, availableAgents)
		})
	})

	t.Run("DeleteAllActiveAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple agents
			agent1, err := service.CreateActiveAgent("test-ws-delete-1", nil)
			require.NoError(t, err)

			jobID := uuid.New()
			agent2, err := service.CreateActiveAgent("test-ws-delete-2", &jobID)
			require.NoError(t, err)

			agent3, err := service.CreateActiveAgent("test-ws-delete-3", nil)
			require.NoError(t, err)

			// Verify agents exist
			_, err = service.GetAgentByID(agent1.ID)
			require.NoError(t, err)
			_, err = service.GetAgentByID(agent2.ID)
			require.NoError(t, err)
			_, err = service.GetAgentByID(agent3.ID)
			require.NoError(t, err)

			// Delete all agents
			err = service.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Verify all agents are gone
			_, err = service.GetAgentByID(agent1.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			_, err = service.GetAgentByID(agent2.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			_, err = service.GetAgentByID(agent3.ID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			// Verify available agents is empty
			availableAgents, err := service.GetAvailableAgents()
			require.NoError(t, err)
			assert.Empty(t, availableAgents)
		})

		t.Run("EmptyTable", func(t *testing.T) {
			// Clear all agents first
			err := service.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Delete all again (should not error on empty table)
			err = service.DeleteAllActiveAgents()
			require.NoError(t, err)
		})
	})
}
