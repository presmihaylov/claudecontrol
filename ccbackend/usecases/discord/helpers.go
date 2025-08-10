package discord

import (
	"context"
	"fmt"
	"log"

	"ccbackend/clients"
	discordclient "ccbackend/clients/discord"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/utils"
)

func (d *DiscordUseCase) getDiscordClientForIntegration(
	ctx context.Context,
	discordIntegrationID string,
) (clients.DiscordClient, error) {
	maybeDiscordInt, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get discord integration: %w", err)
	}
	if !maybeDiscordInt.IsPresent() {
		return nil, fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}

	// TODO: Need to store bot token in Discord integration and retrieve it here
	// For now, we'll use placeholder
	return discordclient.NewDiscordClient(nil, "placeholder-bot-token")
}

func (d *DiscordUseCase) sendStartConversationToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedDiscordMessage,
) error {
	// Get integration-specific Discord client
	_, err := d.getDiscordClientForIntegration(ctx, message.DiscordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord client for integration: %w", err)
	}

	// Get job to access thread ID
	maybeJob, err := d.jobsService.GetJobByID(ctx, message.JobID, message.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate message link for the thread's first message
	if job.DiscordPayload == nil {
		return fmt.Errorf("job has no Discord payload")
	}
	
	// Discord message link format: https://discord.com/channels/{guild_id}/{channel_id}/{message_id}
	messageLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", 
		job.DiscordPayload.GuildID, 
		job.DiscordPayload.ChannelID, 
		job.DiscordPayload.ThreadID)

	// TODO: Resolve user mentions in the message text before sending to agent
	resolvedText := message.TextContent // For now, no resolution

	startConversationMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        messageLink,
		},
	}

	if err := d.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
		return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
	}
	log.Printf("🚀 Sent start conversation message to client %s", clientID)
	return nil
}

func (d *DiscordUseCase) sendUserMessageToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedDiscordMessage,
) error {
	// Get integration-specific Discord client
	_, err := d.getDiscordClientForIntegration(ctx, message.DiscordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord client for integration: %w", err)
	}

	// Get job to access thread ID
	maybeJob, err := d.jobsService.GetJobByID(ctx, message.JobID, message.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate message link for the thread's first message
	if job.DiscordPayload == nil {
		return fmt.Errorf("job has no Discord payload")
	}
	
	// Discord message link format: https://discord.com/channels/{guild_id}/{channel_id}/{message_id}
	messageLink := fmt.Sprintf("https://discord.com/channels/%s/%s/%s", 
		job.DiscordPayload.GuildID, 
		job.DiscordPayload.ChannelID, 
		job.DiscordPayload.ThreadID)

	// TODO: Resolve user mentions in the message text before sending to agent
	resolvedText := message.TextContent // For now, no resolution

	userMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        messageLink,
		},
	}

	if err := d.wsClient.SendMessage(clientID, userMessage); err != nil {
		return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
	}
	log.Printf("💬 Sent user message to client %s", clientID)
	return nil
}

func (d *DiscordUseCase) updateDiscordMessageReaction(
	ctx context.Context,
	channelID, messageID, newEmoji, discordIntegrationID string,
) error {
	// Get integration-specific Discord client
	discordClient, err := d.getDiscordClientForIntegration(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord client for integration: %w", err)
	}

	// Add the new emoji reaction
	if newEmoji != "" {
		if err := discordClient.AddReaction(channelID, messageID, newEmoji); err != nil {
			return fmt.Errorf("failed to add %s reaction: %w", newEmoji, err)
		}
		log.Printf("👍 Added Discord reaction %s to message %s in channel %s", newEmoji, messageID, channelID)
	}

	return nil
}

func (d *DiscordUseCase) sendDiscordMessage(
	ctx context.Context,
	discordIntegrationID, channelID, message string,
) error {
	log.Printf("📋 Starting to send message to channel %s: %s", channelID, message)

	// Get integration-specific Discord client
	discordClient, err := d.getDiscordClientForIntegration(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord client for integration: %w", err)
	}

	// Send message to Discord
	// Note: Discord uses Markdown syntax similar to Slack, so we can convert
	discordMessage := utils.ConvertMarkdownToSlack(message)
	
	_, err = discordClient.SendMessage(channelID, discordMessage)
	if err != nil {
		return fmt.Errorf("failed to send message to Discord: %w", err)
	}

	log.Printf("📋 Completed successfully - sent message to channel %s", channelID)
	return nil
}

func (d *DiscordUseCase) sendSystemMessage(
	ctx context.Context,
	discordIntegrationID, channelID, message string,
) error {
	log.Printf("📋 Starting to send system message to channel %s: %s", channelID, message)

	// Prepend gear emoji to message
	systemMessage := "⚙️ " + message

	// Use the base sendDiscordMessage function
	return d.sendDiscordMessage(ctx, discordIntegrationID, channelID, systemMessage)
}

func DeriveMessageReactionFromStatus(status models.ProcessedDiscordMessageStatus) string {
	switch status {
	case models.ProcessedDiscordMessageStatusInProgress:
		return "⏳"
	case models.ProcessedDiscordMessageStatusQueued:
		return "⏳"
	case models.ProcessedDiscordMessageStatusCompleted:
		return "✅"
	default:
		utils.AssertInvariant(false, "invalid status received")
		return ""
	}
}