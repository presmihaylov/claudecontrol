package slackmessages

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type SlackMessagesService struct {
	processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository
}

func NewSlackMessagesService(
	processedSlackMessagesRepo *db.PostgresProcessedSlackMessagesRepository,
) *SlackMessagesService {
	return &SlackMessagesService{
		processedSlackMessagesRepo: processedSlackMessagesRepo,
	}
}

func (s *SlackMessagesService) CreateProcessedSlackMessage(
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

func (s *SlackMessagesService) UpdateProcessedSlackMessage(
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

func (s *SlackMessagesService) GetProcessedMessagesByJobIDAndStatus(
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

func (s *SlackMessagesService) GetProcessedSlackMessageByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ProcessedSlackMessage], error) {
	log.Printf("📋 Starting to get processed slack message by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("processed slack message ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ProcessedSlackMessage](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeMsg, err := s.processedSlackMessagesRepo.GetProcessedSlackMessageByID(
		ctx,
		id,
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

func (s *SlackMessagesService) GetLatestProcessedMessageForJob(
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
