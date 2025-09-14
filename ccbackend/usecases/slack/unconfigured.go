package slack

import (
	"context"
	"fmt"

	"ccbackend/models"
)

// UnconfiguredSlackUseCase returns errors for all operations when Slack is not configured
type UnconfiguredSlackUseCase struct{}

// NewUnconfiguredSlackUseCase creates a new unconfigured Slack use case
func NewUnconfiguredSlackUseCase() *UnconfiguredSlackUseCase {
	return &UnconfiguredSlackUseCase{}
}

func (u *UnconfiguredSlackUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessQueuedJobs(ctx context.Context) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) CleanupFailedSlackJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	message string,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}

func (u *UnconfiguredSlackUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("slack use case is not configured")
}
