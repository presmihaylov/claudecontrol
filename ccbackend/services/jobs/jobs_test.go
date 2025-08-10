package jobs

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	agents "ccbackend/services/agents"
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
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrganizationID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	// Initialize real transaction manager for tests
	txManager := txmanager.NewTransactionManager(dbConn)
	service := NewJobsService(jobsRepo, processedSlackMessagesRepo, txManager)

	cleanup := func() {
		// Clean up test data
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(context.Background(), testIntegration.ID, testUser.ID)
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

			job, err := service.CreateJob(
				context.Background(),
				slackThreadTS,
				slackChannelID,
				"testuser",
				slackIntegrationID,
				testIntegration.OrganizationID,
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
			_, err := service.CreateJob(
				context.Background(),
				"",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				testIntegration.OrganizationID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.CreateJob(
				context.Background(),
				"test.thread.456",
				"",
				"testuser",
				slackIntegrationID,
				testIntegration.OrganizationID,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.CreateJob(
				context.Background(),
				"test.thread.456",
				"C1234567890",
				"testuser",
				"",
				testIntegration.OrganizationID,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})
	})

	t.Run("GetJobByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			createdJob, err := service.CreateJob(
				context.Background(),
				"test.thread.789",
				"C9876543210",
				"testuser",
				slackIntegrationID,
				testIntegration.OrganizationID,
			)
			require.NoError(t, err)

			// Fetch it by ID
			maybeFetchedJob, err := service.GetJobByID(
				context.Background(),
				createdJob.ID,
				testIntegration.OrganizationID,
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
			_, err := service.GetJobByID(context.Background(), "", testIntegration.OrganizationID)

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
				testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
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
					slackIntegrationID,
					testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
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
					slackIntegrationID,
					testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
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
				testIntegration.OrganizationID,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})
	})

	t.Run("DeleteJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := service.CreateJob(
				context.Background(),
				"delete.test.thread",
				"C1111111111",
				"testuser",
				slackIntegrationID,
				testIntegration.OrganizationID,
			)
			require.NoError(t, err)

			// Verify job exists
			maybeFetchedJob, err := service.GetJobByID(
				context.Background(),
				job.ID,
				testIntegration.OrganizationID,
			)
			require.NoError(t, err)
			require.True(t, maybeFetchedJob.IsPresent())
			fetchedJob := maybeFetchedJob.MustGet()
			assert.Equal(t, job.ID, fetchedJob.ID)

			// Delete the job
			err = service.DeleteJob(context.Background(), job.ID, slackIntegrationID, testIntegration.OrganizationID)
			require.NoError(t, err)

			// Verify job no longer exists
			maybeJob, err := service.GetJobByID(
				context.Background(),
				job.ID,
				testIntegration.OrganizationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeJob.IsPresent())
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := service.DeleteJob(context.Background(), "", slackIntegrationID, testIntegration.OrganizationID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			err := service.DeleteJob(context.Background(), core.NewID("j"), "", testIntegration.OrganizationID)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("j")

			err := service.DeleteJob(context.Background(), id, slackIntegrationID, testIntegration.OrganizationID)
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
	agentsRepo := db.NewPostgresAgentsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create shared test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)

	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrganizationID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")
	defer func() {
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(context.Background(), testIntegration.ID, testUser.ID)
	}()

	// Create both services using the same integration
	txManager := txmanager.NewTransactionManager(dbConn)
	jobsService := NewJobsService(jobsRepo, processedSlackMessagesRepo, txManager)
	agentsService := agents.NewAgentsService(agentsRepo)

	// Use the shared integration ID
	slackIntegrationID := testIntegration.ID
	organizationID := testIntegration.OrganizationID

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
		job, err := jobsService.CreateJob(
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
		job1, err := jobsService.CreateJob(
			context.Background(),
			"multi.thread.1",
			"C1111111111",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		job2, err := jobsService.CreateJob(
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

		job, err := jobsService.CreateJob(
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
		job, err := jobsService.CreateJob(
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
		err = jobsService.UpdateJobTimestamp(context.Background(), job.ID, slackIntegrationID, organizationID)
		require.NoError(t, err)

		// Get the job again to verify timestamp changed
		maybeUpdatedJob, err := jobsService.GetJobByID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		require.True(t, maybeUpdatedJob.IsPresent())
		updatedJob := maybeUpdatedJob.MustGet()

		// The updated_at should be later than the original
		assert.True(t, updatedJob.UpdatedAt.After(originalUpdatedAt), "Updated timestamp should be later than original")

		// Test with invalid job ID
		err = jobsService.UpdateJobTimestamp(context.Background(), "", slackIntegrationID, organizationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "job ID must be a valid ULID")
	})

	t.Run("GetIdleJobs", func(t *testing.T) {
		t.Run("JobWithNoMessages", func(t *testing.T) {
			// Create a job with no messages
			job, err := jobsService.CreateJob(
				context.Background(),
				"idle.no.messages",
				"C1111111111",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

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

		t.Run("JobWithIncompleteMessages", func(t *testing.T) {
			// Create a job and add a message that's not completed
			job, err := jobsService.CreateJob(
				context.Background(),
				"idle.incomplete.messages",
				"C2222222222",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			// Add a message in IN_PROGRESS state
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C2222222222",
				"1234567890.111111",
				"Hello world",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			// Job should not be idle because it has an incomplete message
			idleJobs, err := jobsService.GetIdleJobs(
				context.Background(),
				999,
				organizationID,
			) // Even with very high threshold
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with incomplete messages should not be idle")
		})

		t.Run("JobWithQueuedMessages", func(t *testing.T) {
			// Create a job and add a queued message
			job, err := jobsService.CreateJob(
				context.Background(),
				"idle.queued.messages",
				"C3333333333",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			// Add a message in QUEUED state
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C3333333333",
				"1234567890.222222",
				"Hello queued",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			// Job should not be idle because it has a queued message
			idleJobs, err := jobsService.GetIdleJobs(
				context.Background(),
				999,
				organizationID,
			) // Even with very high threshold
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with queued messages should not be idle")
		})

		t.Run("JobWithOnlyCompletedMessages", func(t *testing.T) {
			// Create a job and add only completed messages
			job, err := jobsService.CreateJob(
				context.Background(),
				"idle.completed.messages",
				"C4444444444",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			// Add a completed message
			message, err := jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C4444444444",
				"1234567890.333333",
				"Hello completed",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			// Since the message was just created, job should not be idle with 1 minute threshold
			idleJobs, err := jobsService.GetIdleJobs(context.Background(), 1, organizationID)
			require.NoError(t, err)

			assert.False(
				t,
				jobFoundInIdleList(job.ID, idleJobs),
				"Job with recently completed messages should not be idle",
			)

			// Now manipulate the timestamp to make the message old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateProcessedSlackMessageUpdatedAt(
				context.Background(),
				message.ID,
				oldTimestamp,
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(context.Background(), 5, organizationID)
			require.NoError(t, err)

			assert.True(t, jobFoundInIdleList(job.ID, idleJobs), "Job with old completed messages should be idle")
		})

		t.Run("JobWithMixedMessages", func(t *testing.T) {
			// Create a job with both completed and incomplete messages
			job, err := jobsService.CreateJob(
				context.Background(),
				"idle.mixed.messages",
				"C5555555555",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			// Add a completed message
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C5555555555",
				"1234567890.444444",
				"Hello completed",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			// Add an incomplete message
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C5555555555",
				"1234567890.555555",
				"Hello in progress",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			// Job should not be idle because it has incomplete messages
			idleJobs, err := jobsService.GetIdleJobs(
				context.Background(),
				999,
				organizationID,
			) // Even with very high threshold
			require.NoError(t, err)

			assert.False(
				t,
				jobFoundInIdleList(job.ID, idleJobs),
				"Job with mixed messages (including incomplete) should not be idle",
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

	t.Run("CreateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := jobsService.CreateJob(
				context.Background(),
				"test.thread.processed",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			slackChannelID := "C1234567890"
			slackTS := "1234567890.123456"
			textContent := "Hello world"
			status := models.ProcessedSlackMessageStatusQueued

			message, err := jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				slackChannelID,
				slackTS,
				textContent,
				slackIntegrationID,
				organizationID,
				status,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, message.ID)
			assert.Equal(t, job.ID, message.JobID)
			assert.Equal(t, slackChannelID, message.SlackChannelID)
			assert.Equal(t, slackTS, message.SlackTS)
			assert.Equal(t, textContent, message.TextContent)
			assert.Equal(t, status, message.Status)
			assert.False(t, message.CreatedAt.IsZero())
			assert.False(t, message.UpdatedAt.IsZero())
		})

		t.Run("NilJobID", func(t *testing.T) {
			_, err := jobsService.CreateProcessedSlackMessage(
				context.Background(),
				"",
				"C1234567890",
				"1234567890.123456",
				"Hello world",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			job, err := jobsService.CreateJob(
				context.Background(),
				"test.thread.empty.channel",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"",
				"1234567890.123456",
				"Hello world",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackTS", func(t *testing.T) {
			job, err := jobsService.CreateJob(
				context.Background(),
				"test.thread.empty.ts",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C1234567890",
				"",
				"Hello world",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "slack_ts cannot be empty", err.Error())
		})

		t.Run("EmptyTextContent", func(t *testing.T) {
			job, err := jobsService.CreateJob(
				context.Background(),
				"test.thread.empty.text",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C1234567890",
				"1234567890.123456",
				"",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "text_content cannot be empty", err.Error())
		})
	})

	t.Run("UpdateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job and processed slack message first
			job, err := jobsService.CreateJob(
				context.Background(),
				"test.thread.update",
				"C1234567890",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			message, err := jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C1234567890",
				"1234567890.123456",
				"Hello world",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			// Update the status
			newStatus := models.ProcessedSlackMessageStatusInProgress
			updatedMessage, err := jobsService.UpdateProcessedSlackMessage(
				context.Background(),
				message.ID,
				newStatus,
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.Equal(t, newStatus, updatedMessage.Status)
			assert.True(t, updatedMessage.UpdatedAt.After(message.UpdatedAt))
		})

		t.Run("NilID", func(t *testing.T) {
			_, err := jobsService.UpdateProcessedSlackMessage(
				context.Background(),
				"",
				models.ProcessedSlackMessageStatusCompleted,
				slackIntegrationID,
				organizationID,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "processed slack message ID must be a valid ULID")
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("j")

			_, err := jobsService.UpdateProcessedSlackMessage(
				context.Background(),
				id,
				models.ProcessedSlackMessageStatusCompleted,
				slackIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.True(t, errors.Is(err, core.ErrNotFound))
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

		job, err := jobsService.CreateJob(
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
		err = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID)
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

	t.Run("DeleteJobCascadesProcessedSlackMessages", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob(
			context.Background(),
			"cascade.delete.thread",
			"C9999999999",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)

		// Create multiple processed slack messages for this job
		message1, err := jobsService.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C9999999999",
			"1234567890.111111",
			"Hello world 1",
			slackIntegrationID,
			organizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		message2, err := jobsService.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C9999999999",
			"1234567890.222222",
			"Hello world 2",
			slackIntegrationID,
			organizationID,
			models.ProcessedSlackMessageStatusInProgress,
		)
		require.NoError(t, err)

		message3, err := jobsService.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C9999999999",
			"1234567890.333333",
			"Hello world 3",
			slackIntegrationID,
			organizationID,
			models.ProcessedSlackMessageStatusCompleted,
		)
		require.NoError(t, err)

		// Verify all messages exist
		_, err = jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message1.ID,
			organizationID,
		)
		require.NoError(t, err)
		_, err = jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message2.ID,
			organizationID,
		)
		require.NoError(t, err)
		_, err = jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message3.ID,
			organizationID,
		)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID)
		require.NoError(t, err)

		// Verify job is deleted
		maybeJob, err := jobsService.GetJobByID(context.Background(), job.ID, organizationID)
		require.NoError(t, err)
		assert.False(t, maybeJob.IsPresent())

		// Verify all processed slack messages are also deleted (cascade)
		maybeMessage1, err := jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message1.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.False(t, maybeMessage1.IsPresent())

		maybeMessage2, err := jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message2.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.False(t, maybeMessage2.IsPresent())

		maybeMessage3, err := jobsService.GetProcessedSlackMessageByID(
			context.Background(),
			message3.ID,
			organizationID,
		)
		require.NoError(t, err)
		assert.False(t, maybeMessage3.IsPresent())
	})

	t.Run("ProcessedSlackMessageStatusTransitions", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob(
			context.Background(),
			"status.transition.thread",
			"C9999999999",
			"testuser",
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)
		defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

		// Create a processed slack message
		message, err := jobsService.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C9999999999",
			"1234567890.444444",
			"Hello world transition",
			slackIntegrationID,
			organizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		// Test status transitions: QUEUED -> IN_PROGRESS -> COMPLETED
		updatedMessage, err := jobsService.UpdateProcessedSlackMessage(
			context.Background(),
			message.ID,
			models.ProcessedSlackMessageStatusInProgress,
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusInProgress, updatedMessage.Status)

		finalMessage, err := jobsService.UpdateProcessedSlackMessage(
			context.Background(),
			message.ID,
			models.ProcessedSlackMessageStatusCompleted,
			slackIntegrationID,
			organizationID,
		)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusCompleted, finalMessage.Status)
		assert.True(t, finalMessage.UpdatedAt.After(updatedMessage.UpdatedAt))
	})

	t.Run("GetJobsWithQueuedMessages", func(t *testing.T) {
		t.Run("NoJobsWithQueuedMessages", func(t *testing.T) {
			// No jobs exist yet, so should return empty list
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(
				context.Background(),
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.Empty(t, queuedJobs)
		})

		t.Run("JobsWithQueuedMessages", func(t *testing.T) {
			// Create multiple jobs
			job1, err := jobsService.CreateJob(
				context.Background(),
				"queued.test.thread.1",
				"C1111111111",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job1.ID, slackIntegrationID, organizationID) }()

			job2, err := jobsService.CreateJob(
				context.Background(),
				"queued.test.thread.2",
				"C2222222222",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job2.ID, slackIntegrationID, organizationID) }()

			job3, err := jobsService.CreateJob(
				context.Background(),
				"queued.test.thread.3",
				"C3333333333",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job3.ID, slackIntegrationID, organizationID) }()

			// Add messages with different statuses
			// Job1: QUEUED message (should be returned)
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job1.ID,
				"C1111111111",
				"1234567890.111111",
				"Queued message 1",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			// Job2: IN_PROGRESS message (should NOT be returned)
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job2.ID,
				"C2222222222",
				"1234567890.222222",
				"In progress message",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			// Job3: COMPLETED message (should NOT be returned)
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job3.ID,
				"C3333333333",
				"1234567890.333333",
				"Completed message",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(
				context.Background(),
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Should only return job1
			require.Len(t, queuedJobs, 1)
			assert.Equal(t, job1.ID, queuedJobs[0].ID)
			assert.NotNil(t, job1.SlackPayload)
			assert.NotNil(t, queuedJobs[0].SlackPayload)
			assert.Equal(t, job1.SlackPayload.ThreadTS, queuedJobs[0].SlackPayload.ThreadTS)
			assert.Equal(t, job1.SlackPayload.ChannelID, queuedJobs[0].SlackPayload.ChannelID)
		})

		t.Run("MultipleJobsWithQueuedMessages", func(t *testing.T) {
			// Create multiple jobs with queued messages
			job1, err := jobsService.CreateJob(
				context.Background(),
				"multi.queued.thread.1",
				"C4444444444",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job1.ID, slackIntegrationID, organizationID) }()

			job2, err := jobsService.CreateJob(
				context.Background(),
				"multi.queued.thread.2",
				"C5555555555",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job2.ID, slackIntegrationID, organizationID) }()

			// Add queued messages to both jobs
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job1.ID,
				"C4444444444",
				"1234567890.444444",
				"Queued message job1",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job2.ID,
				"C5555555555",
				"1234567890.555555",
				"Queued message job2",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(
				context.Background(),
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Should return both jobs, ordered by created_at ASC
			require.Len(t, queuedJobs, 2)

			// Find our test jobs in the results
			var foundJob1, foundJob2 bool
			for _, job := range queuedJobs {
				if job.ID == job1.ID {
					foundJob1 = true
					assert.NotNil(t, job.SlackPayload)
					assert.NotNil(t, job1.SlackPayload)
					assert.Equal(t, job1.SlackPayload.ThreadTS, job.SlackPayload.ThreadTS)
				}
				if job.ID == job2.ID {
					foundJob2 = true
					assert.NotNil(t, job.SlackPayload)
					assert.NotNil(t, job2.SlackPayload)
					assert.Equal(t, job2.SlackPayload.ThreadTS, job.SlackPayload.ThreadTS)
				}
			}
			assert.True(t, foundJob1, "Job1 should be in queued jobs list")
			assert.True(t, foundJob2, "Job2 should be in queued jobs list")
		})

		t.Run("JobWithMixedMessageStatuses", func(t *testing.T) {
			// Create a job with both queued and non-queued messages
			job, err := jobsService.CreateJob(
				context.Background(),
				"mixed.status.thread",
				"C6666666666",
				"testuser",
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(context.Background(), job.ID, slackIntegrationID, organizationID) }()

			// Add messages with different statuses
			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C6666666666",
				"1234567890.666666",
				"Queued message",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C6666666666",
				"1234567890.777777",
				"In progress message",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(
				context.Background(),
				job.ID,
				"C6666666666",
				"1234567890.888888",
				"Completed message",
				slackIntegrationID,
				organizationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(
				context.Background(),
				slackIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Should return the job because it has at least one queued message
			foundJob := false
			for _, queuedJob := range queuedJobs {
				if queuedJob.ID == job.ID {
					foundJob = true
					assert.NotNil(t, job.SlackPayload)
					assert.NotNil(t, queuedJob.SlackPayload)
					assert.Equal(t, job.SlackPayload.ThreadTS, queuedJob.SlackPayload.ThreadTS)
					break
				}
			}
			assert.True(t, foundJob, "Job with mixed statuses (including queued) should be returned")
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := jobsService.GetJobsWithQueuedMessages(context.Background(), "", organizationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "slack_integration_id must be a valid ULID")
		})

	})
}
