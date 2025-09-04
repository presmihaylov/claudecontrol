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
	"ccbackend/salesnotif"
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
	orgID models.OrgID,
	slackThreadTS, slackChannelID, slackUserID, slackIntegrationID string,
) (*models.Job, error) {
	log.Printf(
		"📋 Starting to create job for slack thread: %s, channel: %s, user: %s, organization: %s",
		slackThreadTS,
		slackChannelID,
		slackUserID,
		orgID,
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
	if !core.IsValidULID(string(orgID)) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	job := &models.Job{
		ID:      core.NewID("j"),
		JobType: models.JobTypeSlack,
		OrgID:   orgID,
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

	// Send sales notification for job creation
	salesnotif.New(fmt.Sprintf("Org %s created a new job %s", orgID, job.ID))

	log.Printf("📋 Completed successfully - created job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByID(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.Job](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(string(orgID)) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobByID(ctx, id, orgID)
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
	orgID models.OrgID,
	threadTS, channelID, slackIntegrationID string,
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
	if !core.IsValidULID(string(orgID)) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobBySlackThread(ctx, threadTS, channelID, slackIntegrationID, orgID)
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
	orgID models.OrgID,
	threadTS, channelID, slackUserID, slackIntegrationID string,
) (*models.JobCreationResult, error) {
	log.Printf(
		"📋 Starting to get or create job for slack thread: %s, channel: %s, user: %s, organization: %s",
		threadTS,
		channelID,
		slackUserID,
		orgID,
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
	if !core.IsValidULID(string(orgID)) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Try to find existing job first
	maybeExistingJob, err := s.jobsRepo.GetJobBySlackThread(
		ctx,
		threadTS,
		channelID,
		slackIntegrationID,
		orgID,
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
	newJob, createErr := s.CreateSlackJob(ctx, orgID, threadTS, channelID, slackUserID, slackIntegrationID)
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
	orgID models.OrgID,
	jobID string,
) error {
	log.Printf("📋 Starting to update job timestamp for ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(string(orgID)) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}
	if err := s.jobsRepo.UpdateJobTimestamp(ctx, jobID, orgID); err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - updated timestamp for job ID: %s", jobID)
	return nil
}

func (s *JobsService) GetIdleJobs(
	ctx context.Context,
	orgID models.OrgID,
	idleMinutes int,
) ([]*models.Job, error) {
	log.Printf("📋 Starting to get idle jobs older than %d minutes for organization: %s", idleMinutes, orgID)
	if idleMinutes <= 0 {
		return nil, fmt.Errorf("idle minutes must be greater than 0")
	}
	if !core.IsValidULID(string(orgID)) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Get all jobs from database
	allJobs, err := s.jobsRepo.GetJobs(ctx, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs: %w", err)
	}

	log.Printf("🔍 Found %d total jobs, filtering for idle jobs older than %d minutes", len(allJobs), idleMinutes)

	var idleJobs []*models.Job

	// Check each job for idle status and active messages using direct repository calls
	idleThreshold := time.Now().Add(-time.Duration(idleMinutes) * time.Minute)

	for _, job := range allJobs {
		// First check if job is old enough to be considered for idle status
		if job.UpdatedAt.After(idleThreshold) {
			continue // Job is too recent, skip
		}

		hasActiveMessages := false
		switch job.JobType {
		case models.JobTypeSlack:
			active, err := s.hasActiveSlackMessages(ctx, orgID, job)
			if err != nil {
				log.Printf(
					"⚠️ Failed to check active Slack messages for job %s: %v - being conservative and marking as active",
					job.ID,
					err,
				)
				return nil, fmt.Errorf("failed to check active Slack messages for job %s: %w", job.ID, err)
			}

			hasActiveMessages = active
		case models.JobTypeDiscord:
			active, err := s.hasActiveDiscordMessages(ctx, orgID, job)
			if err != nil {
				log.Printf(
					"⚠️ Failed to check active Discord messages for job %s: %v - being conservative and marking as active",
					job.ID,
					err,
				)
				return nil, fmt.Errorf("failed to check active Discord messages for job %s: %w", job.ID, err)
			}

			hasActiveMessages = active
		}

		if !hasActiveMessages {
			idleJobs = append(idleJobs, job)
			log.Printf("✅ Job %s confirmed idle (no active messages)", job.ID)
		} else {
			log.Printf("🔄 Job %s has active messages - not marking as idle", job.ID)
		}
	}

	log.Printf(
		"📋 Completed successfully - found %d idle jobs out of %d total jobs",
		len(idleJobs),
		len(allJobs),
	)
	return idleJobs, nil
}

// hasActiveSlackMessages checks if a Slack job has any QUEUED or IN_PROGRESS messages
func (s *JobsService) hasActiveSlackMessages(
	ctx context.Context,
	orgID models.OrgID,
	job *models.Job,
) (bool, error) {
	if job.SlackPayload == nil {
		return false, nil
	}

	// Check for QUEUED messages
	queuedMsgs, err := s.slackMessagesService.GetProcessedMessagesByJobIDAndStatus(
		ctx, orgID, job.ID, models.ProcessedSlackMessageStatusQueued, job.SlackPayload.IntegrationID,
	)
	if err != nil {
		return true, fmt.Errorf("failed to check queued messages for job %s: %w", job.ID, err)
	}
	if len(queuedMsgs) > 0 {
		return true, nil
	}

	// Check for IN_PROGRESS messages
	inProgressMsgs, err := s.slackMessagesService.GetProcessedMessagesByJobIDAndStatus(
		ctx, orgID, job.ID, models.ProcessedSlackMessageStatusInProgress, job.SlackPayload.IntegrationID,
	)
	if err != nil {
		return true, fmt.Errorf("failed to check in-progress messages for job %s: %w", job.ID, err)
	}

	return len(inProgressMsgs) > 0, nil
}

// hasActiveDiscordMessages checks if a Discord job has any QUEUED or IN_PROGRESS messages
func (s *JobsService) hasActiveDiscordMessages(
	ctx context.Context,
	orgID models.OrgID,
	job *models.Job,
) (bool, error) {
	if job.DiscordPayload == nil {
		return false, nil
	}

	// Check for QUEUED messages
	queuedMsgs, err := s.discordMessagesService.GetProcessedMessagesByJobIDAndStatus(
		ctx, orgID, job.ID, models.ProcessedDiscordMessageStatusQueued, job.DiscordPayload.IntegrationID,
	)
	if err != nil {
		return true, fmt.Errorf("failed to check queued Discord messages for job %s: %w", job.ID, err)
	}
	if len(queuedMsgs) > 0 {
		return true, nil
	}

	// Check for IN_PROGRESS messages
	inProgressMsgs, err := s.discordMessagesService.GetProcessedMessagesByJobIDAndStatus(
		ctx, orgID, job.ID, models.ProcessedDiscordMessageStatusInProgress, job.DiscordPayload.IntegrationID,
	)
	if err != nil {
		return true, fmt.Errorf("failed to check in-progress Discord messages for job %s: %w", job.ID, err)
	}

	return len(inProgressMsgs) > 0, nil
}

func (s *JobsService) DeleteJob(
	ctx context.Context,
	orgID models.OrgID,
	id string,
) error {
	log.Printf("📋 Starting to delete job with ID: %s", id)
	if !core.IsValidULID(id) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(string(orgID)) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	// First, get the job to determine its type
	maybeJob, err := s.jobsRepo.GetJobByID(ctx, id, orgID)
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
			if err := s.slackMessagesService.DeleteProcessedSlackMessagesByJobID(ctx, orgID, id, job.SlackPayload.IntegrationID); err != nil {
				return fmt.Errorf("failed to delete processed slack messages for job: %w", err)
			}
		case models.JobTypeDiscord:
			if job.DiscordPayload == nil {
				return fmt.Errorf("discord job missing discord payload")
			}
			if err := s.discordMessagesService.DeleteProcessedDiscordMessagesByJobID(ctx, orgID, id, job.DiscordPayload.IntegrationID); err != nil {
				return fmt.Errorf("failed to delete processed discord messages for job: %w", err)
			}
		default:
			return fmt.Errorf("unsupported job type for deletion: %s", job.JobType)
		}

		// Delete the job itself
		if _, err := s.jobsRepo.DeleteJob(ctx, id, orgID); err != nil {
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
	orgID models.OrgID,
	id string,
	updatedAt time.Time,
	slackIntegrationID string,
) error {
	log.Printf("📋 Starting to update job updated_at for testing purposes: %s to %s", id, updatedAt)
	if !core.IsValidULID(id) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(string(orgID)) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	updated, err := s.jobsRepo.TESTS_UpdateJobUpdatedAt(ctx, id, updatedAt, slackIntegrationID, orgID)
	if err != nil {
		return fmt.Errorf("failed to update job updated_at: %w", err)
	}
	if !updated {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - updated job updated_at for ID: %s", id)
	return nil
}

// Discord-specific job methods

func (s *JobsService) CreateDiscordJob(
	ctx context.Context,
	orgID models.OrgID,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
) (*models.Job, error) {
	log.Printf(
		"📋 Starting to create discord job for message: %s, channel: %s, thread: %s, user: %s, organization: %s",
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		orgID,
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
	if !core.IsValidULID(string(orgID)) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	job := &models.Job{
		ID:      core.NewID("j"),
		JobType: models.JobTypeDiscord,
		OrgID:   orgID,
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

	// Send sales notification for job creation
	salesnotif.New(fmt.Sprintf("Org %s created a new job %s", orgID, job.ID))

	log.Printf("📋 Completed successfully - created discord job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByDiscordThread(
	ctx context.Context,
	orgID models.OrgID,
	threadID, discordIntegrationID string,
) (mo.Option[*models.Job], error) {
	log.Printf("📋 Starting to get job by discord thread: %s", threadID)
	if threadID == "" {
		return mo.None[*models.Job](), fmt.Errorf("discord_thread_id cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return mo.None[*models.Job](), fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(string(orgID)) {
		return mo.None[*models.Job](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeJob, err := s.jobsRepo.GetJobByDiscordThread(ctx, threadID, discordIntegrationID, orgID)
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
	orgID models.OrgID,
	discordMessageID, discordChannelID, discordThreadID, discordUserID, discordIntegrationID string,
) (*models.JobCreationResult, error) {
	log.Printf(
		"📋 Starting to get or create job for discord thread: %s, channel: %s, message: %s, user: %s, organization: %s",
		discordThreadID,
		discordChannelID,
		discordMessageID,
		discordUserID,
		orgID,
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
	if !core.IsValidULID(string(orgID)) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Try to find existing job first
	maybeExistingJob, err := s.jobsRepo.GetJobByDiscordThread(
		ctx,
		discordThreadID,
		discordIntegrationID,
		orgID,
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
		orgID,
		discordMessageID,
		discordChannelID,
		discordThreadID,
		discordUserID,
		discordIntegrationID,
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
