package usecases

import (
	"context"

	"ccbackend/models"
)

// SlackUseCaseInterface defines the interface for Slack use case operations
type SlackUseCaseInterface interface {
	ProcessSlackMessageEvent(
		ctx context.Context,
		event models.SlackMessageEvent,
		slackIntegrationID string,
		orgID models.OrgID,
	) error
	ProcessReactionAdded(
		ctx context.Context,
		reactionName, userID, channelID, messageTS, slackIntegrationID string,
		orgID models.OrgID,
	) error
	ProcessProcessingMessage(
		ctx context.Context,
		clientID string,
		payload models.ProcessingMessagePayload,
		orgID models.OrgID,
	) error
	ProcessQueuedJobs(ctx context.Context) error
	ProcessJobComplete(
		ctx context.Context,
		clientID string,
		payload models.JobCompletePayload,
		orgID models.OrgID,
	) error
	CleanupFailedSlackJob(
		ctx context.Context,
		job *models.Job,
		agentID string,
		message string,
	) error
	ProcessAssistantMessage(
		ctx context.Context,
		clientID string,
		payload models.AssistantMessagePayload,
		orgID models.OrgID,
	) error
	ProcessSystemMessage(
		ctx context.Context,
		clientID string,
		payload models.SystemMessagePayload,
		orgID models.OrgID,
	) error
}

// DiscordUseCaseInterface defines the interface for Discord use case operations
type DiscordUseCaseInterface interface {
	ProcessDiscordMessageEvent(
		ctx context.Context,
		event models.DiscordMessageEvent,
		discordIntegrationID string,
		orgID models.OrgID,
	) error
	ProcessDiscordReactionEvent(
		ctx context.Context,
		event models.DiscordReactionEvent,
		discordIntegrationID string,
		orgID models.OrgID,
	) error
	ProcessProcessingMessage(
		ctx context.Context,
		clientID string,
		payload models.ProcessingMessagePayload,
		orgID models.OrgID,
	) error
	ProcessSystemMessage(
		ctx context.Context,
		clientID string,
		payload models.SystemMessagePayload,
		orgID models.OrgID,
	) error
	ProcessJobComplete(
		ctx context.Context,
		clientID string,
		payload models.JobCompletePayload,
		orgID models.OrgID,
	) error
	ProcessAssistantMessage(
		ctx context.Context,
		clientID string,
		payload models.AssistantMessagePayload,
		orgID models.OrgID,
	) error
	CleanupFailedDiscordJob(
		ctx context.Context,
		job *models.Job,
		agentID string,
		message string,
	) error
	ProcessQueuedJobs(ctx context.Context) error
}