package discordmessages

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/testutils"
)

func setupTestDiscordMessagesService(
	t *testing.T,
) (*DiscordMessagesService, *models.DiscordIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	processedDiscordMessagesRepo := db.NewPostgresProcessedDiscordMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	discordIntegrationsRepo := db.NewPostgresDiscordIntegrationsRepository(dbConn, cfg.DatabaseSchema)
	organizationsRepo := db.NewPostgresOrganizationsRepository(dbConn, cfg.DatabaseSchema)
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

	// Create organization first
	testOrgID := core.NewID("org")
	organization := &models.Organization{ID: testOrgID}
	err = organizationsRepo.CreateOrganization(context.Background(), organization)
	require.NoError(t, err, "Failed to create test organization")

	// Create user with the same database connection
	testUserID := core.NewID("u")
	testUser, err := usersRepo.CreateUser(context.Background(), "test", testUserID, testOrgID)
	require.NoError(t, err, "Failed to create test user")

	// Create discord integration using the same organization ID
	testIntegration := testutils.CreateTestDiscordIntegration(testUser.OrganizationID)
	err = discordIntegrationsRepo.CreateDiscordIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test discord integration")

	// Create a test job for message operations
	testJob := &models.Job{
		ID:             core.NewID("j"),
		JobType:        models.JobTypeDiscord,
		OrganizationID: testIntegration.OrganizationID,
		DiscordPayload: &models.DiscordJobPayload{
			MessageID:     "test-discord-message-123",
			ThreadID:      "test-discord-thread-456",
			UserID:        "test-discord-user-789",
			IntegrationID: testIntegration.ID,
		},
	}
	err = jobsRepo.CreateJob(context.Background(), testJob)
	require.NoError(t, err, "Failed to create test job")

	service := NewDiscordMessagesService(processedDiscordMessagesRepo)

	cleanup := func() {
		// Clean up test data
		_, _ = discordIntegrationsRepo.DeleteDiscordIntegrationByID(
			context.Background(),
			testIntegration.ID,
			testUser.ID,
		)
		_, _ = jobsRepo.DeleteJob(context.Background(), testJob.ID, testIntegration.OrganizationID)
		dbConn.Close()
	}

	return service, testIntegration, cleanup
}

func TestDiscordMessagesService(t *testing.T) {
	service, testIntegration, cleanup := setupTestDiscordMessagesService(t)
	defer cleanup()

	discordIntegrationID := testIntegration.ID
	organizationID := testIntegration.OrganizationID

	// Create a test job for these tests
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)
	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	testJob := &models.Job{
		ID:             core.NewID("j"),
		JobType:        models.JobTypeDiscord,
		OrganizationID: organizationID,
		DiscordPayload: &models.DiscordJobPayload{
			MessageID:     "test-message-create",
			ThreadID:      "test-thread-create",
			UserID:        "test-user-create",
			IntegrationID: discordIntegrationID,
		},
	}
	err = jobsRepo.CreateJob(context.Background(), testJob)
	require.NoError(t, err)
	defer func() { _, _ = jobsRepo.DeleteJob(context.Background(), testJob.ID, organizationID) }()

	t.Run("CreateProcessedDiscordMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			message, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-123",
				"discord-thread-456",
				"Hello Discord world!",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.NoError(t, err)
			assert.NotEmpty(t, message.ID)
			assert.Equal(t, testJob.ID, message.JobID)
			assert.Equal(t, "discord-msg-123", message.DiscordMessageID)
			assert.Equal(t, "discord-thread-456", message.DiscordThreadID)
			assert.Equal(t, "Hello Discord world!", message.TextContent)
			assert.Equal(t, models.ProcessedDiscordMessageStatusQueued, message.Status)
			assert.Equal(t, discordIntegrationID, message.DiscordIntegrationID)
			assert.Equal(t, organizationID, message.OrganizationID)
			assert.False(t, message.CreatedAt.IsZero())
			assert.False(t, message.UpdatedAt.IsZero())

			// Cleanup
			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					testJob.ID,
					discordIntegrationID,
					organizationID,
				)
			}()
		})

		t.Run("InvalidJobID", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				"invalid-job-id",
				"discord-msg-123",
				"discord-thread-456",
				"Hello Discord world!",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("EmptyDiscordMessageID", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"",
				"discord-thread-456",
				"Hello Discord world!",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "discord_message_id cannot be empty", err.Error())
		})

		t.Run("EmptyDiscordThreadID", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-123",
				"",
				"Hello Discord world!",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "discord_thread_id cannot be empty", err.Error())
		})

		t.Run("EmptyTextContent", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-123",
				"discord-thread-456",
				"",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Equal(t, "text_content cannot be empty", err.Error())
		})

		t.Run("InvalidDiscordIntegrationID", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-123",
				"discord-thread-456",
				"Hello Discord world!",
				"invalid-integration-id",
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)

			require.Error(t, err)
			assert.Contains(t, err.Error(), "discord_integration_id must be a valid ULID")
		})

		t.Run("EmptyStatus", func(t *testing.T) {
			_, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-123",
				"discord-thread-456",
				"Hello Discord world!",
				discordIntegrationID,
				organizationID,
				"",
			)

			require.Error(t, err)
			assert.Equal(t, "status cannot be empty", err.Error())
		})
	})

	t.Run("GetProcessedDiscordMessageByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a message first
			createdMessage, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-get",
				"discord-thread-get",
				"Message to retrieve",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusInProgress,
			)
			require.NoError(t, err)
			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					testJob.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			// Retrieve it by ID
			maybeFetchedMessage, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				createdMessage.ID,
				organizationID,
			)
			require.NoError(t, err)
			require.True(t, maybeFetchedMessage.IsPresent())
			fetchedMessage := maybeFetchedMessage.MustGet()

			assert.Equal(t, createdMessage.ID, fetchedMessage.ID)
			assert.Equal(t, createdMessage.JobID, fetchedMessage.JobID)
			assert.Equal(t, createdMessage.DiscordMessageID, fetchedMessage.DiscordMessageID)
			assert.Equal(t, createdMessage.DiscordThreadID, fetchedMessage.DiscordThreadID)
			assert.Equal(t, createdMessage.TextContent, fetchedMessage.TextContent)
			assert.Equal(t, createdMessage.Status, fetchedMessage.Status)
		})

		t.Run("NotFound", func(t *testing.T) {
			nonExistentID := core.NewID("pdm")
			maybeMessage, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				nonExistentID,
				organizationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage.IsPresent())
		})

		t.Run("InvalidID", func(t *testing.T) {
			_, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				"invalid-id",
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "processed discord message ID must be a valid ULID")
		})
	})

	t.Run("UpdateProcessedDiscordMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a message first
			createdMessage, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-update",
				"discord-thread-update",
				"Message to update",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)
			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					testJob.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			// Update the status
			updatedMessage, err := service.UpdateProcessedDiscordMessage(
				context.Background(),
				createdMessage.ID,
				models.ProcessedDiscordMessageStatusInProgress,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			assert.Equal(t, createdMessage.ID, updatedMessage.ID)
			assert.Equal(t, models.ProcessedDiscordMessageStatusInProgress, updatedMessage.Status)
			assert.True(t, updatedMessage.UpdatedAt.After(createdMessage.UpdatedAt))
		})

		t.Run("InvalidID", func(t *testing.T) {
			_, err := service.UpdateProcessedDiscordMessage(
				context.Background(),
				"invalid-id",
				models.ProcessedDiscordMessageStatusCompleted,
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "processed discord message ID must be a valid ULID")
		})

		t.Run("EmptyStatus", func(t *testing.T) {
			messageID := core.NewID("pdm")
			_, err := service.UpdateProcessedDiscordMessage(
				context.Background(),
				messageID,
				"",
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Equal(t, "status cannot be empty", err.Error())
		})
	})

	t.Run("GetProcessedMessagesByJobIDAndStatus", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create messages with different statuses
			message1, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-filter-1",
				"discord-thread-filter",
				"Queued message 1",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			message2, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-filter-2",
				"discord-thread-filter",
				"Queued message 2",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			message3, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-filter-3",
				"discord-thread-filter",
				"In progress message",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusInProgress,
			)
			require.NoError(t, err)

			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					testJob.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			// Get only queued messages
			queuedMessages, err := service.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				testJob.ID,
				models.ProcessedDiscordMessageStatusQueued,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			require.Len(t, queuedMessages, 2)

			// Verify the messages are the queued ones
			messageIDs := []string{queuedMessages[0].ID, queuedMessages[1].ID}
			assert.Contains(t, messageIDs, message1.ID)
			assert.Contains(t, messageIDs, message2.ID)
			assert.NotContains(t, messageIDs, message3.ID)

			// Get in progress messages
			inProgressMessages, err := service.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				testJob.ID,
				models.ProcessedDiscordMessageStatusInProgress,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			require.Len(t, inProgressMessages, 1)
			assert.Equal(t, message3.ID, inProgressMessages[0].ID)
		})

		t.Run("EmptyResult", func(t *testing.T) {
			// Get messages for a status that doesn't exist
			messages, err := service.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				testJob.ID,
				models.ProcessedDiscordMessageStatusCompleted,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.Empty(t, messages)
		})
	})

	t.Run("GetLatestProcessedMessageForJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple messages with different timestamps
			message1, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-latest-1",
				"discord-thread-latest",
				"First message",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			// Wait a moment to ensure different timestamps
			time.Sleep(10 * time.Millisecond)

			message2, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-latest-2",
				"discord-thread-latest",
				"Latest message",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusInProgress,
			)
			require.NoError(t, err)

			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					testJob.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			// Get the latest message
			maybeLatestMessage, err := service.GetLatestProcessedMessageForJob(
				context.Background(),
				testJob.ID,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			require.True(t, maybeLatestMessage.IsPresent())
			latestMessage := maybeLatestMessage.MustGet()

			// Should be message2 (the latest one)
			assert.Equal(t, message2.ID, latestMessage.ID)
			assert.Equal(t, "Latest message", latestMessage.TextContent)
			assert.True(
				t,
				latestMessage.CreatedAt.After(message1.CreatedAt) || latestMessage.CreatedAt.Equal(message1.CreatedAt),
			)
		})

		t.Run("NoMessages", func(t *testing.T) {
			// Create a new job with no messages
			newTestJob := &models.Job{
				ID:             core.NewID("j"),
				JobType:        models.JobTypeDiscord,
				OrganizationID: organizationID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     "test-message-no-msg",
					ThreadID:      "test-thread-no-msg",
					UserID:        "test-user-no-msg",
					IntegrationID: discordIntegrationID,
				},
			}
			err = jobsRepo.CreateJob(context.Background(), newTestJob)
			require.NoError(t, err)
			defer func() { _, _ = jobsRepo.DeleteJob(context.Background(), newTestJob.ID, organizationID) }()

			maybeMessage, err := service.GetLatestProcessedMessageForJob(
				context.Background(),
				newTestJob.ID,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage.IsPresent())
		})
	})

	t.Run("GetActiveMessageCountForJobs", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create two jobs
			job1 := &models.Job{
				ID:             core.NewID("j"),
				JobType:        models.JobTypeDiscord,
				OrganizationID: organizationID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     "test-message-count-1",
					ThreadID:      "test-thread-count-1",
					UserID:        "test-user-count-1",
					IntegrationID: discordIntegrationID,
				},
			}
			err = jobsRepo.CreateJob(context.Background(), job1)
			require.NoError(t, err)
			defer func() { _, _ = jobsRepo.DeleteJob(context.Background(), job1.ID, organizationID) }()

			job2 := &models.Job{
				ID:             core.NewID("j"),
				JobType:        models.JobTypeDiscord,
				OrganizationID: organizationID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     "test-message-count-2",
					ThreadID:      "test-thread-count-2",
					UserID:        "test-user-count-2",
					IntegrationID: discordIntegrationID,
				},
			}
			err = jobsRepo.CreateJob(context.Background(), job2)
			require.NoError(t, err)
			defer func() { _, _ = jobsRepo.DeleteJob(context.Background(), job2.ID, organizationID) }()

			// Add active messages (QUEUED and IN_PROGRESS)
			_, err = service.CreateProcessedDiscordMessage(
				context.Background(),
				job1.ID,
				"discord-msg-active-1",
				"discord-thread-active-1",
				"Active message 1",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			_, err = service.CreateProcessedDiscordMessage(
				context.Background(),
				job1.ID,
				"discord-msg-active-2",
				"discord-thread-active-1",
				"Active message 2",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusInProgress,
			)
			require.NoError(t, err)

			_, err = service.CreateProcessedDiscordMessage(
				context.Background(),
				job2.ID,
				"discord-msg-active-3",
				"discord-thread-active-2",
				"Active message 3",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			// Add inactive message (COMPLETED)
			_, err = service.CreateProcessedDiscordMessage(
				context.Background(),
				job2.ID,
				"discord-msg-inactive",
				"discord-thread-active-2",
				"Inactive message",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusCompleted,
			)
			require.NoError(t, err)

			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					job1.ID,
					discordIntegrationID,
					organizationID,
				)
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					job2.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			// Count active messages for both jobs
			count, err := service.GetActiveMessageCountForJobs(
				context.Background(),
				[]string{job1.ID, job2.ID},
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.Equal(t, 3, count) // 2 from job1 + 1 from job2 (completed message not counted)
		})

		t.Run("NoActiveMessages", func(t *testing.T) {
			// Create a job with only completed messages
			job := &models.Job{
				ID:             core.NewID("j"),
				JobType:        models.JobTypeDiscord,
				OrganizationID: organizationID,
				DiscordPayload: &models.DiscordJobPayload{
					MessageID:     "test-message-no-active",
					ThreadID:      "test-thread-no-active",
					UserID:        "test-user-no-active",
					IntegrationID: discordIntegrationID,
				},
			}
			err = jobsRepo.CreateJob(context.Background(), job)
			require.NoError(t, err)
			defer func() { _, _ = jobsRepo.DeleteJob(context.Background(), job.ID, organizationID) }()

			_, err = service.CreateProcessedDiscordMessage(
				context.Background(),
				job.ID,
				"discord-msg-completed",
				"discord-thread-completed",
				"Completed message",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusCompleted,
			)
			require.NoError(t, err)

			defer func() {
				service.DeleteProcessedDiscordMessagesByJobID(
					context.Background(),
					job.ID,
					discordIntegrationID,
					organizationID,
				)
			}()

			count, err := service.GetActiveMessageCountForJobs(
				context.Background(),
				[]string{job.ID},
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)
			assert.Equal(t, 0, count)
		})
	})

	t.Run("DeleteProcessedDiscordMessagesByJobID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create messages
			message1, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-delete-1",
				"discord-thread-delete",
				"Message to delete 1",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusQueued,
			)
			require.NoError(t, err)

			message2, err := service.CreateProcessedDiscordMessage(
				context.Background(),
				testJob.ID,
				"discord-msg-delete-2",
				"discord-thread-delete",
				"Message to delete 2",
				discordIntegrationID,
				organizationID,
				models.ProcessedDiscordMessageStatusInProgress,
			)
			require.NoError(t, err)

			// Verify messages exist
			maybeMessage1, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				message1.ID,
				organizationID,
			)
			require.NoError(t, err)
			require.True(t, maybeMessage1.IsPresent())

			maybeMessage2, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				message2.ID,
				organizationID,
			)
			require.NoError(t, err)
			require.True(t, maybeMessage2.IsPresent())

			// Delete all messages for the job
			err = service.DeleteProcessedDiscordMessagesByJobID(
				context.Background(),
				testJob.ID,
				discordIntegrationID,
				organizationID,
			)
			require.NoError(t, err)

			// Verify messages are deleted
			maybeMessage1After, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				message1.ID,
				organizationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage1After.IsPresent())

			maybeMessage2After, err := service.GetProcessedDiscordMessageByID(
				context.Background(),
				message2.ID,
				organizationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage2After.IsPresent())
		})
	})

	// Test validation errors for various methods
	t.Run("ValidationErrors", func(t *testing.T) {
		t.Run("GetProcessedMessagesByJobIDAndStatus_InvalidJobID", func(t *testing.T) {
			_, err := service.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				"invalid-job-id",
				models.ProcessedDiscordMessageStatusQueued,
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("GetLatestProcessedMessageForJob_InvalidJobID", func(t *testing.T) {
			_, err := service.GetLatestProcessedMessageForJob(
				context.Background(),
				"invalid-job-id",
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})

		t.Run("GetActiveMessageCountForJobs_InvalidJobID", func(t *testing.T) {
			_, err := service.GetActiveMessageCountForJobs(
				context.Background(),
				[]string{"valid-job-id", "invalid-job-id"},
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "all job IDs must be valid ULIDs")
		})

		t.Run("DeleteProcessedDiscordMessagesByJobID_InvalidJobID", func(t *testing.T) {
			err := service.DeleteProcessedDiscordMessagesByJobID(
				context.Background(),
				"invalid-job-id",
				discordIntegrationID,
				organizationID,
			)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "job ID must be a valid ULID")
		})
	})
}
