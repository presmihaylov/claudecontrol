package jobs

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
	"ccbackend/services"
)

type JobsService struct {
	jobsRepo                   *db.PostgresJobsRepository
	processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository
	txManager                  services.TransactionManager
}

func NewJobsService(
	repo *db.PostgresJobsRepository,
	processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository,
	txManager services.TransactionManager,
) *JobsService {
	return &JobsService{
		jobsRepo:                   repo,
		processedSlackMessagesRepo: processedSlackMessagesRepo,
		txManager:                  txManager,
	}
}

func (s *JobsService) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	slackIntegrationID string,
	organizationID string,
) (int, error) {
	log.Printf("📋 Starting to get active message count for %d jobs", len(jobIDs))
	if !core.IsValidULID(organizationID) {
		return 0, fmt.Errorf("organization_id must be a valid ULID")
	}
	count, err := s.processedSlackMessagesRepo.GetActiveMessageCountForJobs(
		ctx,
		jobIDs,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d active messages", count)
	return count, nil
}

func (s *JobsService) CreateJob(
	ctx context.Context,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
	organizationID string,
) (*models.Job, error) {
	log.Printf(
		"📋 Starting to create job for slack thread: %s, channel: %s, user: %s, organization: %s",
		slackThreadTS,
		slackChannelID,
		slackUserID,
		organizationID,
	)

	if slackThreadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}
	if slackChannelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}
	if slackUserID == "" {
		return nil, fmt.Errorf("slack_user_id cannot be empty")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	job := &models.Job{
		ID:                 core.NewID("j"),
		SlackThreadTS:      slackThreadTS,
		SlackChannelID:     slackChannelID,
		SlackUserID:        slackUserID,
		SlackIntegrationID: slackIntegrationID,
		OrganizationID:     organizationID,
	}

	if err := s.jobsRepo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("📋 Completed successfully - created job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByID(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.Job](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return mo.None[*models.Job](), fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobByID(ctx, id, slackIntegrationID, organizationID)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("📋 Completed successfully - job not found")
		return mo.None[*models.Job](), nil
	}
	job := maybeJob.MustGet()

	log.Printf("📋 Completed successfully - retrieved job with ID: %s", job.ID)
	return mo.Some(job), nil
}

func (s *JobsService) GetJobBySlackThread(
	ctx context.Context,
	threadTS, channelID, slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by slack thread: %s, channel: %s", threadTS, channelID)
	if threadTS == "" {
		return mo.None[*models.Job](), fmt.Errorf("slack_thread_ts cannot be empty")
	}
	if channelID == "" {
		return mo.None[*models.Job](), fmt.Errorf("slack_channel_id cannot be empty")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return mo.None[*models.Job](), fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobBySlackThread(ctx, threadTS, channelID, slackIntegrationID, organizationID)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to get job by slack thread: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("📋 Completed successfully - job not found")
		return mo.None[*models.Job](), nil
	}
	job := maybeJob.MustGet()

	log.Printf("📋 Completed successfully - retrieved job with ID: %s", job.ID)
	return mo.Some(job), nil
}

func (s *JobsService) GetOrCreateJobForSlackThread(
	ctx context.Context,
	threadTS, channelID, slackUserID, slackIntegrationID string,
	organizationID string,
) (*models.JobCreationResult, error) {
	log.Printf(
		"📋 Starting to get or create job for slack thread: %s, channel: %s, user: %s, organization: %s",
		threadTS,
		channelID,
		slackUserID,
		organizationID,
	)

	if threadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}
	if channelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}
	if slackUserID == "" {
		return nil, fmt.Errorf("slack_user_id cannot be empty")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Try to find existing job first
	maybeExistingJob, err := s.jobsRepo.GetJobBySlackThread(
		ctx,
		threadTS,
		channelID,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get job by slack thread: %w", err)
	}

	if maybeExistingJob.IsPresent() {
		existingJob := maybeExistingJob.MustGet()
		log.Printf("📋 Completed successfully - found existing job with ID: %s", existingJob.ID)
		return &models.JobCreationResult{
			Job:    existingJob,
			Status: models.JobCreationStatusNA,
		}, nil
	}

	// If not found, create a new job
	newJob, createErr := s.CreateJob(ctx, threadTS, channelID, slackUserID, slackIntegrationID, organizationID)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create new job: %w", createErr)
	}
	log.Printf("📋 Completed successfully - created new job with ID: %s", newJob.ID)
	return &models.JobCreationResult{
		Job:    newJob,
		Status: models.JobCreationStatusCreated,
	}, nil
}

func (s *JobsService) UpdateJobTimestamp(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to update job timestamp for ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}
	if err := s.jobsRepo.UpdateJobTimestamp(ctx, jobID, slackIntegrationID, organizationID); err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - updated timestamp for job ID: %s", jobID)
	return nil
}

func (s *JobsService) GetIdleJobs(ctx context.Context, idleMinutes int, organizationID string) ([]*models.Job, error) {
	log.Printf("📋 Starting to get idle jobs older than %d minutes for organization: %s", idleMinutes, organizationID)
	if idleMinutes <= 0 {
		return nil, fmt.Errorf("idle minutes must be greater than 0")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	jobs, err := s.jobsRepo.GetIdleJobs(ctx, idleMinutes, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle jobs: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d idle jobs", len(jobs))
	return jobs, nil
}

func (s *JobsService) DeleteJob(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to delete job with ID: %s", id)
	if !core.IsValidULID(id) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		if err := s.processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(ctx, id, slackIntegrationID, organizationID); err != nil {
			return fmt.Errorf("failed to delete processed slack messages for job: %w", err)
		}

		if _, err := s.jobsRepo.DeleteJob(ctx, id, slackIntegrationID, organizationID); err != nil {
			return fmt.Errorf("failed to delete job: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete job in transaction: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted job with ID: %s", id)
	return nil
}

func (s *JobsService) CreateProcessedSlackMessage(
	ctx context.Context,
	jobID string,
	slackChannelID, slackTS, textContent, slackIntegrationID string,
	organizationID string,
	status models.ProcessedSlackMessageStatus,
) (*models.ProcessedSlackMessage, error) {
	log.Printf(
		"📋 Starting to create processed slack message for job: %s, channel: %s, ts: %s, organization: %s",
		jobID,
		slackChannelID,
		slackTS,
		organizationID,
	)

	if !core.IsValidULID(jobID) {
		return nil, fmt.Errorf("job ID must be a valid ULID")
	}
	if slackChannelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}
	if slackTS == "" {
		return nil, fmt.Errorf("slack_ts cannot be empty")
	}
	if textContent == "" {
		return nil, fmt.Errorf("text_content cannot be empty")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	message := &models.ProcessedSlackMessage{
		ID:                 core.NewID("psm"),
		JobID:              jobID,
		SlackChannelID:     slackChannelID,
		SlackTS:            slackTS,
		TextContent:        textContent,
		Status:             status,
		SlackIntegrationID: slackIntegrationID,
		OrganizationID:     organizationID,
	}

	if err := s.processedSlackMessagesRepo.CreateProcessedSlackMessage(ctx, message); err != nil {
		return nil, fmt.Errorf("failed to create processed slack message: %w", err)
	}

	log.Printf("📋 Completed successfully - created processed slack message with ID: %s", message.ID)
	return message, nil
}

func (s *JobsService) UpdateProcessedSlackMessage(
	ctx context.Context,
	id string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) (*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to update processed slack message status for ID: %s to %s", id, status)
	if !core.IsValidULID(id) {
		return nil, fmt.Errorf("processed slack message ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeUpdatedMessage, err := s.processedSlackMessagesRepo.UpdateProcessedSlackMessageStatus(
		ctx,
		id,
		status,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update processed slack message status: %w", err)
	}
	if !maybeUpdatedMessage.IsPresent() {
		return nil, core.ErrNotFound
	}
	updatedMessage := maybeUpdatedMessage.MustGet()

	log.Printf("📋 Completed successfully - updated processed slack message ID: %s to status: %s", id, status)
	return updatedMessage, nil
}

func (s *JobsService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedSlackMessageStatus,
	slackIntegrationID string,
	organizationID string,
) ([]*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to get processed messages for job: %s with status: %s", jobID, status)
	if !core.IsValidULID(jobID) {
		return nil, fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	messages, err := s.processedSlackMessagesRepo.GetProcessedMessagesByJobIDAndStatus(
		ctx,
		jobID,
		status,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d processed messages", len(messages))
	return messages, nil
}

func (s *JobsService) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	log.Printf("📋 Starting to get processed slack message by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("processed slack message ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeMsg, err := s.processedSlackMessagesRepo.GetProcessedSlackMessageByID(
		ctx,
		id,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("failed to get processed slack message: %w", err)
	}
	if !maybeMsg.IsPresent() {
		log.Printf("📋 Completed successfully - processed slack message not found")
		return mo.None[*models.ProcessedSlackMessage](), nil
	}
	message := maybeMsg.MustGet()

	log.Printf("📋 Completed successfully - retrieved processed slack message with ID: %s", message.ID)
	return mo.Some(message), nil
}

// TESTS_UpdateJobUpdatedAt updates the updated_at timestamp of a job for testing purposes
func (s *JobsService) TESTS_UpdateJobUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to update job updated_at for testing purposes: %s to %s", id, updatedAt)
	if !core.IsValidULID(id) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	updated, err := s.jobsRepo.TESTS_UpdateJobUpdatedAt(ctx, id, updatedAt, slackIntegrationID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update job updated_at: %w", err)
	}
	if !updated {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - updated job updated_at for ID: %s", id)
	return nil
}

// TESTS_UpdateProcessedSlackMessageUpdatedAt updates the updated_at timestamp of a processed slack message for testing purposes
func (s *JobsService) TESTS_UpdateProcessedSlackMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to update processed slack message updated_at for testing purposes: %s to %s", id, updatedAt)
	if !core.IsValidULID(id) {
		return fmt.Errorf("processed slack message ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	updated, err := s.processedSlackMessagesRepo.TESTS_UpdateProcessedSlackMessageUpdatedAt(
		ctx,
		id,
		updatedAt,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed slack message updated_at: %w", err)
	}
	if !updated {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - updated processed slack message updated_at for ID: %s", id)
	return nil
}

// GetJobsWithQueuedMessages returns jobs that have at least one message in QUEUED status
func (s *JobsService) GetJobsWithQueuedMessages(
	ctx context.Context,
	slackIntegrationID string,
	organizationID string,
) ([]*models.Job, error) {
	log.Printf("📋 Starting to get jobs with queued messages")
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	jobs, err := s.jobsRepo.GetJobsWithQueuedMessages(ctx, slackIntegrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d jobs with queued messages", len(jobs))
	return jobs, nil
}

// GetLatestProcessedMessageForJob returns the most recent processed message for a job
func (s *JobsService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	slackIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	log.Printf("📋 Starting to get latest processed message for job: %s", jobID)
	if !core.IsValidULID(jobID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeMsg, err := s.processedSlackMessagesRepo.GetLatestProcessedMessageForJob(
		ctx,
		jobID,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("failed to get latest processed message: %w", err)
	}
	if !maybeMsg.IsPresent() {
		log.Printf("📋 Completed successfully - no processed message found for job")
		return mo.None[*models.ProcessedSlackMessage](), nil
	}
	message := maybeMsg.MustGet()

	log.Printf("📋 Completed successfully - retrieved latest processed message with ID: %s", message.ID)
	return mo.Some(message), nil
}

// GetJobWithIntegrationByID gets a job by ID using organization_id directly (optimization)
func (s *JobsService) GetJobWithIntegrationByID(
	ctx context.Context,
	jobID string,
	organizationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job with integration by ID: %s for organization: %s", jobID, organizationID)
	if !core.IsValidULID(jobID) {
		return mo.None[*models.Job](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobWithIntegrationByID(ctx, jobID, organizationID)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to get job with integration: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("📋 Completed successfully - job not found")
		return mo.None[*models.Job](), nil
	}
	job := maybeJob.MustGet()

	log.Printf("📋 Completed successfully - retrieved job with integration ID: %s", job.ID)
	return mo.Some(job), nil
}

// GetProcessedSlackMessageWithIntegrationByID gets a processed slack message by ID using organization_id directly (optimization)
func (s *JobsService) GetProcessedSlackMessageWithIntegrationByID(
	ctx context.Context,
	messageID string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	log.Printf(
		"📋 Starting to get processed slack message with integration by ID: %s for organization: %s",
		messageID,
		organizationID,
	)
	if !core.IsValidULID(messageID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("message ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf(
			"organization_id must be a valid ULID",
		)
	}

	maybeMessage, err := s.processedSlackMessagesRepo.GetProcessedSlackMessageWithIntegrationByID(
		ctx,
		messageID,
		organizationID,
	)
	if err != nil {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf(
			"failed to get processed slack message with integration: %w",
			err,
		)
	}
	if !maybeMessage.IsPresent() {
		log.Printf("📋 Completed successfully - processed slack message not found")
		return mo.None[*models.ProcessedSlackMessage](), nil
	}
	message := maybeMessage.MustGet()

	log.Printf(
		"📋 Completed successfully - retrieved processed slack message with integration ID: %s",
		message.ID,
	)
	return mo.Some(message), nil
}
