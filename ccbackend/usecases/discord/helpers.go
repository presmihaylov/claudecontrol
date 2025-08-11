package discord

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/utils"
)

func (d *DiscordUseCase) sendStartConversationToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedDiscordMessage,
) error {
	// Get job to access thread information
	maybeJob, err := d.jobsService.GetJobByID(ctx, message.OrgID, message.JobID)
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

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, message.DiscordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", message.DiscordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// For Discord message links, we need the channel ID where the original message was posted
	// The Discord job payload ThreadID contains either:
	// - For top-level messages: the original channel ID
	// - For thread messages: the thread channel ID
	// We need to get the channel info to determine the correct link structure

	// Try to get channel information to determine if this is a thread or regular channel
	var channelID string
	if message.DiscordThreadID == message.DiscordMessageID {
		// Top-level message case: use the job's ThreadID as the channel ID
		channelID = job.DiscordPayload.ThreadID
	} else {
		// Thread message case: ThreadID is the thread channel ID
		channelID = message.DiscordThreadID
	}

	messageLink := getDiscordMessageLink(integration.DiscordGuildID, channelID, job.DiscordPayload.MessageID)

	// Discord doesn't have user mention resolution like Slack, so we use the content as-is
	startConversationMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			JobID:              message.JobID,
			Message:            message.TextContent,
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
	// Get job to access thread information
	maybeJob, err := d.jobsService.GetJobByID(ctx, message.OrgID, message.JobID)
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

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, message.DiscordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", message.DiscordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// For Discord message links, determine the correct channel ID (same logic as sendStartConversationToAgent)
	var channelID string
	if message.DiscordThreadID == message.DiscordMessageID {
		// Top-level message case: use the job's ThreadID as the channel ID
		channelID = job.DiscordPayload.ThreadID
	} else {
		// Thread message case: ThreadID is the thread channel ID
		channelID = message.DiscordThreadID
	}

	messageLink := getDiscordMessageLink(integration.DiscordGuildID, channelID, job.DiscordPayload.MessageID)

	// Discord doesn't have user mention resolution like Slack, so we use the content as-is
	userMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			JobID:              message.JobID,
			Message:            message.TextContent,
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
	// For Discord, we'll implement a simpler approach than Slack
	// Remove old reactions and add the new one
	oldReactions := getOldDiscordReactions(newEmoji)
	for _, emoji := range oldReactions {
		// Try to remove old reactions - ignore errors since they might not exist
		_ = d.discordClient.RemoveReaction(channelID, messageID, emoji)
	}

	// Add new reaction if not empty
	if newEmoji != "" {
		if err := d.discordClient.AddReaction(channelID, messageID, newEmoji); err != nil {
			return fmt.Errorf("failed to add %s reaction: %w", newEmoji, err)
		}
	}

	return nil
}

func (d *DiscordUseCase) sendDiscordMessage(
	ctx context.Context,
	discordIntegrationID, guildID, channelID, threadID, message string,
) error {
	log.Printf("📋 Starting to send message to channel %s, thread %s: %s", channelID, threadID, message)

	// Trim message to Discord's 2000 character limit
	trimmedMessage := trimDiscordMessage(message)

	// Log if message was trimmed
	if len(message) > len(trimmedMessage) {
		log.Printf(
			"⚠️ Message trimmed from %d to %d characters for Discord API limits",
			len(message),
			len(trimmedMessage),
		)
	}

	// Send message to Discord
	params := clients.DiscordMessageParams{
		Content: trimmedMessage, // Discord natively supports markdown format
	}
	if threadID != "" && threadID != channelID {
		params.ThreadID = &threadID
	}
	_, err := d.discordClient.PostMessage(channelID, params)
	if err != nil {
		return fmt.Errorf("failed to send message to Discord: %w", err)
	}

	log.Printf("📋 Completed successfully - sent message to channel %s, thread %s", channelID, threadID)
	return nil
}

func (d *DiscordUseCase) sendSystemMessage(
	ctx context.Context,
	discordIntegrationID, guildID, channelID, threadID, message string,
) error {
	log.Printf("📋 Starting to send system message to channel %s, thread %s: %s", channelID, threadID, message)

	// Prepend gear emoji to message
	systemMessage := EmojiGear + " " + message

	// Use the base sendDiscordMessage function
	return d.sendDiscordMessage(ctx, discordIntegrationID, guildID, channelID, threadID, systemMessage)
}

func deriveMessageReactionFromStatus(status models.ProcessedDiscordMessageStatus) string {
	switch status {
	case models.ProcessedDiscordMessageStatusInProgress:
		return EmojiHourglass
	case models.ProcessedDiscordMessageStatusQueued:
		return EmojiHourglass
	case models.ProcessedDiscordMessageStatusCompleted:
		return EmojiCheckMark
	default:
		utils.AssertInvariant(false, "invalid status received")
		return ""
	}
}

// getDiscordMessageLink generates a Discord message link
func getDiscordMessageLink(guildID, channelID, messageID string) string {
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildID, channelID, messageID)
}

func getOldDiscordReactions(newEmoji string) []string {
	var result []string
	for _, reaction := range AllStatusEmojis {
		if reaction != newEmoji {
			result = append(result, reaction)
		}
	}

	return result
}

// isAgentErrorMessage determines if a system message from ccagent indicates an error or failure
func isAgentErrorMessage(message string) bool {
	// Check if message starts with the specific error prefix from ccagent
	return strings.HasPrefix(message, "ccagent encountered error:")
}

// trimDiscordMessage trims a Discord message to the 2000 character limit
// Discord has a hard limit of 2000 characters per message via the API
func trimDiscordMessage(message string) string {
	const discordMessageLimit = 2000

	if len(message) <= discordMessageLimit {
		return message
	}

	// Trim to 2000 characters and add ellipsis to indicate truncation
	const truncationSuffix = "..."
	trimmedLength := discordMessageLimit - len(truncationSuffix)

	if trimmedLength <= 0 {
		// Edge case: if somehow the suffix is longer than the limit
		return message[:discordMessageLimit]
	}

	return message[:trimmedLength] + truncationSuffix
}
