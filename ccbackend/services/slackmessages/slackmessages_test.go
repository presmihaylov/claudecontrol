package slackmessages

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

func TestSlackMessagesService(t *testing.T) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err)
	defer dbConn.Close()

	// Create repositories
	processedSlackMessagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create service
	slackMessagesService := NewSlackMessagesService(processedSlackMessagesRepo)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrgID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	// Test organization and integration data
	orgID := testIntegration.OrgID
	slackIntegrationID := testIntegration.ID

	// Create a test job
	job := &models.Job{
		ID:      core.NewID("j"),
		JobType: models.JobTypeSlack,
		SlackPayload: &models.SlackJobPayload{
			ThreadTS:      "test.thread.123",
			ChannelID:     "C1234567",
			UserID:        "U12345",
			IntegrationID: slackIntegrationID,
		},
		OrgID: orgID,
	}
	err = jobsRepo.CreateJob(context.Background(), job)
	require.NoError(t, err, "Failed to create test job")
	jobID := job.ID

	// Cleanup function
	defer func() {
		_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
			context.Background(),
			jobID,
			slackIntegrationID,
			orgID,
		)
		_, _ = slackIntegrationsRepo.DeleteSlackIntegrationByID(
			context.Background(),
			testUser.OrgID,
			testIntegration.ID,
		)
	}()

	t.Run("CreateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			slackChannelID := "C1234567"
			slackTS := "1234567890.123456"
			textContent := "Hello, world!"
			status := models.ProcessedSlackMessageStatusQueued

			message, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				slackChannelID,
				slackTS,
				textContent,
				slackIntegrationID,
				status,
			)
			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			require.NoError(t, err)
			assert.NotEmpty(t, message.ID)
			assert.Equal(t, jobID, message.JobID)
			assert.Equal(t, slackChannelID, message.SlackChannelID)
			assert.Equal(t, slackTS, message.SlackTS)
			assert.Equal(t, textContent, message.TextContent)
			assert.Equal(t, status, message.Status)
			assert.Equal(t, slackIntegrationID, message.SlackIntegrationID)
			assert.Equal(t, orgID, message.OrgID)
		})

		t.Run("EmptySlackChannelID", func(t *testing.T) {
			_, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"",
				"1234567890.123456",
				"Hello",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "slack_channel_id cannot be empty")
		})

		t.Run("EmptySlackTS", func(t *testing.T) {
			_, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"",
				"Hello",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "slack_ts cannot be empty")
		})

		t.Run("EmptyTextContent", func(t *testing.T) {
			_, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "text_content cannot be empty")
		})
	})

	t.Run("UpdateProcessedSlackMessage", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a processed slack message first
			message, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"Hello, world!",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)
			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Update the status
			newStatus := models.ProcessedSlackMessageStatusInProgress
			updatedMessage, err := slackMessagesService.UpdateProcessedSlackMessage(
				context.Background(),
				orgID,
				message.ID,
				newStatus,
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.Equal(t, newStatus, updatedMessage.Status)
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("psm")
			_, err := slackMessagesService.UpdateProcessedSlackMessage(
				context.Background(),
				orgID,
				id,
				models.ProcessedSlackMessageStatusInProgress,
				slackIntegrationID,
			)
			assert.Error(t, err)
			assert.Equal(t, core.ErrNotFound, err)
		})
	})

	t.Run("GetProcessedSlackMessageByID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a processed slack message first
			message, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"Hello, world!",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)
			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Get the message by ID
			maybeMessage, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				message.ID,
			)
			require.NoError(t, err)
			assert.True(t, maybeMessage.IsPresent())
			retrievedMessage := maybeMessage.MustGet()
			assert.Equal(t, message.ID, retrievedMessage.ID)
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("psm")
			maybeMessage, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				id,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage.IsPresent())
		})
	})

	t.Run("GetProcessedMessagesByJobIDAndStatus", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple messages with different statuses
			message1, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"Message 1",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			message2, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123457",
				"Message 2",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Get queued messages
			queuedMessages, err := slackMessagesService.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				orgID,
				jobID,
				models.ProcessedSlackMessageStatusQueued,
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.Len(t, queuedMessages, 1)
			assert.Equal(t, message1.ID, queuedMessages[0].ID)

			// Get in-progress messages
			inProgressMessages, err := slackMessagesService.GetProcessedMessagesByJobIDAndStatus(
				context.Background(),
				orgID,
				jobID,
				models.ProcessedSlackMessageStatusInProgress,
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.Len(t, inProgressMessages, 1)
			assert.Equal(t, message2.ID, inProgressMessages[0].ID)
		})
	})

	t.Run("GetLatestProcessedMessageForJob", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple messages
			_, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"First message",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			time.Sleep(10 * time.Millisecond) // Ensure different timestamps

			latestMessage, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123457",
				"Latest message",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Get the latest message
			maybeLatest, err := slackMessagesService.GetLatestProcessedMessageForJob(
				context.Background(),
				orgID,
				jobID,
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.True(t, maybeLatest.IsPresent())
			retrievedMessage := maybeLatest.MustGet()
			assert.Equal(t, latestMessage.ID, retrievedMessage.ID)
			assert.Equal(t, "Latest message", retrievedMessage.TextContent)
		})

		t.Run("NoMessages", func(t *testing.T) {
			noMessagesJobID := core.NewID("j")
			maybeLatest, err := slackMessagesService.GetLatestProcessedMessageForJob(
				context.Background(),
				orgID,
				noMessagesJobID,
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.False(t, maybeLatest.IsPresent())
		})
	})

	t.Run("DeleteProcessedSlackMessagesByJobID", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create multiple messages for the job
			message1, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"Message 1",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			message2, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123457",
				"Message 2",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			// Verify messages exist
			maybeMessage1, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				message1.ID,
			)
			require.NoError(t, err)
			assert.True(t, maybeMessage1.IsPresent())

			maybeMessage2, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				message2.ID,
			)
			require.NoError(t, err)
			assert.True(t, maybeMessage2.IsPresent())

			// Delete all messages for the job
			err = slackMessagesService.DeleteProcessedSlackMessagesByJobID(
				context.Background(),
				orgID,
				jobID,
				slackIntegrationID,
			)
			require.NoError(t, err)

			// Verify messages are deleted
			maybeMessage1After, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				message1.ID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage1After.IsPresent())

			maybeMessage2After, err := slackMessagesService.GetProcessedSlackMessageByID(
				context.Background(),
				orgID,
				message2.ID,
			)
			require.NoError(t, err)
			assert.False(t, maybeMessage2After.IsPresent())
		})
	})

	t.Run("GetActiveMessageCountForJobs", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create additional jobs for this test
			job1 := &models.Job{
				ID:      core.NewID("j"),
				JobType: models.JobTypeSlack,
				SlackPayload: &models.SlackJobPayload{
					ThreadTS:      "test.thread.456",
					ChannelID:     "C2345678",
					UserID:        "U23456",
					IntegrationID: slackIntegrationID,
				},
				OrgID: orgID,
			}
			err := jobsRepo.CreateJob(context.Background(), job1)
			require.NoError(t, err, "Failed to create test job1")
			job1ID := job1.ID

			job2 := &models.Job{
				ID:      core.NewID("j"),
				JobType: models.JobTypeSlack,
				SlackPayload: &models.SlackJobPayload{
					ThreadTS:      "test.thread.789",
					ChannelID:     "C3456789",
					UserID:        "U34567",
					IntegrationID: slackIntegrationID,
				},
				OrgID: orgID,
			}
			err = jobsRepo.CreateJob(context.Background(), job2)
			require.NoError(t, err, "Failed to create test job2")
			job2ID := job2.ID

			// Create messages with different statuses
			// Job 1: 2 active messages (queued + in progress)
			_, err = slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				job1ID,
				"C1234567",
				"1234567890.123456",
				"Job1 Message 1",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			_, err = slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				job1ID,
				"C1234567",
				"1234567890.123457",
				"Job1 Message 2",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusInProgress,
			)
			require.NoError(t, err)

			// Job 2: 1 active message (queued) and 1 completed (not active)
			_, err = slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				job2ID,
				"C1234567",
				"1234567890.123458",
				"Job2 Message 1",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusQueued,
			)
			require.NoError(t, err)

			_, err = slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				job2ID,
				"C1234567",
				"1234567890.123459",
				"Job2 Message 2",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)

			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					job1ID,
					slackIntegrationID,
					orgID,
				)
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					job2ID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Count active messages for both jobs
			count, err := slackMessagesService.GetActiveMessageCountForJobs(
				context.Background(),
				orgID,
				[]string{job1ID, job2ID},
				slackIntegrationID,
			)
			require.NoError(t, err)
			assert.Equal(t, 3, count) // 2 from job1 + 1 from job2
		})
	})

	t.Run("TESTS_UpdateProcessedSlackMessageUpdatedAt", func(t *testing.T) {
		t.Run("Success", func(t *testing.T) {
			// Create a processed slack message first
			message, err := slackMessagesService.CreateProcessedSlackMessage(
				context.Background(),
				orgID,
				jobID,
				"C1234567",
				"1234567890.123456",
				"Test message",
				slackIntegrationID,
				models.ProcessedSlackMessageStatusCompleted,
			)
			require.NoError(t, err)
			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(),
					jobID,
					slackIntegrationID,
					orgID,
				)
			}()

			// Update the timestamp
			newTimestamp := time.Now().Add(-1 * time.Hour)
			err = slackMessagesService.TESTS_UpdateProcessedSlackMessageUpdatedAt(
				context.Background(),
				orgID,
				message.ID,
				newTimestamp,
				slackIntegrationID,
			)
			require.NoError(t, err)
		})

		t.Run("NotFound", func(t *testing.T) {
			id := core.NewID("psm")
			newTimestamp := time.Now().Add(-1 * time.Hour)
			err := slackMessagesService.TESTS_UpdateProcessedSlackMessageUpdatedAt(
				context.Background(),
				orgID,
				id,
				newTimestamp,
				slackIntegrationID,
			)
			assert.Error(t, err)
			assert.Equal(t, core.ErrNotFound, err)
		})
	})

	t.Run("GetProcessedMessagesByStatus", func(t *testing.T) {
		t.Run("Success_MessagesFound", func(t *testing.T) {
			// Create messages with different statuses
			queuedMessage := testutils.CreateTestProcessedSlackMessage(
				job.ID, orgID, slackIntegrationID, models.ProcessedSlackMessageStatusQueued,
			)
			inProgressMessage := testutils.CreateTestProcessedSlackMessage(
				job.ID, orgID, slackIntegrationID, models.ProcessedSlackMessageStatusInProgress,
			)

			// Create the messages in database
			defer func() {
				_ = processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(
					context.Background(), job.ID, slackIntegrationID, orgID,
				)
			}()

			err = processedSlackMessagesRepo.CreateProcessedSlackMessage(context.Background(), queuedMessage)
			require.NoError(t, err)
			err = processedSlackMessagesRepo.CreateProcessedSlackMessage(context.Background(), inProgressMessage)
			require.NoError(t, err)

			// Test getting queued messages
			messages, err := slackMessagesService.GetProcessedMessagesByStatus(
				context.Background(),
				orgID,
				models.ProcessedSlackMessageStatusQueued,
				slackIntegrationID,
			)

			require.NoError(t, err)
			require.Len(t, messages, 1)
			assert.Equal(t, queuedMessage.ID, messages[0].ID)
			assert.Equal(t, models.ProcessedSlackMessageStatusQueued, messages[0].Status)

			// Test getting in-progress messages
			messages, err = slackMessagesService.GetProcessedMessagesByStatus(
				context.Background(),
				orgID,
				models.ProcessedSlackMessageStatusInProgress,
				slackIntegrationID,
			)

			require.NoError(t, err)
			require.Len(t, messages, 1)
			assert.Equal(t, inProgressMessage.ID, messages[0].ID)
			assert.Equal(t, models.ProcessedSlackMessageStatusInProgress, messages[0].Status)
		})

		t.Run("Success_NoMessagesFound", func(t *testing.T) {
			// Test getting messages of a status that doesn't exist
			messages, err := slackMessagesService.GetProcessedMessagesByStatus(
				context.Background(),
				orgID,
				models.ProcessedSlackMessageStatusCompleted,
				slackIntegrationID,
			)

			require.NoError(t, err)
			assert.Empty(t, messages)
		})
	})
}
