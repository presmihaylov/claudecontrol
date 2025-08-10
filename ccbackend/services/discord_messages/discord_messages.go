package discordmessages

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type DiscordMessagesService struct {
	processedDiscordMessagesRepo *db.PostgresProcessedDiscordMessagesRepository
}

func NewDiscordMessagesService(repo *db.PostgresProcessedDiscordMessagesRepository) *DiscordMessagesService {
	return &DiscordMessagesService{
		processedDiscordMessagesRepo: repo,
	}
}

func (s *DiscordMessagesService) CreateProcessedDiscordMessage(
	ctx context.Context,
	jobID string,
	discordMessageID, discordThreadID, textContent, discordIntegrationID string,
	organizationID string,
	status models.ProcessedDiscordMessageStatus,
) (*models.ProcessedDiscordMessage, error) {
	log.Printf(
		"📋 Starting to create processed discord message for job: %s, message: %s, thread: %s, organization: %s",
		jobID,
		discordMessageID,
		discordThreadID,
		organizationID,
	)

	if !core.IsValidULID(jobID) {
		return nil, fmt.Errorf("job ID must be a valid ULID")
	}
	if discordMessageID == "" {
		return nil, fmt.Errorf("discord_message_id cannot be empty")
	}
	if discordThreadID == "" {
		return nil, fmt.Errorf("discord_thread_id cannot be empty")
	}
	if textContent == "" {
		return nil, fmt.Errorf("text_content cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return nil, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}
	if status == "" {
		return nil, fmt.Errorf("status cannot be empty")
	}

	processedMessage := &models.ProcessedDiscordMessage{
		ID:                   core.NewID("dm"),
		JobID:                jobID,
		DiscordMessageID:     discordMessageID,
		DiscordThreadID:      discordThreadID,
		TextContent:          textContent,
		Status:               status,
		DiscordIntegrationID: discordIntegrationID,
		OrganizationID:       organizationID,
	}

	if err := s.processedDiscordMessagesRepo.CreateProcessedDiscordMessage(ctx, processedMessage); err != nil {
		return nil, fmt.Errorf("failed to create processed discord message: %w", err)
	}

	log.Printf("📋 Completed successfully - created processed discord message with ID: %s", processedMessage.ID)
	return processedMessage, nil
}

func (s *DiscordMessagesService) UpdateProcessedDiscordMessage(
	ctx context.Context,
	id string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) (*models.ProcessedDiscordMessage, error) {
	log.Printf("📋 Starting to update processed discord message status for ID: %s to %s", id, status)
	if !core.IsValidULID(id) {
		return nil, fmt.Errorf("processed discord message ID must be a valid ULID")
	}
	if status == "" {
		return nil, fmt.Errorf("status cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return nil, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	updatedMessage, err := s.processedDiscordMessagesRepo.UpdateProcessedDiscordMessage(
		ctx,
		id,
		status,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to update processed discord message: %w", err)
	}

	log.Printf("📋 Completed successfully - updated processed discord message status for ID: %s", id)
	return updatedMessage, nil
}

func (s *DiscordMessagesService) GetProcessedMessagesByJobIDAndStatus(
	ctx context.Context,
	jobID string,
	status models.ProcessedDiscordMessageStatus,
	discordIntegrationID string,
	organizationID string,
) ([]*models.ProcessedDiscordMessage, error) {
	log.Printf("📋 Starting to get processed discord messages by job ID: %s and status: %s", jobID, status)
	if !core.IsValidULID(jobID) {
		return nil, fmt.Errorf("job ID must be a valid ULID")
	}
	if status == "" {
		return nil, fmt.Errorf("status cannot be empty")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return nil, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	messages, err := s.processedDiscordMessagesRepo.GetProcessedMessagesByJobIDAndStatus(
		ctx,
		jobID,
		status,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get processed discord messages by job ID and status: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d processed discord messages", len(messages))
	return messages, nil
}

func (s *DiscordMessagesService) GetProcessedDiscordMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	log.Printf("📋 Starting to get processed discord message by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf(
			"processed discord message ID must be a valid ULID",
		)
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeMessage, err := s.processedDiscordMessagesRepo.GetProcessedDiscordMessageByID(ctx, id, organizationID)
	if err != nil {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf(
			"failed to get processed discord message by ID: %w",
			err,
		)
	}
	if !maybeMessage.IsPresent() {
		log.Printf("📋 Completed successfully - processed discord message not found")
		return mo.None[*models.ProcessedDiscordMessage](), nil
	}
	message := maybeMessage.MustGet()

	log.Printf("📋 Completed successfully - retrieved processed discord message with ID: %s", message.ID)
	return mo.Some(message), nil
}

func (s *DiscordMessagesService) GetLatestProcessedMessageForJob(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) (mo.Option[*models.ProcessedDiscordMessage], error) {
	log.Printf("📋 Starting to get latest processed discord message for job: %s", jobID)
	if !core.IsValidULID(jobID) {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeMessage, err := s.processedDiscordMessagesRepo.GetLatestProcessedMessageForJob(
		ctx,
		jobID,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return mo.None[*models.ProcessedDiscordMessage](), fmt.Errorf(
			"failed to get latest processed discord message for job: %w",
			err,
		)
	}
	if !maybeMessage.IsPresent() {
		log.Printf("📋 Completed successfully - no processed discord messages found for job")
		return mo.None[*models.ProcessedDiscordMessage](), nil
	}
	message := maybeMessage.MustGet()

	log.Printf("📋 Completed successfully - retrieved latest processed discord message for job: %s", jobID)
	return mo.Some(message), nil
}

func (s *DiscordMessagesService) GetActiveMessageCountForJobs(
	ctx context.Context,
	jobIDs []string,
	discordIntegrationID string,
	organizationID string,
) (int, error) {
	log.Printf("📋 Starting to get active discord message count for %d jobs", len(jobIDs))
	if len(jobIDs) == 0 {
		return 0, fmt.Errorf("job IDs cannot be empty")
	}
	for _, jobID := range jobIDs {
		if !core.IsValidULID(jobID) {
			return 0, fmt.Errorf("job ID must be a valid ULID: %s", jobID)
		}
	}
	if !core.IsValidULID(discordIntegrationID) {
		return 0, fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return 0, fmt.Errorf("organization_id must be a valid ULID")
	}

	count, err := s.processedDiscordMessagesRepo.GetActiveMessageCountForJobs(
		ctx,
		jobIDs,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get active discord message count: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d active discord messages", count)
	return count, nil
}

func (s *DiscordMessagesService) TESTS_UpdateProcessedDiscordMessageUpdatedAt(
	ctx context.Context,
	id string,
	updatedAt time.Time,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to update processed discord message updated_at for testing purposes: %s to %s",
		id,
		updatedAt,
	)
	if !core.IsValidULID(id) {
		return fmt.Errorf("processed discord message ID must be a valid ULID")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	updated, err := s.processedDiscordMessagesRepo.TESTS_UpdateProcessedDiscordMessageUpdatedAt(
		ctx,
		id,
		updatedAt,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed discord message updated_at: %w", err)
	}
	if !updated {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - updated processed discord message updated_at for ID: %s", id)
	return nil
}

func (s *DiscordMessagesService) DeleteProcessedDiscordMessagesByJobID(
	ctx context.Context,
	jobID string,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to delete processed discord messages by job ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(discordIntegrationID) {
		return fmt.Errorf("discord_integration_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	err := s.processedDiscordMessagesRepo.DeleteProcessedDiscordMessagesByJobID(
		ctx,
		jobID,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to delete processed discord messages by job ID: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted processed discord messages for job ID: %s", jobID)
	return nil
}
