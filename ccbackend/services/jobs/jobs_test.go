package jobs

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	agents "ccbackend/services/agents"
	discordmessages "ccbackend/services/discordmessages"
	slackmessages "ccbackend/services/slackmessages"
	"ccbackend/services/txmanager"
	"ccbackend/testutils"
)

// Helper function to check if a job is in the idle jobs list
func jobFoundInIdleList(jobID string, idleJobs []*models.Job) bool {
	for _, idleJob := range idleJobs {
		if idleJob.ID == jobID {
			return true
		}
	}
	return false
}

func setupTestJobsService(t *testing.T) (*JobsService, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	processedSlackMessagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	processedDiscordMessagesRepo := db.NewPostgresProcessedDiscordMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrgID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	// Initialize real transaction manager and services for tests
	txManager := txmanager.NewTransactionManager(dbConn)
	slackMessagesService := slackmessages.NewSlackMessagesService(processedSlackMessagesRepo)
	discordMessagesService := discordmessages.NewDiscordMessagesService(processedDiscordMessagesRepo)
	service := NewJobsService(jobsRepo, slackMessagesService, discordMessagesService, txManager)

	cleanup := func() {
		// Clean up test data
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(
			context.Background(),
			testIntegration.ID,
			testUser.OrgID,
		)
		dbConn.Close()
	}

	return service, testIntegration, cleanup
}

func TestJobsService(t *testing.T) {
	service, testIntegration, cleanup := setupTestJobsService(t)
	defer cleanup()

	slackIntegrationID := testIntegration.ID

	t.Run("CreateJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			slackThreadTS := "test.thread.123"
			slackChannelID := "C1234567890"

			job, err := service.CreateSlackJob(
				context.Background(),
				slackThreadTS,
				slackChannelID,
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.NoError(t, err)

			assert.NotEmpty(t, job.ID)
			assert.NotNil(t, job.SlackPayload)
			assert.Equal(t, slackThreadTS, job.SlackPayload.ThreadTS)
			assert.Equal(t, slackChannelID, job.SlackPayload.ChannelID)
			assert.Equal(t, testIntegration.ID, job.SlackPayload.IntegrationID)
			assert.False(t, job.CreatedAt.IsZero())
			assert.False(t, job.UpdatedAt.IsZero())
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.CreateSlackJob(
				context.Background(),
				"",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.CreateSlackJob(
				context.Background(),
				"test.thread.456",
				"",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.CreateSlackJob(
				context.Background(),
				"test.thread.456",
				"C1234567890",
				"testuser",
				"",
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})
	})

	t.Run("GetJobByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			createdJob, err := service.CreateSlackJob(
				context.Background(),
				"test.thread.789",
				"C9876543210",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)

			// Fetch it by ID
			maybeFetchedJob, err := service.GetJobByID(
				context.Background(),
				createdJob.ID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			require.True(t, maybeFetchedJob.IsPresent())
			fetchedJob := maybeFetchedJob.MustGet()

			assert.Equal(t, createdJob.ID, fetchedJob.ID)
			assert.NotNil(t, createdJob.SlackPayload)
			assert.NotNil(t, fetchedJob.SlackPayload)
			assert.Equal(t, createdJob.SlackPayload.ThreadTS, fetchedJob.SlackPayload.ThreadTS)
			assert.Equal(t, createdJob.SlackPayload.ChannelID, fetchedJob.SlackPayload.ChannelID)
			assert.Equal(t, testIntegration.ID, fetchedJob.SlackPayload.IntegrationID)
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := service.GetJobByID(context.Background(), "", testIntegration.OrgID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			_, err := service.GetJobByID(context.Background(), core.NewID("j"), "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("j")

			maybeJob, err := service.GetJobByID(
				context.Background(),
				id,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			assert.False(t, maybeJob.IsPresent())
		})
	})

	t.Run("GetOrCreateJobForSlackThread", func(t *testing.T) {
		t.Run("CreateNew", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("new.thread.%d", time.Now().UnixNano())
			slackChannelID := "C5555555555"

			result, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				slackThreadTS,
				slackChannelID,
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, result.Job.ID)
			assert.NotNil(t, result.Job.SlackPayload)
			assert.Equal(t, slackThreadTS, result.Job.SlackPayload.ThreadTS)
			assert.Equal(t, slackChannelID, result.Job.SlackPayload.ChannelID)
			assert.Equal(t, testIntegration.ID, result.Job.SlackPayload.IntegrationID)
			assert.Equal(t, models.JobCreationStatusCreated, result.Status)

			// Cleanup
			defer func() {
				service.DeleteJob(
					context.Background(),
					result.Job.ID,
					testIntegration.OrgID,
				)
			}()
		})

		t.Run("GetExisting", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("existing.thread.%d", time.Now().UnixNano())
			slackChannelID := "C7777777777"

			// Create job first
			firstResult, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				slackThreadTS,
				slackChannelID,
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusCreated, firstResult.Status)

			// Get the same job again
			secondResult, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				slackThreadTS,
				slackChannelID,
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusNA, secondResult.Status)

			// Should be the same job
			assert.Equal(t, firstResult.Job.ID, secondResult.Job.ID)
			assert.NotNil(t, firstResult.Job.SlackPayload)
			assert.NotNil(t, secondResult.Job.SlackPayload)
			assert.Equal(t, firstResult.Job.SlackPayload.ThreadTS, secondResult.Job.SlackPayload.ThreadTS)
			assert.Equal(t, firstResult.Job.SlackPayload.ChannelID, secondResult.Job.SlackPayload.ChannelID)
			assert.Equal(t, testIntegration.ID, secondResult.Job.SlackPayload.IntegrationID)

			// Cleanup
			defer func() {
				service.DeleteJob(
					context.Background(),
					firstResult.Job.ID,
					testIntegration.OrgID,
				)
			}()
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				"",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				"test.thread.999",
				"",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread(
				context.Background(),
				"test.thread.999",
				"C1234567890",
				"testuser",
				"",
				testIntegration.OrgID,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})
	})

	t.Run("DeleteJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := service.CreateSlackJob(
				context.Background(),
				"delete.test.thread",
				"C1111111111",
				"testuser",
				slackIntegrationID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)

			// Verify job exists
			maybeFetchedJob, err := service.GetJobByID(
				context.Background(),
				job.ID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			require.True(t, maybeFetchedJob.IsPresent())
			fetchedJob := maybeFetchedJob.MustGet()
			assert.Equal(t, job.ID, fetchedJob.ID)

			// Delete the job
			err = service.DeleteJob(context.Background(), job.ID, testIntegration.OrgID)
			require.NoError(t, err)

			// Verify job no longer exists
			maybeJob, err := service.GetJobByID(
				context.Background(),
				job.ID,
				testIntegration.OrgID,
			)
			require.NoError(t, err)
			assert.False(t, maybeJob.IsPresent())
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := service.DeleteJob(context.Background(), "", testIntegration.OrgID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("EmptyOrganizationID", func(t *testing.T) {
			err := service.DeleteJob(context.Background(), core.NewID("j"), "")

			require.Error(t, err)
			assert.Contains(t, err.Error(), "organization_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("j")

			err := service.DeleteJob(context.Background(), id, testIntegration.OrgID)
			require.NoError(t, err)
		})
	})
}

func TestJobsAndAgentsIntegration(t *testing.T) {
	// Setup shared database connection and test data
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")
	defer dbConn.Close()

	// Create repositories
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	processedSlackMessagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	processedDiscordMessagesRepo := db.NewPostgresProcessedDiscordMessagesRepository(dbConn, cfg.DatabaseSchema)
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create shared test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)

	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrgID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")
	defer func() {
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(
			context.Background(),
			testIntegration.ID,
			testUser.OrgID,
		)
	}()

	// Create both services using the same integration
	txManager := txmanager.NewTransactionManager(dbConn)
	slackMessagesService := slackmessages.NewSlackMessagesService(processedSlackMessagesRepo)
	discordMessagesService := discordmessages.NewDiscordMessagesService(processedDiscordMessagesRepo)
	jobsService := NewJobsService(jobsRepo, slackMessagesService, discordMessagesService, txManager)
	agentsService := agents.NewAgentsService(agentsRepo)

	// Use the shared integration ID
	slackIntegrationID := testIntegration.ID
	organizationID := testIntegration.OrgID

	t.Run("JobAssignmentWorkflow", func(t *testing.T) {
		// Create an agent first
		agent, err := agentsService.UpsertActiveAgent(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent.ID, organizationID) }()

		// Create a job
		job, err := jobsService.CreateSlackJob(
			context.Background(),
			"integration.thread.123",
			"C1234567890",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignAgentToJob(context.Background(), agent.ID, job.ID, organizationID)
		require.NoError(t, err)

		// Verify agent has the job assigned
		maybeUpdatedAgent, err := agentsService.GetAgentByID(context.Background(), agent.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeUpdatedAgent.IsPresent())
		updatedAgent := maybeUpdatedAgent.MustGet()

		// Verify agent has the assigned job
		jobs, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			updatedAgent.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, job.ID, jobs[0])

		// Verify agent is no longer available
		availableAgents, err := agentsService.GetAvailableAgents(context.Background(), organizationID)
		require.NoError(t, err)

		// Should not find our agent in available list since it has a job
		foundInAvailable := false
		for _, availableAgent := range availableAgents {
			if availableAgent.ID == agent.ID {
				foundInAvailable = true
				break
			}
		}
		assert.False(t, foundInAvailable, "Agent with assigned job should not be in available list")

		// Unassign the job
		err = agentsService.UnassignAgentFromJob(context.Background(), agent.ID, job.ID, organizationID)
		require.NoError(t, err)

		// Verify agent is available again
		availableAgents, err = agentsService.GetAvailableAgents(context.Background(), organizationID)
		require.NoError(t, err)

		foundInAvailable = false
		for _, availableAgent := range availableAgents {
			if availableAgent.ID == agent.ID {
				foundInAvailable = true
				break
			}
		}
		assert.True(t, foundInAvailable, "Agent without assigned job should be in available list")
	})

	t.Run("MultipleAgentsJobAssignment", func(t *testing.T) {
		// Create multiple agents
		agent1, err := agentsService.UpsertActiveAgent(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent1.ID, organizationID) }()

		agent2, err := agentsService.UpsertActiveAgent(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent2.ID, organizationID) }()

		// Create multiple jobs
		job1, err := jobsService.CreateSlackJob(
			context.Background(),
			"multi.thread.1",
			"C1111111111",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		job2, err := jobsService.CreateSlackJob(
			context.Background(),
			"multi.thread.2",
			"C2222222222",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		// Assign different jobs to different agents
		err = agentsService.AssignAgentToJob(context.Background(), agent1.ID, job1.ID, organizationID)
		require.NoError(t, err)

		err = agentsService.AssignAgentToJob(context.Background(), agent2.ID, job2.ID, organizationID)
		require.NoError(t, err)

		// Verify both agents have their respective jobs
		maybeUpdatedAgent1, err := agentsService.GetAgentByID(context.Background(), agent1.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeUpdatedAgent1.IsPresent())
		updatedAgent1 := maybeUpdatedAgent1.MustGet()

		// Verify agent1 has the assigned job
		jobs1, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			updatedAgent1.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Len(t, jobs1, 1)
		assert.Equal(t, job1.ID, jobs1[0])

		maybeUpdatedAgent2, err := agentsService.GetAgentByID(context.Background(), agent2.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeUpdatedAgent2.IsPresent())
		updatedAgent2 := maybeUpdatedAgent2.MustGet()

		// Verify agent2 has the assigned job
		jobs2, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			updatedAgent2.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Len(t, jobs2, 1)
		assert.Equal(t, job2.ID, jobs2[0])

		// Verify no agents are available
		availableAgents, err := agentsService.GetAvailableAgents(context.Background(), organizationID)
		require.NoError(t, err)

		// Filter out our test agents from available list
		testAgentCount := 0
		for _, agent := range availableAgents {
			if agent.ID == agent1.ID || agent.ID == agent2.ID {
				testAgentCount++
			}
		}
		assert.Equal(t, 0, testAgentCount, "Both agents should be unavailable since they have jobs")
	})

	t.Run("GetAgentByJobID", func(t *testing.T) {
		// Create an agent and job
		agent, err := agentsService.UpsertActiveAgent(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent.ID, organizationID) }()

		job, err := jobsService.CreateSlackJob(
			context.Background(),
			"job.lookup.thread",
			"C9999999999",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		// Initially no agent should be assigned to this job
		maybeFoundAgent, err := agentsService.GetAgentByJobID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		assert.False(t, maybeFoundAgent.IsPresent())

		// Assign job to agent
		err = agentsService.AssignAgentToJob(context.Background(), agent.ID, job.ID, organizationID)
		require.NoError(t, err)

		// Now we should be able to find the agent by job ID
		maybeFoundAgent, err = agentsService.GetAgentByJobID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeFoundAgent.IsPresent())
		foundAgent := maybeFoundAgent.MustGet()
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, agent.WSConnectionID, foundAgent.WSConnectionID)

		// Verify found agent has the assigned job
		foundJobs, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			foundAgent.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Len(t, foundJobs, 1)
		assert.Equal(t, job.ID, foundJobs[0])
	})

	t.Run("GetAgentByWSConnectionID", func(t *testing.T) {
		// Create an agent
		wsConnectionID := core.NewID("wsc")
		agent, err := agentsService.UpsertActiveAgent(
			context.Background(),
			wsConnectionID,
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent.ID, organizationID) }()

		// Find agent by WebSocket connection ID
		maybeFoundAgent, err := agentsService.GetAgentByWSConnectionID(
			context.Background(),
			wsConnectionID,
			organizationID,
		)
		require.NoError(t, err)
		require.True(t, maybeFoundAgent.IsPresent())
		foundAgent := maybeFoundAgent.MustGet()
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, wsConnectionID, foundAgent.WSConnectionID)

		// Verify agent has no job assignments
		foundJobs, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			foundAgent.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Empty(t, foundJobs)

		// Test with non-existent connection ID
		maybeAgent, err := agentsService.GetAgentByWSConnectionID(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
		)
		require.NoError(t, err)
		assert.False(t, maybeAgent.IsPresent())

		// Test with empty connection ID
		_, err = agentsService.GetAgentByWSConnectionID(context.Background(), "", organizationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "ws_connection_id must be a valid ULID")
	})

	t.Run("UpdateJobTimestamp", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateSlackJob(
			context.Background(),
			"timestamp.test.thread",
			"C9999999999",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		originalUpdatedAt := job.UpdatedAt

		// Update the job timestamp
		err = jobsService.UpdateJobTimestamp(context.Background(), job.ID, organizationID)
		require.NoError(t, err)

		// Get the job again to verify timestamp changed
		maybeUpdatedJob, err := jobsService.GetJobByID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeUpdatedJob.IsPresent())
		updatedJob := maybeUpdatedJob.MustGet()

		// The updated_at should be later than the original
		assert.True(t, updatedJob.UpdatedAt.After(originalUpdatedAt), "Updated timestamp should be later than original")

		// Test with invalid job ID
		err = jobsService.UpdateJobTimestamp(context.Background(), "", organizationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job ID must be a valid ULID")
	})

	t.Run("GetIdleJobs", func(t *testing.T) {
		t.Run("JobWithNoMessages", func(t *testing.T) {
			// Create a job with no messages
			job, err := jobsService.CreateSlackJob(
				context.Background(),
				"idle.no.messages",
				"C1111111111",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, organizationID) }()

			// Since we just created the job, it shouldn't be idle
			idleJobs, err := jobsService.GetIdleJobs(context.Background(), 1, organizationID)
			require.NoError(t, err)

			// Filter out our test job - it should not be in idle list
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Newly created job should not be in idle list")

			// Now manipulate the job timestamp to make it old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateJobUpdatedAt(
				context.Background(),
				job.ID,
				oldTimestamp,
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(context.Background(), 5, organizationID)
			require.NoError(t, err)

			assert.True(
				t,
				jobFoundInIdleList(job.ID, idleJobs),
				"Job with old updated_at and no messages should be idle",
			)
		})

		t.Run("InvalidIdleMinutes", func(t *testing.T) {
			// Test with invalid idle minutes
			_, err := jobsService.GetIdleJobs(context.Background(), 0, organizationID)
			require.Error(t, err)
			assert.Equal(t, "idle minutes must be greater than 0", err.Error())

			_, err = jobsService.GetIdleJobs(context.Background(), -5, organizationID)
			require.Error(t, err)
			assert.Equal(t, "idle minutes must be greater than 0", err.Error())
		})
	})

	t.Run("DeleteJobWithAgentAssignment", func(t *testing.T) {
		// Create an agent and job
		agent, err := agentsService.UpsertActiveAgent(
			context.Background(),
			core.NewID("wsc"),
			organizationID,
			core.NewID("ccaid"),
		)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(context.Background(), agent.ID, organizationID) }()

		job, err := jobsService.CreateSlackJob(
			context.Background(),
			"delete.assigned.thread",
			"C8888888888",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignAgentToJob(context.Background(), agent.ID, job.ID, organizationID)
		require.NoError(t, err)

		// Verify assignment
		maybeAssignedAgent, err := agentsService.GetAgentByJobID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeAssignedAgent.IsPresent())
		assignedAgent := maybeAssignedAgent.MustGet()
		assert.Equal(t, agent.ID, assignedAgent.ID)

		// Unassign agent (simulating cleanup process)
		err = agentsService.UnassignAgentFromJob(context.Background(), agent.ID, job.ID, organizationID)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(context.Background(), job.ID, organizationID)
		require.NoError(t, err)

		// Verify job is deleted
		maybeJob, err := jobsService.GetJobByID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		assert.False(t, maybeJob.IsPresent())

		// Verify agent still exists but has no job assigned
		maybeRemainingAgent, err := agentsService.GetAgentByID(context.Background(), agent.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeRemainingAgent.IsPresent())
		remainingAgent := maybeRemainingAgent.MustGet()

		// Verify agent has no job assignments
		remainingJobs, err := agentsService.GetActiveAgentJobAssignments(
			context.Background(),
			remainingAgent.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Empty(t, remainingJobs)
	})
}
