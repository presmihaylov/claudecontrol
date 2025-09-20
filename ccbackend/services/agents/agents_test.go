package agents

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/clients/socketio"
	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
	discordmessages "ccbackend/services/discordmessages"
	jobs "ccbackend/services/jobs"
	slackmessages "ccbackend/services/slackmessages"
	"ccbackend/services/txmanager"
	"ccbackend/testutils"
)

func setupTestService(t *testing.T) (*AgentsService, services.JobsService, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	messagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	discordMessagesRepo := db.NewPostgresProcessedDiscordMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrgID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	txManager := txmanager.NewTransactionManager(dbConn)
	agentsService := NewAgentsService(agentsRepo, nil)
	slackMessagesService := slackmessages.NewSlackMessagesService(messagesRepo)
	discordMessagesService := discordmessages.NewDiscordMessagesService(discordMessagesRepo)
	jobsService := jobs.NewJobsService(jobsRepo, slackMessagesService, discordMessagesService, txManager)

	cleanup := func() {
		// Clean up test data
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(
			context.Background(),
			testUser.OrgID,
			testIntegration.ID,
		)
		dbConn.Close()
	}

	return agentsService, jobsService, testIntegration, cleanup
}

func TestAgentsService(t *testing.T) {
	agentsService, jobsService, testIntegration, cleanup := setupTestService(t)
	defer cleanup()

	orgID := testIntegration.OrgID

	t.Run("UpsertActiveAgent", func(t *testing.T) {
		t.Run("Success with agent ID", func(t *testing.T) {
			wsConnectionID := core.NewID("wsc")
			agentID := core.NewID("ccaid")
			agent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)

			require.NoError(t, err)
			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent.ID)
			}()

			assert.NotEmpty(t, agent.ID)
			assert.NotNil(t, agent.CCAgentID)
			assert.Equal(t, agentID, agent.CCAgentID)
			// Verify agent has no job assignments
			jobs, err := agentsService.GetActiveAgentJobAssignments(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			assert.Empty(t, jobs)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			assert.Equal(t, testIntegration.OrgID, agent.OrgID)
			assert.False(t, agent.CreatedAt.IsZero())
			assert.False(t, agent.UpdatedAt.IsZero())

			// Verify agent exists in database
			maybeFetchedAgent, err := agentsService.GetAgentByID(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			require.True(t, maybeFetchedAgent.IsPresent())
			fetchedAgent := maybeFetchedAgent.MustGet()
			assert.Equal(t, agent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.OrgID, fetchedAgent.OrgID)
			assert.Equal(t, agent.CCAgentID, fetchedAgent.CCAgentID)
		})

		t.Run("WithAssignedJobID", func(t *testing.T) {
			wsConnectionID := core.NewID("wsc")
			agentID := core.NewID("ccaid")

			// Create a real job first
			job, err := jobsService.CreateSlackJob(
				context.Background(),
				testIntegration.OrgID,
				"test.thread.assigned",
				"C1234567890",
				"testuser",
				testIntegration.ID,
			)
			require.NoError(t, err)

			agent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)
			require.NoError(t, err)

			// Assign job to agent
			err = agentsService.AssignAgentToJob(context.Background(), orgID, agent.ID, job.ID)
			require.NoError(t, err)

			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent.ID)
			}()

			assert.NotEmpty(t, agent.ID)
			assert.Equal(t, wsConnectionID, agent.WSConnectionID)
			assert.Equal(t, testIntegration.OrgID, agent.OrgID)
			// Verify agent has the assigned job
			jobs, err := agentsService.GetActiveAgentJobAssignments(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			assert.Len(t, jobs, 1)
			assert.Equal(t, job.ID, jobs[0])

			// Verify agent exists in database with correct job ID
			maybeFetchedAgent, err := agentsService.GetAgentByID(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			require.True(t, maybeFetchedAgent.IsPresent())
			fetchedAgent := maybeFetchedAgent.MustGet()
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.OrgID, fetchedAgent.OrgID)
			// Verify fetched agent has the assigned job
			fetchedJobs, err := agentsService.GetActiveAgentJobAssignments(
				context.Background(),
				orgID,
				fetchedAgent.ID,
			)
			require.NoError(t, err)
			assert.Len(t, fetchedJobs, 1)
			assert.Equal(t, job.ID, fetchedJobs[0])
		})

		t.Run("EmptyWSConnectionID", func(t *testing.T) {
			agentID := core.NewID("ccaid")
			_, err := agentsService.UpsertActiveAgent(context.Background(), orgID, "", agentID, nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "ws_connection_id must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			agentID := core.NewID("ccaid")
			_, err := agentsService.UpsertActiveAgent(context.Background(), "", core.NewID("wsc"), agentID, nil)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("UpsertBehavior - Updates existing agent", func(t *testing.T) {
			wsConnectionID1 := core.NewID("wsc")
			wsConnectionID2 := core.NewID("wsc")
			agentID := core.NewID("ccaid")

			// First upsert - creates the agent
			agent1, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID1,
				agentID,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			originalID := agent1.ID
			originalCreatedAt := agent1.CreatedAt

			// Second upsert with same ccagent_id but different ws_connection_id - should update existing
			agent2, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID2,
				agentID,
				nil,
			)
			require.NoError(t, err)

			// Should return the same ID (existing record was updated)
			assert.Equal(t, originalID, agent2.ID)
			assert.Equal(t, wsConnectionID2, agent2.WSConnectionID)
			assert.Equal(t, agentID, agent2.CCAgentID)
			assert.Equal(t, originalCreatedAt, agent2.CreatedAt)
			assert.True(t, agent2.UpdatedAt.After(agent1.UpdatedAt))

			// Verify only one agent exists for this ccagent_id
			allAgents, err := agentsService.agentsRepo.GetAllActiveAgents(context.Background(), orgID)
			require.NoError(t, err)

			agentCount := 0
			for _, agent := range allAgents {
				if agent.CCAgentID == agentID {
					agentCount++
				}
			}
			assert.Equal(t, 1, agentCount, "Should have exactly one agent for this ccagent_id")
		})

		t.Run("UpsertBehavior - Creates new agent for different ccagent_id", func(t *testing.T) {
			wsConnectionID := core.NewID("wsc")
			agentID1 := core.NewID("ccaid")
			agentID2 := core.NewID("ccaid")

			// Create first agent
			agent1, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID1,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			// Create second agent with different ccagent_id - should create new record
			agent2, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID2,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			// Should have different IDs
			assert.NotEqual(t, agent1.ID, agent2.ID)
			assert.Equal(t, agentID1, agent1.CCAgentID)
			assert.Equal(t, agentID2, agent2.CCAgentID)
		})
	})

	t.Run("DeleteActiveAgent", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := core.NewID("wsc")
			agentID := core.NewID("ccaid")
			agent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)
			require.NoError(t, err)

			err = agentsService.DeleteActiveAgent(context.Background(), orgID, agent.ID)
			require.NoError(t, err)

			// Verify agent no longer exists
			maybeAgent, err := agentsService.GetAgentByID(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			assert.False(t, maybeAgent.IsPresent())
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := agentsService.DeleteActiveAgent(context.Background(), orgID, "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "agent ID must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			err := agentsService.DeleteActiveAgent(context.Background(), "", core.NewID("ccaid"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("ccaid")

			err := agentsService.DeleteActiveAgent(context.Background(), orgID, id)
			require.Error(t, err)
			assert.True(t, errors.Is(err, core.ErrNotFound))
		})
	})

	t.Run("GetAgentByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			wsConnectionID := core.NewID("wsc")

			// Create a real job first
			job, err := jobsService.CreateSlackJob(
				context.Background(),
				testIntegration.OrgID,
				"test.thread.getbyid",
				"C1234567890",
				"testuser",
				testIntegration.ID,
			)
			require.NoError(t, err)

			agentID := core.NewID("ccaid")
			createdAgent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)
			require.NoError(t, err)

			// Assign job to agent
			err = agentsService.AssignAgentToJob(context.Background(), orgID, createdAgent.ID, job.ID)
			require.NoError(t, err)

			defer func() {
				// Cleanup: delete the agent we created
				_ = agentsService.DeleteActiveAgent(context.Background(), orgID, createdAgent.ID)
			}()

			maybeFetchedAgent, err := agentsService.GetAgentByID(
				context.Background(),
				orgID,
				createdAgent.ID,
			)
			require.NoError(t, err)
			require.True(t, maybeFetchedAgent.IsPresent())
			fetchedAgent := maybeFetchedAgent.MustGet()

			assert.Equal(t, createdAgent.ID, fetchedAgent.ID)
			assert.Equal(t, wsConnectionID, fetchedAgent.WSConnectionID)
			assert.Equal(t, testIntegration.OrgID, fetchedAgent.OrgID)

			// Verify agent has the assigned job
			jobs, err := agentsService.GetActiveAgentJobAssignments(
				context.Background(),
				orgID,
				fetchedAgent.ID,
			)
			require.NoError(t, err)
			assert.Len(t, jobs, 1)
			assert.Equal(t, job.ID, jobs[0])
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := agentsService.GetAgentByID(context.Background(), orgID, "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "agent ID must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			_, err := agentsService.GetAgentByID(context.Background(), "", core.NewID("ccaid"))

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("ccaid")

			maybeAgent, err := agentsService.GetAgentByID(context.Background(), orgID, id)
			require.NoError(t, err)
			assert.False(t, maybeAgent.IsPresent())
		})
	})

	t.Run("GetAvailableAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple agents - some with jobs, some without
			agentID1 := core.NewID("ccaid")
			agent1, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID1,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			// Create a real job first
			job, err := jobsService.CreateSlackJob(
				context.Background(),
				testIntegration.OrgID,
				"test.thread.available",
				"C1234567890",
				"testuser",
				testIntegration.ID,
			)
			require.NoError(t, err)

			agentID2 := core.NewID("ccaid")
			agent2, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID2,
				nil,
			)
			require.NoError(t, err)

			// Assign job to agent2
			err = agentsService.AssignAgentToJob(context.Background(), orgID, agent2.ID, job.ID)
			require.NoError(t, err)

			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			agentID3 := core.NewID("ccaid")
			agent3, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID3,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent3.ID) }()

			// Get available agents (should only return agent1 and agent3)
			availableAgents, err := agentsService.GetAvailableAgents(context.Background(), orgID)
			require.NoError(t, err)

			// Should have at least 2 available agents (the ones we created without jobs)
			foundAgent1 := false
			foundAgent3 := false
			for _, agent := range availableAgents {
				// Verify agent has no job assignments
				jobs, err := agentsService.GetActiveAgentJobAssignments(
					context.Background(),
					orgID,
					agent.ID,
				)
				require.NoError(t, err)
				assert.Empty(t, jobs)
				assert.Equal(t, testIntegration.OrgID, agent.OrgID)
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
			// Test with no available agents - all have jobs assigned

			// Create only agents with jobs
			// Create real jobs first
			job1, err := jobsService.CreateSlackJob(
				context.Background(),
				testIntegration.OrgID,
				"test.thread.busy1",
				"C1111111111",
				"testuser",
				testIntegration.ID,
			)
			require.NoError(t, err)

			agentIDBusy1 := core.NewID("ccaid")
			agent1, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentIDBusy1,
				nil,
			)
			require.NoError(t, err)

			// Assign job to agent1
			err = agentsService.AssignAgentToJob(context.Background(), orgID, agent1.ID, job1.ID)
			require.NoError(t, err)

			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			job2, err := jobsService.CreateSlackJob(
				context.Background(),
				testIntegration.OrgID,
				"test.thread.busy2",
				"C2222222222",
				"testuser",
				testIntegration.ID,
			)
			require.NoError(t, err)
			agentIDBusy2 := core.NewID("ccaid")
			agent2, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentIDBusy2,
				nil,
			)
			require.NoError(t, err)

			// Assign job to agent2
			err = agentsService.AssignAgentToJob(context.Background(), orgID, agent2.ID, job2.ID)
			require.NoError(t, err)

			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			// Get available agents (should be empty)
			availableAgents, err := agentsService.GetAvailableAgents(context.Background(), orgID)
			require.NoError(t, err)
			assert.Empty(t, availableAgents)
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			_, err := agentsService.GetAvailableAgents(context.Background(), "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})
	})

	t.Run("UpdateAgentLastActiveAt", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create an agent
			wsConnectionID := core.NewID("wsc")
			agentID := core.NewID("ccaid")
			agent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent.ID) }()

			// Get initial last_active_at timestamp
			maybeInitialAgent, err := agentsService.GetAgentByID(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			require.True(t, maybeInitialAgent.IsPresent())
			initialAgent := maybeInitialAgent.MustGet()
			initialLastActive := initialAgent.LastActiveAt

			// Wait a bit to ensure timestamp difference
			time.Sleep(10 * time.Millisecond)

			// Update last_active_at
			err = agentsService.UpdateAgentLastActiveAt(context.Background(), orgID, wsConnectionID)
			require.NoError(t, err)

			// Verify the timestamp was updated
			maybeUpdatedAgent, err := agentsService.GetAgentByID(context.Background(), orgID, agent.ID)
			require.NoError(t, err)
			require.True(t, maybeUpdatedAgent.IsPresent())
			updatedAgent := maybeUpdatedAgent.MustGet()
			assert.True(t, updatedAgent.LastActiveAt.After(initialLastActive),
				"last_active_at should be updated to a more recent time")
		})

		t.Run("EmptyWSConnectionID", func(t *testing.T) {
			err := agentsService.UpdateAgentLastActiveAt(context.Background(), orgID, "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "ws_connection_id must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			err := agentsService.UpdateAgentLastActiveAt(context.Background(), "", core.NewID("wsc"))
			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			err := agentsService.UpdateAgentLastActiveAt(context.Background(), orgID, core.NewID("wsc"))
			require.Error(t, err)
			assert.True(t, errors.Is(err, core.ErrNotFound))
		})
	})

	t.Run("GetInactiveAgents", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create agents with different last_active_at timestamps
			// Agent 1 - recently active
			wsConnectionID1 := core.NewID("wsc")
			agentID1 := core.NewID("ccaid")
			agent1, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID1,
				agentID1,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			// Agent 2 - inactive (we'll manually set an old timestamp)
			wsConnectionID2 := core.NewID("wsc")
			agentID2 := core.NewID("ccaid")
			agent2, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID2,
				agentID2,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			// Manually update agent2's last_active_at to be old (>15 minutes ago)
			// We need to access the repository directly for this test
			cfg, err := testutils.LoadTestConfig()
			require.NoError(t, err)
			dbConn, err := db.NewConnection(cfg.DatabaseURL)
			require.NoError(t, err)

			// Update agent2 to have an old last_active_at timestamp
			oldTimestamp := time.Now().Add(-20 * time.Minute)
			_, err = dbConn.Exec("UPDATE "+cfg.DatabaseSchema+".active_agents SET last_active_at = $1 WHERE id = $2",
				oldTimestamp, agent2.ID)
			require.NoError(t, err)

			// Get inactive agents with 15 minute threshold
			inactiveAgents, err := agentsService.GetInactiveAgents(context.Background(), orgID, 15)
			require.NoError(t, err)

			// Should find agent2 but not agent1
			foundAgent1 := false
			foundAgent2 := false
			for _, agent := range inactiveAgents {
				if agent.ID == agent1.ID {
					foundAgent1 = true
				}
				if agent.ID == agent2.ID {
					foundAgent2 = true
				}
			}

			assert.False(t, foundAgent1, "Recently active agent should not be in inactive list")
			assert.True(t, foundAgent2, "Old agent should be in inactive list")
		})

		t.Run("EmptyResult", func(t *testing.T) {
			// Test with only recently active agents

			// Create only recently active agents
			wsConnectionID := core.NewID("wsc")
			agentID := core.NewID("ccaid")
			agent, err := agentsService.UpsertActiveAgent(
				context.Background(),
				orgID,
				wsConnectionID,
				agentID,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), orgID, agent.ID) }()

			// Get inactive agents with 10 minute threshold
			inactiveAgents, err := agentsService.GetInactiveAgents(context.Background(), orgID, 10)
			require.NoError(t, err)
			assert.Empty(t, inactiveAgents)
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			_, err := agentsService.GetInactiveAgents(context.Background(), "", 10)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("InvalidThreshold", func(t *testing.T) {
			_, err := agentsService.GetInactiveAgents(context.Background(), orgID, 0)
			require.Error(t, err)
			assert.Equal(t, "inactive threshold must be positive", err.Error())

			_, err = agentsService.GetInactiveAgents(context.Background(), orgID, -5)
			require.Error(t, err)
			assert.Equal(t, "inactive threshold must be positive", err.Error())
		})
	})

	t.Run("DisconnectAllActiveAgentsByOrganization", func(t *testing.T) {
		// Create a mock Socket.IO client for these tests
		mockSocketIO := &socketio.MockSocketIOClient{}

		// Create test service with mock
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err, "Failed to create database connection")
		defer dbConn.Close()

		agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
		testServiceWithMock := NewAgentsService(agentsRepo, mockSocketIO)

		t.Run("Success - disconnects all agents", func(t *testing.T) {
			// Create multiple test agents
			agentID1 := core.NewID("ccaid")
			agent1, err := testServiceWithMock.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID1,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = testServiceWithMock.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			agentID2 := core.NewID("ccaid")
			agent2, err := testServiceWithMock.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID2,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = testServiceWithMock.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			// Set up mock expectations - both agents should be disconnected
			mockSocketIO.On("DisconnectClientByID", agentID1).Return(nil)
			mockSocketIO.On("DisconnectClientByID", agentID2).Return(nil)

			// Call the disconnect function
			err = testServiceWithMock.DisconnectAllActiveAgentsByOrganization(context.Background(), orgID)
			require.NoError(t, err)

			// Verify all mock expectations were met
			mockSocketIO.AssertExpectations(t)
		})

		t.Run("Success - no agents to disconnect", func(t *testing.T) {
			// Reset mock for this test
			mockSocketIO.ExpectedCalls = nil
			mockSocketIO.Calls = nil

			// Call disconnect with no agents
			err = testServiceWithMock.DisconnectAllActiveAgentsByOrganization(context.Background(), orgID)
			require.NoError(t, err)

			// Should not call DisconnectClientByID at all
			mockSocketIO.AssertNotCalled(t, "DisconnectClientByID")
		})

		t.Run("Fails immediately on first disconnect error", func(t *testing.T) {
			// Reset mock for this test
			mockSocketIO.ExpectedCalls = nil
			mockSocketIO.Calls = nil

			// Create test agents
			agentID1 := core.NewID("ccaid")
			agent1, err := testServiceWithMock.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID1,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = testServiceWithMock.DeleteActiveAgent(context.Background(), orgID, agent1.ID) }()

			agentID2 := core.NewID("ccaid")
			agent2, err := testServiceWithMock.UpsertActiveAgent(
				context.Background(),
				orgID,
				core.NewID("wsc"),
				agentID2,
				nil,
			)
			require.NoError(t, err)
			defer func() { _ = testServiceWithMock.DeleteActiveAgent(context.Background(), orgID, agent2.ID) }()

			// Set up mock expectations - first succeeds, second fails
			mockSocketIO.On("DisconnectClientByID", agentID1).Return(nil)
			mockSocketIO.On("DisconnectClientByID", agentID2).Return(fmt.Errorf("disconnect failed"))

			// Call the disconnect function
			err = testServiceWithMock.DisconnectAllActiveAgentsByOrganization(context.Background(), orgID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "failed to disconnect agent")
			assert.Contains(t, err.Error(), agentID2) // Should mention the specific agent that failed
			assert.Contains(t, err.Error(), "disconnect failed")

			// Verify all mock expectations were met (both calls should still happen)
			mockSocketIO.AssertExpectations(t)
		})

		t.Run("Error - empty organization ID", func(t *testing.T) {
			// Reset mock for this test
			mockSocketIO.ExpectedCalls = nil
			mockSocketIO.Calls = nil

			// Call with empty organization ID
			err = testServiceWithMock.DisconnectAllActiveAgentsByOrganization(context.Background(), "")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization ID must be a valid ULID")

			// Should not call DisconnectClientByID
			mockSocketIO.AssertNotCalled(t, "DisconnectClientByID")
		})

		t.Run("Error - invalid organization ULID", func(t *testing.T) {
			// Reset mock for this test
			mockSocketIO.ExpectedCalls = nil
			mockSocketIO.Calls = nil

			// Call with invalid organization ID
			err = testServiceWithMock.DisconnectAllActiveAgentsByOrganization(context.Background(), "invalid-ulid")
			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization ID must be a valid ULID")

			// Should not call DisconnectClientByID
			mockSocketIO.AssertNotCalled(t, "DisconnectClientByID")
		})
	})
}
