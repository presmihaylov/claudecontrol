package services

import (
	"testing"

	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestService(t *testing.T) (*AgentsService, *JobsService, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	messagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(t, slackIntegrationsRepo, testUser.ID)

	agentsService := NewAgentsService(agentsRepo)
	jobsService := NewJobsService(jobsRepo, messagesRepo)

	cleanup := func() {
		// Clean up test data
		_ = slackIntegrationsRepo.DeleteSlackIntegrationByID(testIntegration.ID, testUser.ID)
		dbConn.Close()
	}

	return agentsService, jobsService, testIntegration, cleanup
}

func TestAgentsService(t *testing.T) {
	agentsService, jobsService, testIntegration, cleanup := setupTestService(t)
	defer cleanup()
	
	slackIntegrationID := testIntegration.ID.String()

	t.Run("CreateActiveAgent", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-1"
			agent, err := agentsService.CreateActiveAgent(wsConnectionID, slackIntegrationID)

			require.NoError(t, err)
			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID)
			}()

			assert.NotEqual(t, uuid.Nil, agent.ID)
			// Verify agent has no job assignments
			jobs, err := agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Empty(t, jobs)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			assert.Equal(t, testIntegration.ID, agent.SlackIntegrationID)
			assert.False(t, agent.CreatedAt.IsZero())
			assert.False(t, agent.UpdatedAt.IsZero())

			// Verify agent exists in database
			fetchedAgent, err := agentsService.GetAgentByID(agent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, agent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.ID, fetchedAgent.SlackIntegrationID)
		})

		t.Run("WithAssignedJobID", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-2"

			// Create a real job first
			job, err := jobsService.CreateJob("test.thread.assigned", "C1234567890", slackIntegrationID)
			require.NoError(t, err)

			agent, err := agentsService.CreateActiveAgent(wsConnectionID, slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent
			err = agentsService.AssignAgentToJob(agent.ID, job.ID, slackIntegrationID)
			require.NoError(t, err)

			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID)
			}()

			assert.NotEqual(t, uuid.Nil, agent.ID)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			assert.Equal(t, testIntegration.ID, agent.SlackIntegrationID)
			// Verify agent has the assigned job
			jobs, err := agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Len(t, jobs, 1)
			assert.Equal(t, job.ID, jobs[0])

			// Verify agent exists in database with correct job ID
			fetchedAgent, err := agentsService.GetAgentByID(agent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.ID, fetchedAgent.SlackIntegrationID)
			// Verify fetched agent has the assigned job
			fetchedJobs, err := agentsService.GetActiveAgentJobAssignments(fetchedAgent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Len(t, fetchedJobs, 1)
			assert.Equal(t, job.ID, fetchedJobs[0])
		})

		t.Run("EmptyWSConnectionID", func(t *testing.T) {
			_, err := agentsService.CreateActiveAgent("", slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "ws_connection_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := agentsService.CreateActiveAgent("test-ws-connection", "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

	})

	t.Run("DeleteActiveAgent", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-3"
			agent, err := agentsService.CreateActiveAgent(wsConnectionID, slackIntegrationID)
			require.NoError(t, err)

			err = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID)
			require.NoError(t, err)

			// Verify agent no longer exists
			_, err = agentsService.GetAgentByID(agent.ID, slackIntegrationID)
			assert.Error(t, err)
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := agentsService.DeleteActiveAgent(uuid.Nil, slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "agent ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			err := agentsService.DeleteActiveAgent(uuid.New(), "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			err := agentsService.DeleteActiveAgent(id, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetAgentByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := "test-ws-connection-4"

			// Create a real job first
			job, err := jobsService.CreateJob("test.thread.getbyid", "C1234567890", slackIntegrationID)
			require.NoError(t, err)

			createdAgent, err := agentsService.CreateActiveAgent(wsConnectionID, slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent
			err = agentsService.AssignAgentToJob(createdAgent.ID, job.ID, slackIntegrationID)
			require.NoError(t, err)
			
			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(createdAgent.ID, slackIntegrationID)
			}()

			fetchedAgent, err := agentsService.GetAgentByID(createdAgent.ID, slackIntegrationID)
			require.NoError(t, err)

			assert.Equal(t, createdAgent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.ID, fetchedAgent.SlackIntegrationID)
			
			// Verify agent has the assigned job
			jobs, err := agentsService.GetActiveAgentJobAssignments(fetchedAgent.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Len(t, jobs, 1)
			assert.Equal(t, job.ID, jobs[0])
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := agentsService.GetAgentByID(uuid.Nil, slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "agent ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := agentsService.GetAgentByID(uuid.New(), "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := agentsService.GetAgentByID(id, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetAvailableAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple agents - some with jobs, some without
			agent1, err := agentsService.CreateActiveAgent("test-ws-1", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(agent1.ID, slackIntegrationID) }()

			// Create a real job first
			job, err := jobsService.CreateJob("test.thread.available", "C1234567890", slackIntegrationID)
			require.NoError(t, err)

			agent2, err := agentsService.CreateActiveAgent("test-ws-2", slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent2
			err = agentsService.AssignAgentToJob(agent2.ID, job.ID, slackIntegrationID)
			require.NoError(t, err)
			
			defer func() { _ = agentsService.DeleteActiveAgent(agent2.ID, slackIntegrationID) }()

			agent3, err := agentsService.CreateActiveAgent("test-ws-3", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(agent3.ID, slackIntegrationID) }()

			// Get available agents (should only return agent1 and agent3)
			availableAgents, err := agentsService.GetAvailableAgents(slackIntegrationID)
			require.NoError(t, err)

			// Should have at least 2 available agents (the ones we created without jobs)
			foundAgent1 := false
			foundAgent3 := false
			for _, agent := range availableAgents {
				// Verify agent has no job assignments
				jobs, err := agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
				require.NoError(t, err)
				assert.Empty(t, jobs)
				assert.Equal(t, testIntegration.ID, agent.SlackIntegrationID)
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
			// Clear all agents first (this clears across all integrations)
			err := agentsService.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Create only agents with jobs
			// Create real jobs first
			job1, err := jobsService.CreateJob("test.thread.busy1", "C1111111111", slackIntegrationID)
			require.NoError(t, err)
			
			agent1, err := agentsService.CreateActiveAgent("test-ws-busy-1", slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent1
			err = agentsService.AssignAgentToJob(agent1.ID, job1.ID, slackIntegrationID)
			require.NoError(t, err)
			
			defer func() { _ = agentsService.DeleteActiveAgent(agent1.ID, slackIntegrationID) }()

			job2, err := jobsService.CreateJob("test.thread.busy2", "C2222222222", slackIntegrationID)
			require.NoError(t, err)
			agent2, err := agentsService.CreateActiveAgent("test-ws-busy-2", slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent2
			err = agentsService.AssignAgentToJob(agent2.ID, job2.ID, slackIntegrationID)
			require.NoError(t, err)
			
			defer func() { _ = agentsService.DeleteActiveAgent(agent2.ID, slackIntegrationID) }()

			// Get available agents (should be empty)
			availableAgents, err := agentsService.GetAvailableAgents(slackIntegrationID)
			require.NoError(t, err)
			assert.Empty(t, availableAgents)
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := agentsService.GetAvailableAgents("")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})
	})

	t.Run("DeleteAllActiveAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple agents
			agent1, err := agentsService.CreateActiveAgent("test-ws-delete-1", slackIntegrationID)
			require.NoError(t, err)

			// Create a real job first
			job, err := jobsService.CreateJob("test.thread.delete", "C1234567890", slackIntegrationID)
			require.NoError(t, err)

			agent2, err := agentsService.CreateActiveAgent("test-ws-delete-2", slackIntegrationID)
			require.NoError(t, err)
			
			// Assign job to agent2
			err = agentsService.AssignAgentToJob(agent2.ID, job.ID, slackIntegrationID)
			require.NoError(t, err)

			agent3, err := agentsService.CreateActiveAgent("test-ws-delete-3", slackIntegrationID)
			require.NoError(t, err)

			// Verify agents exist
			_, err = agentsService.GetAgentByID(agent1.ID, slackIntegrationID)
			require.NoError(t, err)
			_, err = agentsService.GetAgentByID(agent2.ID, slackIntegrationID)
			require.NoError(t, err)
			_, err = agentsService.GetAgentByID(agent3.ID, slackIntegrationID)
			require.NoError(t, err)

			// Delete all agents (note: this method deletes across ALL integrations)
			err = agentsService.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Verify all agents are gone
			_, err = agentsService.GetAgentByID(agent1.ID, slackIntegrationID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			_, err = agentsService.GetAgentByID(agent2.ID, slackIntegrationID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			_, err = agentsService.GetAgentByID(agent3.ID, slackIntegrationID)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "not found")

			// Verify available agents is empty
			availableAgents, err := agentsService.GetAvailableAgents(slackIntegrationID)
			require.NoError(t, err)
			assert.Empty(t, availableAgents)
		})

		t.Run("EmptyTable", func(t *testing.T) {
			// Clear all agents first
			err := agentsService.DeleteAllActiveAgents()
			require.NoError(t, err)

			// Delete all again (should not error on empty table)
			err = agentsService.DeleteAllActiveAgents()
			require.NoError(t, err)
		})
	})
}
