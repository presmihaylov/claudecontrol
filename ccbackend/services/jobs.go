package services

import (
	"fmt"
	"log"
	"strings"

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

func (s *JobsService) CreateJob(slackThreadTS, slackChannelID string) (*models.Job, error) {
	log.Printf("📋 Starting to create job for slack thread: %s, channel: %s", slackThreadTS, slackChannelID)

	if slackThreadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}

	if slackChannelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}

	id := uuid.New()

	job := &models.Job{
		ID:             id,
		SlackThreadTS:  slackThreadTS,
		SlackChannelID: slackChannelID,
	}

	if err := s.jobsRepo.CreateJob(job); err != nil {
		return nil, fmt.Errorf("failed to create job: %w", err)
	}

	log.Printf("📋 Completed successfully - created job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetJobByID(id uuid.UUID) (*models.Job, error) {
	log.Printf("📋 Starting to get job by ID: %s", id)

	if id == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
	}

	job, err := s.jobsRepo.GetJobByID(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved job with ID: %s", job.ID)
	return job, nil
}

func (s *JobsService) GetOrCreateJobForSlackThread(threadTS, channelID string) (*models.Job, error) {
	log.Printf("📋 Starting to get or create job for slack thread: %s, channel: %s", threadTS, channelID)

	if threadTS == "" {
		return nil, fmt.Errorf("slack_thread_ts cannot be empty")
	}

	if channelID == "" {
		return nil, fmt.Errorf("slack_channel_id cannot be empty")
	}

	// Try to find existing job first
	existingJob, err := s.jobsRepo.GetJobBySlackThread(threadTS, channelID)
	if err == nil {
		log.Printf("📋 Completed successfully - found existing job with ID: %s", existingJob.ID)
		return existingJob, nil
	}

	// If not found, create a new job
	if strings.Contains(fmt.Sprintf("%v", err), "not found") {
		newJob, createErr := s.CreateJob(threadTS, channelID)
		if createErr != nil {
			return nil, fmt.Errorf("failed to create new job: %w", createErr)
		}
		log.Printf("📋 Completed successfully - created new job with ID: %s", newJob.ID)
		return newJob, nil
	}

	// If there was a different error, return it
	return nil, fmt.Errorf("failed to get job by slack thread: %w", err)
}

func (s *JobsService) UpdateJobTimestamp(jobID uuid.UUID) error {
	log.Printf("📋 Starting to update job timestamp for ID: %s", jobID)

	if jobID == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if err := s.jobsRepo.UpdateJobTimestamp(jobID); err != nil {
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - updated timestamp for job ID: %s", jobID)
	return nil
}

func (s *JobsService) GetIdleJobs(idleMinutes int) ([]*models.Job, error) {
	log.Printf("📋 Starting to get idle jobs older than %d minutes", idleMinutes)

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

func (s *JobsService) DeleteJob(id uuid.UUID) error {
	log.Printf("📋 Starting to delete job with ID: %s", id)

	if id == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if err := s.processedSlackMessagesRepo.DeleteProcessedSlackMessagesByJobID(id); err != nil {
		return fmt.Errorf("failed to delete processed slack messages for job: %w", err)
	}

	if err := s.jobsRepo.DeleteJob(id); err != nil {
		return fmt.Errorf("failed to delete job: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted job with ID: %s", id)
	return nil
}

func (s *JobsService) CreateProcessedSlackMessage(jobID uuid.UUID, slackChannelID, slackTS, textContent string, status models.ProcessedSlackMessageStatus) (*models.ProcessedSlackMessage, error) {
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

	id := uuid.New()

	message := &models.ProcessedSlackMessage{
		ID:             id,
		JobID:          jobID,
		SlackChannelID: slackChannelID,
		SlackTS:        slackTS,
		TextContent:    textContent,
		Status:         status,
	}

	if err := s.processedSlackMessagesRepo.CreateProcessedSlackMessage(message); err != nil {
		return nil, fmt.Errorf("failed to create processed slack message: %w", err)
	}

	log.Printf("📋 Completed successfully - created processed slack message with ID: %s", message.ID)
	return message, nil
}

func (s *JobsService) UpdateProcessedSlackMessage(id uuid.UUID, status models.ProcessedSlackMessageStatus) error {
	log.Printf("📋 Starting to update processed slack message status for ID: %s to %s", id, status)

	if id == uuid.Nil {
		return fmt.Errorf("processed slack message ID cannot be nil")
	}

	if err := s.processedSlackMessagesRepo.UpdateProcessedSlackMessageStatus(id, status); err != nil {
		return fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	log.Printf("📋 Completed successfully - updated processed slack message ID: %s to status: %s", id, status)
	return nil
}

func (s *JobsService) GetProcessedMessagesByJobIDAndStatus(jobID uuid.UUID, status models.ProcessedSlackMessageStatus) ([]*models.ProcessedSlackMessage, error) {
	log.Printf("📋 Starting to get processed messages for job: %s with status: %s", jobID, status)

	if jobID == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
	}

	messages, err := s.processedSlackMessagesRepo.GetProcessedMessagesByJobIDAndStatus(jobID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed messages: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d processed messages", len(messages))
	return messages, nil
}