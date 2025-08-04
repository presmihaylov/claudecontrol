package services

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"
)

// Helper function to check if a job is in the idle jobs list
func jobFoundInIdleList(jobID uuid.UUID, idleJobs []*models.Job) bool {
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
	testIntegration := testutils.CreateTestSlackIntegration(t, slackIntegrationsRepo, testUser.ID)

	service := NewJobsService(jobsRepo, processedSlackMessagesRepo)

	cleanup := func() {
		// Clean up test data
		_ = slackIntegrationsRepo.DeleteSlackIntegrationByID(testIntegration.ID, testUser.ID)
		dbConn.Close()
	}

	return service, testIntegration, cleanup
}

func TestJobsService(t *testing.T) {
	service, testIntegration, cleanup := setupTestJobsService(t)
	defer cleanup()

	slackIntegrationID := testIntegration.ID.String()

	t.Run("CreateJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			slackThreadTS := "test.thread.123"
			slackChannelID := "C1234567890"

			job, err := service.CreateJob(slackThreadTS, slackChannelID, "testuser", slackIntegrationID)

			require.NoError(t, err)

			assert.NotEqual(t, uuid.Nil, job.ID)
			assert.Equal(t, slackThreadTS, job.SlackThreadTS)
			assert.Equal(t, slackChannelID, job.SlackChannelID)
			assert.Equal(t, testIntegration.ID, job.SlackIntegrationID)
			assert.False(t, job.CreatedAt.IsZero())
			assert.False(t, job.UpdatedAt.IsZero())
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.CreateJob("", "C1234567890", "testuser", slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.CreateJob("test.thread.456", "", "testuser", slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.CreateJob("test.thread.456", "C1234567890", "testuser", "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})
	})

	t.Run("GetJobByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			createdJob, err := service.CreateJob("test.thread.789", "C9876543210", "testuser", slackIntegrationID)
			require.NoError(t, err)

			// Fetch it by ID
			fetchedJob, err := service.GetJobByID(createdJob.ID, slackIntegrationID)
			require.NoError(t, err)

			assert.Equal(t, createdJob.ID, fetchedJob.ID)
			assert.Equal(t, createdJob.SlackThreadTS, fetchedJob.SlackThreadTS)
			assert.Equal(t, createdJob.SlackChannelID, fetchedJob.SlackChannelID)
			assert.Equal(t, testIntegration.ID, fetchedJob.SlackIntegrationID)
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := service.GetJobByID(uuid.Nil, slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.GetJobByID(uuid.New(), "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := service.GetJobByID(id, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetOrCreateJobForSlackThread", func(t *testing.T) {
		t.Run("CreateNew", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("new.thread.%d", time.Now().UnixNano())
			slackChannelID := "C5555555555"

			result, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID, "testuser", slackIntegrationID)

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, result.Job.ID)
			assert.Equal(t, slackThreadTS, result.Job.SlackThreadTS)
			assert.Equal(t, slackChannelID, result.Job.SlackChannelID)
			assert.Equal(t, testIntegration.ID, result.Job.SlackIntegrationID)
			assert.Equal(t, models.JobCreationStatusCreated, result.Status)

			// Cleanup
			defer func() {
				service.DeleteJob(result.Job.ID, slackIntegrationID)
			}()
		})

		t.Run("GetExisting", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("existing.thread.%d", time.Now().UnixNano())
			slackChannelID := "C7777777777"

			// Create job first
			firstResult, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID, "testuser", slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusCreated, firstResult.Status)

			// Get the same job again
			secondResult, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID, "testuser", slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusNA, secondResult.Status)

			// Should be the same job
			assert.Equal(t, firstResult.Job.ID, secondResult.Job.ID)
			assert.Equal(t, firstResult.Job.SlackThreadTS, secondResult.Job.SlackThreadTS)
			assert.Equal(t, firstResult.Job.SlackChannelID, secondResult.Job.SlackChannelID)
			assert.Equal(t, testIntegration.ID, secondResult.Job.SlackIntegrationID)

			// Cleanup
			defer func() {
				service.DeleteJob(firstResult.Job.ID, slackIntegrationID)
			}()
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread("", "C1234567890", "testuser", slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread("test.thread.999", "", "testuser", slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread("test.thread.999", "C1234567890", "testuser", "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})
	})

	t.Run("DeleteJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := service.CreateJob("delete.test.thread", "C1111111111", "testuser", slackIntegrationID)
			require.NoError(t, err)

			// Verify job exists
			fetchedJob, err := service.GetJobByID(job.ID, slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, job.ID, fetchedJob.ID)

			// Delete the job
			err = service.DeleteJob(job.ID, slackIntegrationID)
			require.NoError(t, err)

			// Verify job no longer exists
			_, err = service.GetJobByID(job.ID, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := service.DeleteJob(uuid.Nil, slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			err := service.DeleteJob(uuid.New(), "")

			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			err := service.DeleteJob(id, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
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
	testIntegration := testutils.CreateTestSlackIntegration(t, slackIntegrationsRepo, testUser.ID)
	defer func() {
		_ = slackIntegrationsRepo.DeleteSlackIntegrationByID(testIntegration.ID, testUser.ID)
	}()

	// Create both services using the same integration
	jobsService := NewJobsService(jobsRepo, processedSlackMessagesRepo)
	agentsService := NewAgentsService(agentsRepo)

	// Use the shared integration ID
	slackIntegrationID := testIntegration.ID.String()

	t.Run("JobAssignmentWorkflow", func(t *testing.T) {
		// Create an agent first
		agent, err := agentsService.UpsertActiveAgent("test-ws-integration", slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID) }()

		// Create a job
		job, err := jobsService.CreateJob("integration.thread.123", "C1234567890", "testuser", slackIntegrationID)
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignAgentToJob(agent.ID, job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent has the job assigned
		updatedAgent, err := agentsService.GetAgentByID(agent.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent has the assigned job
		jobs, err := agentsService.GetActiveAgentJobAssignments(updatedAgent.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Len(t, jobs, 1)
		assert.Equal(t, job.ID, jobs[0])

		// Verify agent is no longer available
		availableAgents, err := agentsService.GetAvailableAgents(slackIntegrationID)
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
		err = agentsService.UnassignAgentFromJob(agent.ID, job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent is available again
		availableAgents, err = agentsService.GetAvailableAgents(slackIntegrationID)
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
		agent1, err := agentsService.UpsertActiveAgent("test-ws-multi-1", slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent1.ID, slackIntegrationID) }()

		agent2, err := agentsService.UpsertActiveAgent("test-ws-multi-2", slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent2.ID, slackIntegrationID) }()

		// Create multiple jobs
		job1, err := jobsService.CreateJob("multi.thread.1", "C1111111111", "testuser", slackIntegrationID)
		require.NoError(t, err)

		job2, err := jobsService.CreateJob("multi.thread.2", "C2222222222", "testuser", slackIntegrationID)
		require.NoError(t, err)

		// Assign different jobs to different agents
		err = agentsService.AssignAgentToJob(agent1.ID, job1.ID, slackIntegrationID)
		require.NoError(t, err)

		err = agentsService.AssignAgentToJob(agent2.ID, job2.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify both agents have their respective jobs
		updatedAgent1, err := agentsService.GetAgentByID(agent1.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent1 has the assigned job
		jobs1, err := agentsService.GetActiveAgentJobAssignments(updatedAgent1.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Len(t, jobs1, 1)
		assert.Equal(t, job1.ID, jobs1[0])

		updatedAgent2, err := agentsService.GetAgentByID(agent2.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent2 has the assigned job
		jobs2, err := agentsService.GetActiveAgentJobAssignments(updatedAgent2.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Len(t, jobs2, 1)
		assert.Equal(t, job2.ID, jobs2[0])

		// Verify no agents are available
		availableAgents, err := agentsService.GetAvailableAgents(slackIntegrationID)
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
		agent, err := agentsService.UpsertActiveAgent("test-ws-job-lookup", slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID) }()

		job, err := jobsService.CreateJob("job.lookup.thread", "C9999999999", "testuser", slackIntegrationID)
		require.NoError(t, err)

		// Initially no agent should be assigned to this job
		foundAgent, err := agentsService.GetAgentByJobID(job.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Nil(t, foundAgent)

		// Assign job to agent
		err = agentsService.AssignAgentToJob(agent.ID, job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Now we should be able to find the agent by job ID
		foundAgent, err = agentsService.GetAgentByJobID(job.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, agent.WSConnectionID, foundAgent.WSConnectionID)

		// Verify found agent has the assigned job
		foundJobs, err := agentsService.GetActiveAgentJobAssignments(foundAgent.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Len(t, foundJobs, 1)
		assert.Equal(t, job.ID, foundJobs[0])
	})

	t.Run("GetAgentByWSConnectionID", func(t *testing.T) {
		// Create an agent
		wsConnectionID := "test-ws-connection-lookup"
		agent, err := agentsService.UpsertActiveAgent(wsConnectionID, slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID) }()

		// Find agent by WebSocket connection ID
		foundAgent, err := agentsService.GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, wsConnectionID, foundAgent.WSConnectionID)

		// Verify agent has no job assignments
		foundJobs, err := agentsService.GetActiveAgentJobAssignments(foundAgent.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Empty(t, foundJobs)

		// Test with non-existent connection ID
		_, err = agentsService.GetAgentByWSConnectionID("non-existent-connection", slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test with empty connection ID
		_, err = agentsService.GetAgentByWSConnectionID("", slackIntegrationID)
		require.Error(t, err)
		assert.Equal(t, "ws_connection_id cannot be empty", err.Error())
	})

	t.Run("UpdateJobTimestamp", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("timestamp.test.thread", "C9999999999", "testuser", slackIntegrationID)
		require.NoError(t, err)

		originalUpdatedAt := job.UpdatedAt

		// Update the job timestamp
		err = jobsService.UpdateJobTimestamp(job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Get the job again to verify timestamp changed
		updatedJob, err := jobsService.GetJobByID(job.ID, slackIntegrationID)
		require.NoError(t, err)

		// The updated_at should be later than the original
		assert.True(t, updatedJob.UpdatedAt.After(originalUpdatedAt), "Updated timestamp should be later than original")

		// Test with invalid job ID
		err = jobsService.UpdateJobTimestamp(uuid.Nil, slackIntegrationID)
		require.Error(t, err)
		assert.Equal(t, "job ID cannot be nil", err.Error())
	})

	t.Run("GetIdleJobs", func(t *testing.T) {
		t.Run("JobWithNoMessages", func(t *testing.T) {
			// Create a job with no messages
			job, err := jobsService.CreateJob("idle.no.messages", "C1111111111", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Since we just created the job, it shouldn't be idle
			idleJobs, err := jobsService.GetIdleJobs(1)
			require.NoError(t, err)

			// Filter out our test job - it should not be in idle list
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Newly created job should not be in idle list")

			// Now manipulate the job timestamp to make it old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateJobUpdatedAt(job.ID, oldTimestamp, slackIntegrationID)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(5)
			require.NoError(t, err)

			assert.True(t, jobFoundInIdleList(job.ID, idleJobs), "Job with old updated_at and no messages should be idle")
		})

		t.Run("JobWithIncompleteMessages", func(t *testing.T) {
			// Create a job and add a message that's not completed
			job, err := jobsService.CreateJob("idle.incomplete.messages", "C2222222222", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Add a message in IN_PROGRESS state
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C2222222222", "1234567890.111111", "Hello world", slackIntegrationID, models.ProcessedSlackMessageStatusInProgress)
			require.NoError(t, err)

			// Job should not be idle because it has an incomplete message
			idleJobs, err := jobsService.GetIdleJobs(999) // Even with very high threshold
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with incomplete messages should not be idle")
		})

		t.Run("JobWithQueuedMessages", func(t *testing.T) {
			// Create a job and add a queued message
			job, err := jobsService.CreateJob("idle.queued.messages", "C3333333333", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Add a message in QUEUED state
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C3333333333", "1234567890.222222", "Hello queued", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Job should not be idle because it has a queued message
			idleJobs, err := jobsService.GetIdleJobs(999) // Even with very high threshold
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with queued messages should not be idle")
		})

		t.Run("JobWithOnlyCompletedMessages", func(t *testing.T) {
			// Create a job and add only completed messages
			job, err := jobsService.CreateJob("idle.completed.messages", "C4444444444", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Add a completed message
			message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C4444444444", "1234567890.333333", "Hello completed", slackIntegrationID, models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Since the message was just created, job should not be idle with 1 minute threshold
			idleJobs, err := jobsService.GetIdleJobs(1)
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with recently completed messages should not be idle")

			// Now manipulate the timestamp to make the message old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateProcessedSlackMessageUpdatedAt(message.ID, oldTimestamp, slackIntegrationID)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(5)
			require.NoError(t, err)

			assert.True(t, jobFoundInIdleList(job.ID, idleJobs), "Job with old completed messages should be idle")
		})

		t.Run("JobWithMixedMessages", func(t *testing.T) {
			// Create a job with both completed and incomplete messages
			job, err := jobsService.CreateJob("idle.mixed.messages", "C5555555555", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Add a completed message
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C5555555555", "1234567890.444444", "Hello completed", slackIntegrationID, models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Add an incomplete message
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C5555555555", "1234567890.555555", "Hello in progress", slackIntegrationID, models.ProcessedSlackMessageStatusInProgress)
			require.NoError(t, err)

			// Job should not be idle because it has incomplete messages
			idleJobs, err := jobsService.GetIdleJobs(999) // Even with very high threshold
			require.NoError(t, err)

			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with mixed messages (including incomplete) should not be idle")
		})

		t.Run("InvalidIdleMinutes", func(t *testing.T) {
			// Test with invalid idle minutes
			_, err := jobsService.GetIdleJobs(0)
			require.Error(t, err)
			assert.Equal(t, "idle minutes must be greater than 0", err.Error())

			_, err = jobsService.GetIdleJobs(-5)
			require.Error(t, err)
			assert.Equal(t, "idle minutes must be greater than 0", err.Error())
		})
	})

	t.Run("CreateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := jobsService.CreateJob("test.thread.processed", "C1234567890", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			slackChannelID := "C1234567890"
			slackTS := "1234567890.123456"
			textContent := "Hello world"
			status := models.ProcessedSlackMessageStatusQueued

			message, err := jobsService.CreateProcessedSlackMessage(job.ID, slackChannelID, slackTS, textContent, slackIntegrationID, status)

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, message.ID)
			assert.Equal(t, job.ID, message.JobID)
			assert.Equal(t, slackChannelID, message.SlackChannelID)
			assert.Equal(t, slackTS, message.SlackTS)
			assert.Equal(t, textContent, message.TextContent)
			assert.Equal(t, status, message.Status)
			assert.False(t, message.CreatedAt.IsZero())
			assert.False(t, message.UpdatedAt.IsZero())
		})

		t.Run("NilJobID", func(t *testing.T) {
			_, err := jobsService.CreateProcessedSlackMessage(uuid.Nil, "C1234567890", "1234567890.123456", "Hello world", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.channel", "C1234567890", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "", "1234567890.123456", "Hello world", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackTS", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.ts", "C1234567890", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "", "Hello world", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "slack_ts cannot be empty", err.Error())
		})

		t.Run("EmptyTextContent", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.text", "C1234567890", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "1234567890.123456", "", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "text_content cannot be empty", err.Error())
		})
	})

	t.Run("UpdateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job and processed slack message first
			job, err := jobsService.CreateJob("test.thread.update", "C1234567890", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "1234567890.123456", "Hello world", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Update the status
			newStatus := models.ProcessedSlackMessageStatusInProgress
			updatedMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, newStatus, slackIntegrationID)
			require.NoError(t, err)
			assert.Equal(t, newStatus, updatedMessage.Status)
			assert.True(t, updatedMessage.UpdatedAt.After(message.UpdatedAt))
		})

		t.Run("NilID", func(t *testing.T) {
			_, err := jobsService.UpdateProcessedSlackMessage(uuid.Nil, models.ProcessedSlackMessageStatusCompleted, slackIntegrationID)

			require.Error(t, err)
			assert.Equal(t, "processed slack message ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := jobsService.UpdateProcessedSlackMessage(id, models.ProcessedSlackMessageStatusCompleted, slackIntegrationID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("DeleteJobWithAgentAssignment", func(t *testing.T) {
		// Create an agent and job
		agent, err := agentsService.UpsertActiveAgent("test-ws-delete-job", slackIntegrationID, uuid.New())
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID) }()

		job, err := jobsService.CreateJob("delete.assigned.thread", "C8888888888", "testuser", slackIntegrationID)
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignAgentToJob(agent.ID, job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify assignment
		assignedAgent, err := agentsService.GetAgentByJobID(job.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, assignedAgent.ID)

		// Unassign agent (simulating cleanup process)
		err = agentsService.UnassignAgentFromJob(agent.ID, job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify job is deleted
		_, err = jobsService.GetJobByID(job.ID, slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify agent still exists but has no job assigned
		remainingAgent, err := agentsService.GetAgentByID(agent.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify agent has no job assignments
		remainingJobs, err := agentsService.GetActiveAgentJobAssignments(remainingAgent.ID, slackIntegrationID)
		require.NoError(t, err)
		assert.Empty(t, remainingJobs)
	})

	t.Run("DeleteJobCascadesProcessedSlackMessages", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("cascade.delete.thread", "C9999999999", "testuser", slackIntegrationID)
		require.NoError(t, err)

		// Create multiple processed slack messages for this job
		message1, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.111111", "Hello world 1", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
		require.NoError(t, err)

		message2, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.222222", "Hello world 2", slackIntegrationID, models.ProcessedSlackMessageStatusInProgress)
		require.NoError(t, err)

		message3, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.333333", "Hello world 3", slackIntegrationID, models.ProcessedSlackMessageStatusCompleted)
		require.NoError(t, err)

		// Verify all messages exist
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message1.ID, slackIntegrationID)
		require.NoError(t, err)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message2.ID, slackIntegrationID)
		require.NoError(t, err)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message3.ID, slackIntegrationID)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(job.ID, slackIntegrationID)
		require.NoError(t, err)

		// Verify job is deleted
		_, err = jobsService.GetJobByID(job.ID, slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify all processed slack messages are also deleted (cascade)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message1.ID, slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message2.ID, slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message3.ID, slackIntegrationID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ProcessedSlackMessageStatusTransitions", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("status.transition.thread", "C9999999999", "testuser", slackIntegrationID)
		require.NoError(t, err)
		defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

		// Create a processed slack message
		message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.444444", "Hello world transition", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
		require.NoError(t, err)

		// Test status transitions: QUEUED -> IN_PROGRESS -> COMPLETED
		updatedMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, models.ProcessedSlackMessageStatusInProgress, slackIntegrationID)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusInProgress, updatedMessage.Status)

		finalMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, models.ProcessedSlackMessageStatusCompleted, slackIntegrationID)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusCompleted, finalMessage.Status)
		assert.True(t, finalMessage.UpdatedAt.After(updatedMessage.UpdatedAt))
	})

	t.Run("GetJobsWithQueuedMessages", func(t *testing.T) {
		t.Run("NoJobsWithQueuedMessages", func(t *testing.T) {
			// No jobs exist yet, so should return empty list
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
			require.NoError(t, err)
			assert.Empty(t, queuedJobs)
		})

		t.Run("JobsWithQueuedMessages", func(t *testing.T) {
			// Create multiple jobs
			job1, err := jobsService.CreateJob("queued.test.thread.1", "C1111111111", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job1.ID, slackIntegrationID) }()

			job2, err := jobsService.CreateJob("queued.test.thread.2", "C2222222222", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job2.ID, slackIntegrationID) }()

			job3, err := jobsService.CreateJob("queued.test.thread.3", "C3333333333", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job3.ID, slackIntegrationID) }()

			// Add messages with different statuses
			// Job1: QUEUED message (should be returned)
			_, err = jobsService.CreateProcessedSlackMessage(job1.ID, "C1111111111", "1234567890.111111", "Queued message 1", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Job2: IN_PROGRESS message (should NOT be returned)
			_, err = jobsService.CreateProcessedSlackMessage(job2.ID, "C2222222222", "1234567890.222222", "In progress message", slackIntegrationID, models.ProcessedSlackMessageStatusInProgress)
			require.NoError(t, err)

			// Job3: COMPLETED message (should NOT be returned)
			_, err = jobsService.CreateProcessedSlackMessage(job3.ID, "C3333333333", "1234567890.333333", "Completed message", slackIntegrationID, models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
			require.NoError(t, err)

			// Should only return job1
			require.Len(t, queuedJobs, 1)
			assert.Equal(t, job1.ID, queuedJobs[0].ID)
			assert.Equal(t, job1.SlackThreadTS, queuedJobs[0].SlackThreadTS)
			assert.Equal(t, job1.SlackChannelID, queuedJobs[0].SlackChannelID)
		})

		t.Run("MultipleJobsWithQueuedMessages", func(t *testing.T) {
			// Create multiple jobs with queued messages
			job1, err := jobsService.CreateJob("multi.queued.thread.1", "C4444444444", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job1.ID, slackIntegrationID) }()

			job2, err := jobsService.CreateJob("multi.queued.thread.2", "C5555555555", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job2.ID, slackIntegrationID) }()

			// Add queued messages to both jobs
			_, err = jobsService.CreateProcessedSlackMessage(job1.ID, "C4444444444", "1234567890.444444", "Queued message job1", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(job2.ID, "C5555555555", "1234567890.555555", "Queued message job2", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
			require.NoError(t, err)

			// Should return both jobs, ordered by created_at ASC
			require.Len(t, queuedJobs, 2)

			// Find our test jobs in the results
			var foundJob1, foundJob2 bool
			for _, job := range queuedJobs {
				if job.ID == job1.ID {
					foundJob1 = true
					assert.Equal(t, job1.SlackThreadTS, job.SlackThreadTS)
				}
				if job.ID == job2.ID {
					foundJob2 = true
					assert.Equal(t, job2.SlackThreadTS, job.SlackThreadTS)
				}
			}
			assert.True(t, foundJob1, "Job1 should be in queued jobs list")
			assert.True(t, foundJob2, "Job2 should be in queued jobs list")
		})

		t.Run("JobWithMixedMessageStatuses", func(t *testing.T) {
			// Create a job with both queued and non-queued messages
			job, err := jobsService.CreateJob("mixed.status.thread", "C6666666666", "testuser", slackIntegrationID)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID) }()

			// Add messages with different statuses
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C6666666666", "1234567890.666666", "Queued message", slackIntegrationID, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C6666666666", "1234567890.777777", "In progress message", slackIntegrationID, models.ProcessedSlackMessageStatusInProgress)
			require.NoError(t, err)

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C6666666666", "1234567890.888888", "Completed message", slackIntegrationID, models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Get jobs with queued messages
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
			require.NoError(t, err)

			// Should return the job because it has at least one queued message
			foundJob := false
			for _, queuedJob := range queuedJobs {
				if queuedJob.ID == job.ID {
					foundJob = true
					assert.Equal(t, job.SlackThreadTS, queuedJob.SlackThreadTS)
					break
				}
			}
			assert.True(t, foundJob, "Job with mixed statuses (including queued) should be returned")
		})

		t.Run("EmptySlackIntegrationID", func(t *testing.T) {
			_, err := jobsService.GetJobsWithQueuedMessages("")
			require.Error(t, err)
			assert.Equal(t, "slack_integration_id cannot be empty", err.Error())
		})

		t.Run("JobFromDifferentIntegration", func(t *testing.T) {
			// Create another test integration
			cfg, err := testutils.LoadTestConfig()
			require.NoError(t, err)

			dbConn2, err := db.NewConnection(cfg.DatabaseURL)
			require.NoError(t, err)
			defer dbConn2.Close()

			usersRepo2 := db.NewPostgresUsersRepository(dbConn2, cfg.DatabaseSchema)
			slackIntegrationsRepo2 := db.NewPostgresSlackIntegrationsRepository(dbConn2, cfg.DatabaseSchema)

			testUser2 := testutils.CreateTestUser(t, usersRepo2)
			testIntegration2 := testutils.CreateTestSlackIntegration(t, slackIntegrationsRepo2, testUser2.ID)
			defer func() { _ = slackIntegrationsRepo2.DeleteSlackIntegrationByID(testIntegration2.ID, testUser2.ID) }()

			slackIntegrationID2 := testIntegration2.ID.String()

			// Create a job with queued message in the second integration
			job, err := jobsService.CreateJob("other.integration.thread", "C7777777777", "testuser", slackIntegrationID2)
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID, slackIntegrationID2) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C7777777777", "1234567890.999999", "Queued message other integration", slackIntegrationID2, models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Query with original integration ID - should not return the job from other integration
			queuedJobs, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
			require.NoError(t, err)

			// Should not find the job from the other integration
			for _, queuedJob := range queuedJobs {
				assert.NotEqual(t, job.ID, queuedJob.ID, "Job from different integration should not be returned")
			}

			// Query with second integration ID - should return the job
			queuedJobs2, err := jobsService.GetJobsWithQueuedMessages(slackIntegrationID2)
			require.NoError(t, err)

			foundJob := false
			for _, queuedJob := range queuedJobs2 {
				if queuedJob.ID == job.ID {
					foundJob = true
					break
				}
			}
			assert.True(t, foundJob, "Job should be found when querying with correct integration ID")
		})
	})
}
