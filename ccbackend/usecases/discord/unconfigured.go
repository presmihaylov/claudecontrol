package discord

import (
	"context"
	"fmt"

	"ccbackend/models"
)

// UnconfiguredDiscordUseCase returns errors for all operations when Discord is not configured
type UnconfiguredDiscordUseCase struct{}

// NewUnconfiguredDiscordUseCase creates a new unconfigured Discord use case
func NewUnconfiguredDiscordUseCase() *UnconfiguredDiscordUseCase {
	return &UnconfiguredDiscordUseCase{}
}

func (u *UnconfiguredDiscordUseCase) ProcessDiscordMessageEvent(
	ctx context.Context,
	event models.DiscordMessageEvent,
	discordIntegrationID string,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessDiscordReactionEvent(
	ctx context.Context,
	event models.DiscordReactionEvent,
	discordIntegrationID string,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) CleanupFailedDiscordJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	message string,
) error {
	return fmt.Errorf("Discord use case is not configured")
}

func (u *UnconfiguredDiscordUseCase) ProcessQueuedJobs(ctx context.Context) error {
	return fmt.Errorf("Discord use case is not configured")
}