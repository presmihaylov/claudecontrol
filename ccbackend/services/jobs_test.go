package services

import (
	"fmt"
	"testing"
	"time"

	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func setupTestJobsService(t *testing.T) (*JobsService, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	repo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	processedSlackMessagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	service := NewJobsService(repo, processedSlackMessagesRepo)

	cleanup := func() {
		dbConn.Close()
	}

	return service, cleanup
}

func TestJobsService(t *testing.T) {
	service, cleanup := setupTestJobsService(t)
	defer cleanup()

	t.Run("CreateJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			slackThreadTS := "test.thread.123"
			slackChannelID := "C1234567890"

			job, err := service.CreateJob(slackThreadTS, slackChannelID)

			require.NoError(t, err)

			assert.NotEqual(t, uuid.Nil, job.ID)
			assert.Equal(t, slackThreadTS, job.SlackThreadTS)
			assert.Equal(t, slackChannelID, job.SlackChannelID)
			assert.False(t, job.CreatedAt.IsZero())
			assert.False(t, job.UpdatedAt.IsZero())
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.CreateJob("", "C1234567890")

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.CreateJob("test.thread.456", "")

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})
	})

	t.Run("GetJobByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			createdJob, err := service.CreateJob("test.thread.789", "C9876543210")
			require.NoError(t, err)

			// Fetch it by ID
			fetchedJob, err := service.GetJobByID(createdJob.ID)
			require.NoError(t, err)

			assert.Equal(t, createdJob.ID, fetchedJob.ID)
			assert.Equal(t, createdJob.SlackThreadTS, fetchedJob.SlackThreadTS)
			assert.Equal(t, createdJob.SlackChannelID, fetchedJob.SlackChannelID)
		})

		t.Run("NilUUID", func(t *testing.T) {
			_, err := service.GetJobByID(uuid.Nil)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := service.GetJobByID(id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("GetOrCreateJobForSlackThread", func(t *testing.T) {
		t.Run("CreateNew", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("new.thread.%d", time.Now().UnixNano())
			slackChannelID := "C5555555555"

			result, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID)

			require.NoError(t, err)
			assert.NotEqual(t, uuid.Nil, result.Job.ID)
			assert.Equal(t, slackThreadTS, result.Job.SlackThreadTS)
			assert.Equal(t, slackChannelID, result.Job.SlackChannelID)
			assert.Equal(t, models.JobCreationStatusCreated, result.Status)
			
			// Cleanup
			defer func() {
				service.DeleteJob(result.Job.ID)
			}()
		})

		t.Run("GetExisting", func(t *testing.T) {
			// Use unique thread ID to avoid conflicts with previous test runs
			slackThreadTS := fmt.Sprintf("existing.thread.%d", time.Now().UnixNano())
			slackChannelID := "C7777777777"

			// Create job first
			firstResult, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusCreated, firstResult.Status)

			// Get the same job again
			secondResult, err := service.GetOrCreateJobForSlackThread(slackThreadTS, slackChannelID)
			require.NoError(t, err)
			assert.Equal(t, models.JobCreationStatusNA, secondResult.Status)

			// Should be the same job
			assert.Equal(t, firstResult.Job.ID, secondResult.Job.ID)
			assert.Equal(t, firstResult.Job.SlackThreadTS, secondResult.Job.SlackThreadTS)
			assert.Equal(t, firstResult.Job.SlackChannelID, secondResult.Job.SlackChannelID)
			
			// Cleanup
			defer func() {
				service.DeleteJob(firstResult.Job.ID)
			}()
		})

		t.Run("EmptySlackThreadTS", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread("", "C1234567890")

			require.Error(t, err)
			assert.Equal(t, "slack_thread_ts cannot be empty", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := service.GetOrCreateJobForSlackThread("test.thread.999", "")

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})
	})

	t.Run("DeleteJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job first
			job, err := service.CreateJob("delete.test.thread", "C1111111111")
			require.NoError(t, err)

			// Verify job exists
			fetchedJob, err := service.GetJobByID(job.ID)
			require.NoError(t, err)
			assert.Equal(t, job.ID, fetchedJob.ID)

			// Delete the job
			err = service.DeleteJob(job.ID)
			require.NoError(t, err)

			// Verify job no longer exists
			_, err = service.GetJobByID(job.ID)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})

		t.Run("NilUUID", func(t *testing.T) {
			err := service.DeleteJob(uuid.Nil)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			err := service.DeleteJob(id)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})
}

func TestJobsAndAgentsIntegration(t *testing.T) {
	// Setup both services
	jobsService, jobsCleanup := setupTestJobsService(t)
	defer jobsCleanup()

	agentsService, agentsCleanup := setupTestService(t)
	defer agentsCleanup()

	t.Run("JobAssignmentWorkflow", func(t *testing.T) {
		// Create an agent first
		agent, err := agentsService.CreateActiveAgent("test-ws-integration", nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID) }()

		// Create a job
		job, err := jobsService.CreateJob("integration.thread.123", "C1234567890")
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignJobToAgent(agent.ID, job.ID)
		require.NoError(t, err)

		// Verify agent has the job assigned
		updatedAgent, err := agentsService.GetAgentByID(agent.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedAgent.AssignedJobID)
		assert.Equal(t, job.ID, *updatedAgent.AssignedJobID)

		// Verify agent is no longer available
		availableAgents, err := agentsService.GetAvailableAgents()
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
		err = agentsService.UnassignJobFromAgent(agent.ID)
		require.NoError(t, err)

		// Verify agent is available again
		availableAgents, err = agentsService.GetAvailableAgents()
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
		agent1, err := agentsService.CreateActiveAgent("test-ws-multi-1", nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent1.ID) }()

		agent2, err := agentsService.CreateActiveAgent("test-ws-multi-2", nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent2.ID) }()

		// Create multiple jobs
		job1, err := jobsService.CreateJob("multi.thread.1", "C1111111111")
		require.NoError(t, err)

		job2, err := jobsService.CreateJob("multi.thread.2", "C2222222222")
		require.NoError(t, err)

		// Assign different jobs to different agents
		err = agentsService.AssignJobToAgent(agent1.ID, job1.ID)
		require.NoError(t, err)

		err = agentsService.AssignJobToAgent(agent2.ID, job2.ID)
		require.NoError(t, err)

		// Verify both agents have their respective jobs
		updatedAgent1, err := agentsService.GetAgentByID(agent1.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedAgent1.AssignedJobID)
		assert.Equal(t, job1.ID, *updatedAgent1.AssignedJobID)

		updatedAgent2, err := agentsService.GetAgentByID(agent2.ID)
		require.NoError(t, err)
		require.NotNil(t, updatedAgent2.AssignedJobID)
		assert.Equal(t, job2.ID, *updatedAgent2.AssignedJobID)

		// Verify no agents are available
		availableAgents, err := agentsService.GetAvailableAgents()
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
		agent, err := agentsService.CreateActiveAgent("test-ws-job-lookup", nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID) }()

		job, err := jobsService.CreateJob("job.lookup.thread", "C9999999999")
		require.NoError(t, err)

		// Initially no agent should be assigned to this job
		_, err = agentsService.GetAgentByJobID(job.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Assign job to agent
		err = agentsService.AssignJobToAgent(agent.ID, job.ID)
		require.NoError(t, err)

		// Now we should be able to find the agent by job ID
		foundAgent, err := agentsService.GetAgentByJobID(job.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, agent.WSConnectionID, foundAgent.WSConnectionID)
		require.NotNil(t, foundAgent.AssignedJobID)
		assert.Equal(t, job.ID, *foundAgent.AssignedJobID)
	})

	t.Run("GetAgentByWSConnectionID", func(t *testing.T) {
		// Create an agent
		wsConnectionID := "test-ws-connection-lookup"
		agent, err := agentsService.CreateActiveAgent(wsConnectionID, nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID) }()

		// Find agent by WebSocket connection ID
		foundAgent, err := agentsService.GetAgentByWSConnectionID(wsConnectionID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, foundAgent.ID)
		assert.Equal(t, wsConnectionID, foundAgent.WSConnectionID)
		assert.Nil(t, foundAgent.AssignedJobID)

		// Test with non-existent connection ID
		_, err = agentsService.GetAgentByWSConnectionID("non-existent-connection")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test with empty connection ID
		_, err = agentsService.GetAgentByWSConnectionID("")
		require.Error(t, err)
		assert.Equal(t, "ws_connection_id cannot be empty", err.Error())
	})

	t.Run("UpdateJobTimestamp", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("timestamp.test.thread", "C9999999999")
		require.NoError(t, err)

		originalUpdatedAt := job.UpdatedAt

		// Update the job timestamp
		err = jobsService.UpdateJobTimestamp(job.ID)
		require.NoError(t, err)

		// Get the job again to verify timestamp changed
		updatedJob, err := jobsService.GetJobByID(job.ID)
		require.NoError(t, err)
		
		// The updated_at should be later than the original
		assert.True(t, updatedJob.UpdatedAt.After(originalUpdatedAt), "Updated timestamp should be later than original")

		// Test with invalid job ID
		err = jobsService.UpdateJobTimestamp(uuid.Nil)
		require.Error(t, err)
		assert.Equal(t, "job ID cannot be nil", err.Error())
	})

	t.Run("GetIdleJobs", func(t *testing.T) {
		t.Run("JobWithNoMessages", func(t *testing.T) {
			// Create a job with no messages
			job, err := jobsService.CreateJob("idle.no.messages", "C1111111111")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			// Since we just created the job, it shouldn't be idle
			idleJobs, err := jobsService.GetIdleJobs(1)
			require.NoError(t, err)
			
			// Filter out our test job - it should not be in idle list
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Newly created job should not be in idle list")

			// Now manipulate the job timestamp to make it old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateJobUpdatedAt(job.ID, oldTimestamp)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(5)
			require.NoError(t, err)
			
			assert.True(t, jobFoundInIdleList(job.ID, idleJobs), "Job with old updated_at and no messages should be idle")
		})

		t.Run("JobWithIncompleteMessages", func(t *testing.T) {
			// Create a job and add a message that's not completed
			job, err := jobsService.CreateJob("idle.incomplete.messages", "C2222222222")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			// Add a message in IN_PROGRESS state
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C2222222222", "1234567890.111111", "Hello world", models.ProcessedSlackMessageStatusInProgress)
			require.NoError(t, err)

			// Job should not be idle because it has an incomplete message
			idleJobs, err := jobsService.GetIdleJobs(999) // Even with very high threshold
			require.NoError(t, err)
			
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with incomplete messages should not be idle")
		})

		t.Run("JobWithQueuedMessages", func(t *testing.T) {
			// Create a job and add a queued message
			job, err := jobsService.CreateJob("idle.queued.messages", "C3333333333")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			// Add a message in QUEUED state
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C3333333333", "1234567890.222222", "Hello queued", models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Job should not be idle because it has a queued message
			idleJobs, err := jobsService.GetIdleJobs(999) // Even with very high threshold
			require.NoError(t, err)
			
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with queued messages should not be idle")
		})

		t.Run("JobWithOnlyCompletedMessages", func(t *testing.T) {
			// Create a job and add only completed messages
			job, err := jobsService.CreateJob("idle.completed.messages", "C4444444444")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			// Add a completed message
			message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C4444444444", "1234567890.333333", "Hello completed", models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Since the message was just created, job should not be idle with 1 minute threshold
			idleJobs, err := jobsService.GetIdleJobs(1)
			require.NoError(t, err)
			
			assert.False(t, jobFoundInIdleList(job.ID, idleJobs), "Job with recently completed messages should not be idle")

			// Now manipulate the timestamp to make the message old
			oldTimestamp := time.Now().Add(-10 * time.Minute) // 10 minutes ago
			err = jobsService.TESTS_UpdateProcessedSlackMessageUpdatedAt(message.ID, oldTimestamp)
			require.NoError(t, err)

			// Now the job should be idle with 5 minute threshold
			idleJobs, err = jobsService.GetIdleJobs(5)
			require.NoError(t, err)
			
			assert.True(t, jobFoundInIdleList(job.ID, idleJobs), "Job with old completed messages should be idle")
		})

		t.Run("JobWithMixedMessages", func(t *testing.T) {
			// Create a job with both completed and incomplete messages
			job, err := jobsService.CreateJob("idle.mixed.messages", "C5555555555")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			// Add a completed message
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C5555555555", "1234567890.444444", "Hello completed", models.ProcessedSlackMessageStatusCompleted)
			require.NoError(t, err)

			// Add an incomplete message
			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C5555555555", "1234567890.555555", "Hello in progress", models.ProcessedSlackMessageStatusInProgress)
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
			job, err := jobsService.CreateJob("test.thread.processed", "C1234567890")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			slackChannelID := "C1234567890"
			slackTS := "1234567890.123456"
			textContent := "Hello world"
			status := models.ProcessedSlackMessageStatusQueued

			message, err := jobsService.CreateProcessedSlackMessage(job.ID, slackChannelID, slackTS, textContent, status)

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
			_, err := jobsService.CreateProcessedSlackMessage(uuid.Nil, "C1234567890", "1234567890.123456", "Hello world", models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "job ID cannot be nil", err.Error())
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.channel", "C1234567890")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "", "1234567890.123456", "Hello world", models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "slack_channel_id cannot be empty", err.Error())
		})

		t.Run("EmptySlackTS", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.ts", "C1234567890")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "", "Hello world", models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "slack_ts cannot be empty", err.Error())
		})

		t.Run("EmptyTextContent", func(t *testing.T) {
			job, err := jobsService.CreateJob("test.thread.empty.text", "C1234567890")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			_, err = jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "1234567890.123456", "", models.ProcessedSlackMessageStatusQueued)

			require.Error(t, err)
			assert.Equal(t, "text_content cannot be empty", err.Error())
		})
	})

	t.Run("UpdateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a job and processed slack message first
			job, err := jobsService.CreateJob("test.thread.update", "C1234567890")
			require.NoError(t, err)
			defer func() { _ = jobsService.DeleteJob(job.ID) }()

			message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C1234567890", "1234567890.123456", "Hello world", models.ProcessedSlackMessageStatusQueued)
			require.NoError(t, err)

			// Update the status
			newStatus := models.ProcessedSlackMessageStatusInProgress
			updatedMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, newStatus)
			require.NoError(t, err)
			assert.Equal(t, newStatus, updatedMessage.Status)
			assert.True(t, updatedMessage.UpdatedAt.After(message.UpdatedAt))
		})

		t.Run("NilID", func(t *testing.T) {
			_, err := jobsService.UpdateProcessedSlackMessage(uuid.Nil, models.ProcessedSlackMessageStatusCompleted)

			require.Error(t, err)
			assert.Equal(t, "processed slack message ID cannot be nil", err.Error())
		})

		t.Run("NotFound", func(t *testing.T) {
			id := uuid.New()

			_, err := jobsService.UpdateProcessedSlackMessage(id, models.ProcessedSlackMessageStatusCompleted)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not found")
		})
	})

	t.Run("DeleteJobWithAgentAssignment", func(t *testing.T) {
		// Create an agent and job
		agent, err := agentsService.CreateActiveAgent("test-ws-delete-job", nil)
		require.NoError(t, err)
		defer func() { _ = agentsService.DeleteActiveAgent(agent.ID) }()

		job, err := jobsService.CreateJob("delete.assigned.thread", "C8888888888")
		require.NoError(t, err)

		// Assign job to agent
		err = agentsService.AssignJobToAgent(agent.ID, job.ID)
		require.NoError(t, err)

		// Verify assignment
		assignedAgent, err := agentsService.GetAgentByJobID(job.ID)
		require.NoError(t, err)
		assert.Equal(t, agent.ID, assignedAgent.ID)

		// Unassign agent (simulating cleanup process)
		err = agentsService.UnassignJobFromAgent(agent.ID)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(job.ID)
		require.NoError(t, err)

		// Verify job is deleted
		_, err = jobsService.GetJobByID(job.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify agent still exists but has no job assigned
		remainingAgent, err := agentsService.GetAgentByID(agent.ID)
		require.NoError(t, err)
		assert.Nil(t, remainingAgent.AssignedJobID)
	})

	t.Run("DeleteJobCascadesProcessedSlackMessages", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("cascade.delete.thread", "C9999999999")
		require.NoError(t, err)

		// Create multiple processed slack messages for this job
		message1, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.111111", "Hello world 1", models.ProcessedSlackMessageStatusQueued)
		require.NoError(t, err)

		message2, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.222222", "Hello world 2", models.ProcessedSlackMessageStatusInProgress)
		require.NoError(t, err)

		message3, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.333333", "Hello world 3", models.ProcessedSlackMessageStatusCompleted)
		require.NoError(t, err)

		// Verify all messages exist
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message1.ID)
		require.NoError(t, err)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message2.ID)
		require.NoError(t, err)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message3.ID)
		require.NoError(t, err)

		// Delete the job
		err = jobsService.DeleteJob(job.ID)
		require.NoError(t, err)

		// Verify job is deleted
		_, err = jobsService.GetJobByID(job.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Verify all processed slack messages are also deleted (cascade)
		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message1.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message2.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		_, err = jobsService.processedSlackMessagesRepo.GetProcessedSlackMessageByID(message3.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("ProcessedSlackMessageStatusTransitions", func(t *testing.T) {
		// Create a job
		job, err := jobsService.CreateJob("status.transition.thread", "C9999999999")
		require.NoError(t, err)
		defer func() { _ = jobsService.DeleteJob(job.ID) }()

		// Create a processed slack message
		message, err := jobsService.CreateProcessedSlackMessage(job.ID, "C9999999999", "1234567890.444444", "Hello world transition", models.ProcessedSlackMessageStatusQueued)
		require.NoError(t, err)

		// Test status transitions: QUEUED -> IN_PROGRESS -> COMPLETED
		updatedMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, models.ProcessedSlackMessageStatusInProgress)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusInProgress, updatedMessage.Status)

		finalMessage, err := jobsService.UpdateProcessedSlackMessage(message.ID, models.ProcessedSlackMessageStatusCompleted)
		require.NoError(t, err)
		assert.Equal(t, models.ProcessedSlackMessageStatusCompleted, finalMessage.Status)
		assert.True(t, finalMessage.UpdatedAt.After(updatedMessage.UpdatedAt))
	})
}