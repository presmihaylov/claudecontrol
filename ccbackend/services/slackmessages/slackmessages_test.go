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

func setupSlackMessagesServiceTest(t *testing.T) (*SlackMessagesService, *models.SlackIntegration, func()) {
	cfg, err := testutils.LoadTestConfig()
	require.NoError(t, err)

	dbConn, err := db.NewConnection(cfg.DatabaseURL)
	require.NoError(t, err, "Failed to create database connection")

	// Create repositories
	messagesRepo := db.NewPostgresProcessedSlackMessagesRepository(dbConn, cfg.DatabaseSchema)
	usersRepo := db.NewPostgresUsersRepository(dbConn, cfg.DatabaseSchema)
	slackIntegrationsRepo := db.NewPostgresSlackIntegrationsRepository(dbConn, cfg.DatabaseSchema)

	// Create test user and slack integration
	testUser := testutils.CreateTestUser(t, usersRepo)
	testIntegration := testutils.CreateTestSlackIntegration(testUser.OrganizationID)
	err = slackIntegrationsRepo.CreateSlackIntegration(context.Background(), testIntegration)
	require.NoError(t, err, "Failed to create test slack integration")

	// Create service
	service := NewSlackMessagesService(messagesRepo)

	cleanup := func() {
		dbConn.Close()
	}

	return service, testIntegration, cleanup
}

func TestSlackMessagesService_CreateProcessedSlackMessage(t *testing.T) {
	service, testIntegration, cleanup := setupSlackMessagesServiceTest(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first (required for the message)
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.processed",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		slackChannelID := "C1234567890"
		slackTS := "1234567890.123456"
		textContent := "Hello world"
		status := models.ProcessedSlackMessageStatusQueued

		message, err := service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			slackChannelID,
			slackTS,
			textContent,
			testIntegration.ID,
			testIntegration.OrganizationID,
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

	t.Run("InvalidJobID", func(t *testing.T) {
		_, err := service.CreateProcessedSlackMessage(
			context.Background(),
			"invalid-job-id",
			"C1234567890",
			"1234567890.123456",
			"Hello world",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "job ID must be a valid ULID")
	})

	t.Run("EmptySlackChannelID", func(t *testing.T) {
		jobID := core.NewID("j")
		_, err := service.CreateProcessedSlackMessage(
			context.Background(),
			jobID,
			"",
			"1234567890.123456",
			"Hello world",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slack_channel_id cannot be empty")
	})

	t.Run("EmptySlackTS", func(t *testing.T) {
		jobID := core.NewID("j")
		_, err := service.CreateProcessedSlackMessage(
			context.Background(),
			jobID,
			"C1234567890",
			"",
			"Hello world",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "slack_ts cannot be empty")
	})

	t.Run("EmptyTextContent", func(t *testing.T) {
		jobID := core.NewID("j")
		_, err := service.CreateProcessedSlackMessage(
			context.Background(),
			jobID,
			"C1234567890",
			"1234567890.123456",
			"",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "text_content cannot be empty")
	})
}

func TestSlackMessagesService_UpdateProcessedSlackMessage(t *testing.T) {
	service, testIntegration, cleanup := setupSlackMessagesServiceTest(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.update",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		message, err := service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123456",
			"Hello world",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		updatedMessage, err := service.UpdateProcessedSlackMessage(
			context.Background(),
			message.ID,
			models.ProcessedSlackMessageStatusInProgress,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		assert.Equal(t, message.ID, updatedMessage.ID)
		assert.Equal(t, models.ProcessedSlackMessageStatusInProgress, updatedMessage.Status)
		assert.True(t, updatedMessage.UpdatedAt.After(message.UpdatedAt))
	})

	t.Run("InvalidID", func(t *testing.T) {
		_, err := service.UpdateProcessedSlackMessage(
			context.Background(),
			"invalid-id",
			models.ProcessedSlackMessageStatusInProgress,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "processed slack message ID must be a valid ULID")
	})
}

func TestSlackMessagesService_GetProcessedSlackMessageByID(t *testing.T) {
	service, testIntegration, cleanup := setupSlackMessagesServiceTest(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.get",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		message, err := service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123456",
			"Hello world",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		maybeMessage, err := service.GetProcessedSlackMessageByID(
			context.Background(),
			message.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		require.True(t, maybeMessage.IsPresent())
		retrievedMessage := maybeMessage.MustGet()
		assert.Equal(t, message.ID, retrievedMessage.ID)
		assert.Equal(t, message.TextContent, retrievedMessage.TextContent)
	})

	t.Run("NotFound", func(t *testing.T) {
		nonExistentID := core.NewID("psm")
		maybeMessage, err := service.GetProcessedSlackMessageByID(
			context.Background(),
			nonExistentID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		assert.False(t, maybeMessage.IsPresent())
	})

	t.Run("InvalidID", func(t *testing.T) {
		_, err := service.GetProcessedSlackMessageByID(
			context.Background(),
			"invalid-id",
			testIntegration.OrganizationID,
		)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "processed slack message ID must be a valid ULID")
	})
}

func TestSlackMessagesService_GetProcessedMessagesByJobIDAndStatus(t *testing.T) {
	service, testIntegration, cleanup := setupSlackMessagesServiceTest(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.list",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		// Create multiple messages with different statuses
		_, err = service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123456",
			"Message 1",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		_, err = service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123457",
			"Message 2",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusInProgress,
		)
		require.NoError(t, err)

		_, err = service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123458",
			"Message 3",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		// Get queued messages
		queuedMessages, err := service.GetProcessedMessagesByJobIDAndStatus(
			context.Background(),
			job.ID,
			models.ProcessedSlackMessageStatusQueued,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		assert.Len(t, queuedMessages, 2)

		// Get in-progress messages
		inProgressMessages, err := service.GetProcessedMessagesByJobIDAndStatus(
			context.Background(),
			job.ID,
			models.ProcessedSlackMessageStatusInProgress,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		assert.Len(t, inProgressMessages, 1)
	})
}

func TestSlackMessagesService_GetLatestProcessedMessageForJob(t *testing.T) {
	service, testIntegration, cleanup := setupSlackMessagesServiceTest(t)
	defer cleanup()

	t.Run("Success", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.latest",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		// Create first message
		message1, err := service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123456",
			"First message",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusQueued,
		)
		require.NoError(t, err)

		// Wait a moment to ensure different timestamps
		time.Sleep(10 * time.Millisecond)

		// Create second message (should be latest)
		message2, err := service.CreateProcessedSlackMessage(
			context.Background(),
			job.ID,
			"C1234567890",
			"1234567890.123457",
			"Second message",
			testIntegration.ID,
			testIntegration.OrganizationID,
			models.ProcessedSlackMessageStatusInProgress,
		)
		require.NoError(t, err)

		// Get latest message
		maybeLatest, err := service.GetLatestProcessedMessageForJob(
			context.Background(),
			job.ID,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		require.True(t, maybeLatest.IsPresent())
		latest := maybeLatest.MustGet()
		assert.Equal(t, message2.ID, latest.ID)
		assert.Equal(t, "Second message", latest.TextContent)
		assert.True(t, latest.CreatedAt.After(message1.CreatedAt) || latest.CreatedAt.Equal(message1.CreatedAt))
	})

	t.Run("NoMessages", func(t *testing.T) {
		// Create service with repos for job creation
		cfg, err := testutils.LoadTestConfig()
		require.NoError(t, err)
		dbConn, err := db.NewConnection(cfg.DatabaseURL)
		require.NoError(t, err)
		defer dbConn.Close()

		jobsRepo := db.NewPostgresJobsRepository(dbConn, cfg.DatabaseSchema)

		// Create a job first
		job := &models.Job{
			ID:             core.NewID("j"),
			JobType:        models.JobTypeSlack,
			OrganizationID: testIntegration.OrganizationID,
			SlackPayload: &models.SlackJobPayload{
				ThreadTS:      "test.thread.empty",
				ChannelID:     "C1234567890",
				UserID:        core.NewID("u"),
				IntegrationID: testIntegration.ID,
			},
		}
		err = jobsRepo.CreateJob(context.Background(), job)
		require.NoError(t, err)
		defer func() {
			_, _ = jobsRepo.DeleteJob(context.Background(), job.ID, testIntegration.ID, testIntegration.OrganizationID)
		}()

		// Get latest message (should be None)
		maybeLatest, err := service.GetLatestProcessedMessageForJob(
			context.Background(),
			job.ID,
			testIntegration.ID,
			testIntegration.OrganizationID,
		)

		require.NoError(t, err)
		assert.False(t, maybeLatest.IsPresent())
	})
}
