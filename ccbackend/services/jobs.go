package services

import (
	"fmt"
	"log"
	"strings"
	"time"

	"ccbackend/db"
	"ccbackend/models"

	"github.com/google/uuid"
)

type JobsService struct {
	jobsRepo                   *db.PostgresJobsRepository
	processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository
}

func NewJobsService(repo *db.PostgresJobsRepository, processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository) *JobsService {
	return &JobsService{
		jobsRepo:                   repo,
		processedSlackMessagesRepo: processedSlackMessagesRepo,
	}
}

func (s *JobsService) GetActiveMessageCountForJobs(jobIDs []uuid.UUID, slackIntegrationID string) (int, error) {
	log.Printf("📋 Starting to get active message count for %d jobs", len(jobIDs))
	
	count, err := s.processedSlackMessagesRepo.GetActiveMessageCountForJobs(jobIDs, slackIntegrationID)
	if err != nil {
		return 0, fmt.Errorf("failed to get active message count: %w", err)
	}
	
	log.Printf("📋 Completed successfully - found %d active messages", count)
	return count, nil
}

func (s *JobsService) CreateJob(slackThreadTS, slackChannelID, slackIntegrationID string) (*models.Job, error) {
	log.Printf("📋 Starting to create job for slack thread: %s, channel: %s", slackThreadTS, slackChannelID)

	if slackThreadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}

	if slackChannelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	id := uuid.New()
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("invalid slack_integration_id format: %w", err)
	}

	job := &models.Job{
		ID:                 id,
		SlackThreadTS:      slackThreadTS,
		SlackChannelID:     slackChannelID,
		SlackIntegrationID: integrationUUID,
	}

	if err := s.jobsRepo.CreateJob(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("📋 Completed successfully - created job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByID(id uuid.UUID, slackIntegrationID string) (*models.Job, error) {
	log.Printf("📋 Starting to get job by ID: %s", id)

	if id == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	job, err := s.jobsRepo.GetJobByID(id, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetOrCreateJobForSlackThread(threadTS, channelID, slackIntegrationID string) (*models.JobCreationResult, error) {
	log.Printf("📋 Starting to get or create job for slack thread: %s, channel: %s", threadTS, channelID)

	if threadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}

	if channelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	// Try to find existing job first
	existingJob, err := s.jobsRepo.GetJobBySlackThread(threadTS, channelID, slackIntegrationID)
	if err == nil {
		log.Printf("📋 Completed successfully - found existing job with ID: %s", existingJob.ID)
		return &models.JobCreationResult{
			Job:    existingJob,
			Status: models.JobCreationStatusNA,
		}, nil
	}

	// If not found, create a new job
	if strings.Contains(fmt.Sprintf("%v", err), "not found") {
		newJob, createErr := s.CreateJob(threadTS, channelID, slackIntegrationID)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create new job: %w", createErr)
		}
		log.Printf("📋 Completed successfully - created new job with ID: %s", newJob.ID)
		return &models.JobCreationResult{
			Job:    newJob,
			Status: models.JobCreationStatusCreated,
		}, nil
	}

	// If there was a different error, return it
	return nil, fmt.Errorf("failed to get job by slack thread: %w", err)
}

func (s *JobsService) UpdateJobTimestamp(jobID uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to update job timestamp for ID: %s", jobID)

	if jobID == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.jobsRepo.UpdateJobTimestamp(jobID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - updated timestamp for job ID: %s", jobID)
	return nil
}

func (s *JobsService) GetIdleJobs(idleMinutes int) ([]*models.Job, error) {
	log.Printf("📋 Starting to get idle jobs older than %d minutes across all integrations", idleMinutes)

	if idleMinutes <= 0 {
		return nil, fmt.Errorf("idle minutes must be greater than 0")
	}

	jobs, err := s.jobsRepo.GetIdleJobs(idleMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get idle jobs: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d idle jobs", len(jobs))
	return jobs, nil
}

func (s *JobsService) DeleteJob(id uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to delete job with ID: %s", id)

	if id == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(id, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to delete processed slack messages for job: %w", err)
	}

	if err := s.jobsRepo.DeleteJob(id, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted job with ID: %s", id)
	return nil
}

func (s *JobsService) CreateProcessedSlackMessage(jobID uuid.UUID, slackChannelID, slackTS, textContent, slackIntegrationID string, status models.ProcessedSlackMessageStatus) (*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to create processed slack message for job: %s, channel: %s, ts: %s", jobID, slackChannelID, slackTS)

	if jobID == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
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

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	id := uuid.New()
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("invalid slack_integration_id format: %w", err)
	}

	message := &models.ProcessedSlackMessage{
		ID:                 id,
		JobID:              jobID,
		SlackChannelID:     slackChannelID,
		SlackTS:            slackTS,
		TextContent:        textContent,
		Status:             status,
		SlackIntegrationID: integrationUUID,
	}

	if err := s.processedSlackMessagesRepo.CreateProcessedSlackMessage(message); err != nil {
		return nil, fmt.Errorf("failed to create processed slack message: %w", err)
	}

	log.Printf("📋 Completed successfully - created processed slack message with ID: %s", message.ID)
	return message, nil
}

func (s *JobsService) UpdateProcessedSlackMessage(id uuid.UUID, status models.ProcessedSlackMessageStatus, slackIntegrationID string) (*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to update processed slack message status for ID: %s to %s", id, status)

	if id == uuid.Nil {
		return nil, fmt.Errorf("processed slack message ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	updatedMessage, err := s.processedSlackMessagesRepo.UpdateProcessedSlackMessageStatus(id, status, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	log.Printf("📋 Completed successfully - updated processed slack message ID: %s to status: %s", id, status)
	return updatedMessage, nil
}

func (s *JobsService) GetProcessedMessagesByJobIDAndStatus(jobID uuid.UUID, status models.ProcessedSlackMessageStatus, slackIntegrationID string) ([]*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to get processed messages for job: %s with status: %s", jobID, status)

	if jobID == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	messages, err := s.processedSlackMessagesRepo.GetProcessedMessagesByJobIDAndStatus(jobID, status, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d processed messages", len(messages))
	return messages, nil
}

func (s *JobsService) GetProcessedSlackMessageByID(id uuid.UUID, slackIntegrationID string) (*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to get processed slack message by ID: %s", id)

	if id == uuid.Nil {
		return nil, fmt.Errorf("processed slack message ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	message, err := s.processedSlackMessagesRepo.GetProcessedSlackMessageByID(id, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed slack message: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved processed slack message with ID: %s", message.ID)
	return message, nil
}

// TESTS_UpdateJobUpdatedAt updates the updated_at timestamp of a job for testing purposes
func (s *JobsService) TESTS_UpdateJobUpdatedAt(id uuid.UUID, updatedAt time.Time, slackIntegrationID string) error {
	log.Printf("📋 Starting to update job updated_at for testing purposes: %s to %s", id, updatedAt)

	if id == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.jobsRepo.TESTS_UpdateJobUpdatedAt(id, updatedAt, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update job updated_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated job updated_at for ID: %s", id)
	return nil
}

// TESTS_UpdateProcessedSlackMessageUpdatedAt updates the updated_at timestamp of a processed slack message for testing purposes
func (s *JobsService) TESTS_UpdateProcessedSlackMessageUpdatedAt(id uuid.UUID, updatedAt time.Time, slackIntegrationID string) error {
	log.Printf("📋 Starting to update processed slack message updated_at for testing purposes: %s to %s", id, updatedAt)

	if id == uuid.Nil {
		return fmt.Errorf("processed slack message ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.processedSlackMessagesRepo.TESTS_UpdateProcessedSlackMessageUpdatedAt(id, updatedAt, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update processed slack message updated_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated processed slack message updated_at for ID: %s", id)
	return nil
}

// GetJobsWithQueuedMessages returns jobs that have at least one message in QUEUED status
func (s *JobsService) GetJobsWithQueuedMessages(slackIntegrationID string) ([]*models.Job, error) {
	log.Printf("📋 Starting to get jobs with queued messages")

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	jobs, err := s.jobsRepo.GetJobsWithQueuedMessages(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get jobs with queued messages: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d jobs with queued messages", len(jobs))
	return jobs, nil
}