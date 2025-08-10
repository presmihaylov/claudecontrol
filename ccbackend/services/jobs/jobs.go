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
	jobsRepo               *db.PostgresJobsRepository
	slackMessagesService   services.SlackMessagesService
	discordMessagesService services.DiscordMessagesService
	txManager              services.TransactionManager
}

func NewJobsService(
	repo *db.PostgresJobsRepository,
	slackMessagesService services.SlackMessagesService,
	discordMessagesService services.DiscordMessagesService,
	txManager services.TransactionManager,
) *JobsService {
	return &JobsService{
		jobsRepo:               repo,
		slackMessagesService:   slackMessagesService,
		discordMessagesService: discordMessagesService,
		txManager:              txManager,
	}
}

func (s *JobsService) CreateSlackJob(
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
		ID:             core.NewID("j"),
		JobType:        models.JobTypeSlack,
		OrganizationID: organizationID,
		SlackPayload: &models.SlackJobPayload{
			ThreadTS:      slackThreadTS,
			ChannelID:     slackChannelID,
			UserID:        slackUserID,
			IntegrationID: slackIntegrationID,
		},
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
	organizationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.Job](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobByID(ctx, id, organizationID)
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
	newJob, createErr := s.CreateSlackJob(ctx, threadTS, channelID, slackUserID, slackIntegrationID, organizationID)
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
	organizationID string,
) error {
	log.Printf("📋 Starting to update job timestamp for ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}
	if err := s.jobsRepo.UpdateJobTimestamp(ctx, jobID, organizationID); err != nil {
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
	organizationID string,
) error {
	log.Printf("📋 Starting to delete job with ID: %s", id)
	if !core.IsValidULID(id) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	// First, get the job to determine its type
	maybeJob, err := s.jobsRepo.GetJobByID(ctx, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job for deletion: %w", err)
	}
	if !maybeJob.IsPresent() {
		// Job not found - delete operation is idempotent, so this is successful
		log.Printf("📋 Completed successfully - job not found (idempotent delete): %s", id)
		return nil
	}
	job := maybeJob.MustGet()

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Delete messages based on job type
		switch job.JobType {
		case models.JobTypeSlack:
			if job.SlackPayload == nil {
				return fmt.Errorf("slack job missing slack payload")
			}
			if err := s.slackMessagesService.DeleteProcessedSlackMessagesByJobID(ctx, id, job.SlackPayload.IntegrationID, organizationID); err != nil {
				return fmt.Errorf("failed to delete processed slack messages for job: %w", err)
			}
		case models.JobTypeDiscord:
			if job.DiscordPayload == nil {
				return fmt.Errorf("discord job missing discord payload")
			}
			if err := s.discordMessagesService.DeleteProcessedDiscordMessagesByJobID(ctx, id, job.DiscordPayload.IntegrationID, organizationID); err != nil {
				return fmt.Errorf("failed to delete processed discord messages for job: %w", err)
			}
		default:
			return fmt.Errorf("unsupported job type for deletion: %s", job.JobType)
		}

		// Delete the job itself
		if _, err := s.jobsRepo.DeleteJob(ctx, id, organizationID); err != nil {
			return fmt.Errorf("failed to delete job: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to delete job in transaction: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted job with ID: %s", id)
	return nil
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

// GetJobsWithQueuedMessages returns jobs that have at least one message in QUEUED status
func (s *JobsService) GetJobsWithQueuedMessages(
	ctx context.Context,
	jobType models.JobType,
	integrationID string,
	organizationID string,
) ([]*models.Job, error) {
	log.Printf("📋 Starting to get jobs with queued messages for job type: %s", jobType)
	if jobType == "" {
		return nil, fmt.Errorf("job_type cannot be empty")
	}
	if !core.IsValidULID(integrationID) {
		return nil, fmt.Errorf("integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	jobs, err := s.jobsRepo.GetJobsWithQueuedMessages(ctx, jobType, integrationID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d jobs with queued messages for job type: %s", len(jobs), jobType)
	return jobs, nil
}

// Discord-specific job methods

func (s *JobsService) CreateDiscordJob(
	ctx context.Context,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	organizationID string,
) (*models.Job, error) {
	log.Printf(
		"📋 Starting to create discord job for message: %s, channel: %s, thread: %s, user: %s, organization: %s",
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		organizationID,
	)

	if discordMessageID == "" {
		return nil, fmt.Errorf("discord_message_id cannot be empty")
	}
	if discordChannelID == "" {
		return nil, fmt.Errorf("discord_channel_id cannot be empty")
	}
	if discordThreadID == "" {
		return nil, fmt.Errorf("discord_thread_id cannot be empty")
	}
	if discordUserID == "" {
		return nil, fmt.Errorf("discord_user_id cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return nil, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	job := &models.Job{
		ID:             core.NewID("j"),
		JobType:        models.JobTypeDiscord,
		OrganizationID: organizationID,
		DiscordPayload: &models.DiscordJobPayload{
			MessageID:     discordMessageID,
			ChannelID:     discordChannelID,
			ThreadID:      discordThreadID,
			UserID:        discordUserID,
			IntegrationID: discordIntegrationID,
		},
	}

	if err := s.jobsRepo.CreateJob(ctx, job); err != nil {
		return nil, fmt.Errorf("failed to create discord job: %w", err)
	}

	log.Printf("📋 Completed successfully - created discord job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByDiscordThread(
	ctx context.Context,
	threadID, discordIntegrationID, organizationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by discord thread: %s", threadID)
	if threadID == "" {
		return mo.None[*models.Job](), fmt.Errorf("discord_thread_id cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return mo.None[*models.Job](), fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobByDiscordThread(ctx, threadID, discordIntegrationID, organizationID)
	if err != nil {
		return mo.None[*models.Job](), fmt.Errorf("failed to get job by discord thread: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("📋 Completed successfully - job not found")
		return mo.None[*models.Job](), nil
	}
	job := maybeJob.MustGet()

	log.Printf("📋 Completed successfully - retrieved job with ID: %s", job.ID)
	return mo.Some(job), nil
}

func (s *JobsService) GetOrCreateJobForDiscordThread(
	ctx context.Context,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
	organizationID string,
) (*models.JobCreationResult, error) {
	log.Printf(
		"📋 Starting to get or create job for discord thread: %s, channel: %s, message: %s, user: %s, organization: %s",
		discordThreadID,
		discordChannelID,
		discordMessageID,
		discordUserID,
		organizationID,
	)

	if discordMessageID == "" {
		return nil, fmt.Errorf("discord_message_id cannot be empty")
	}
	if discordThreadID == "" {
		return nil, fmt.Errorf("discord_thread_id cannot be empty")
	}
	if discordUserID == "" {
		return nil, fmt.Errorf("discord_user_id cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return nil, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Try to find existing job first
	maybeExistingJob, err := s.jobsRepo.GetJobByDiscordThread(
		ctx,
		discordThreadID,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get job by discord thread: %w", err)
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
	newJob, createErr := s.CreateDiscordJob(
		ctx,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
		organizationID,
	)
	if createErr != nil {
		return nil, fmt.Errorf("failed to create new discord job: %w", createErr)
	}
	log.Printf("📋 Completed successfully - created new discord job with ID: %s", newJob.ID)
	return &models.JobCreationResult{
		Job:    newJob,
		Status: models.JobCreationStatusCreated,
	}, nil
}
