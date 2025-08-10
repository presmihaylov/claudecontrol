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

	// Send message to Discord
	params := clients.DiscordMessageParams{
		Content: message, // Discord natively supports markdown format
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

func DeriveMessageReactionFromStatus(status models.ProcessedDiscordMessageStatus) string {
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

// ProcessAssistantMessage handles assistant messages from agents and updates Discord accordingly
func (d *DiscordUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process assistant message from client %s", clientID)

	// Get the agent by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}
	agent := maybeAgent.MustGet()

	// Get the specific job from the payload to find the Discord thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in AssistantMessage payload")

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := d.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf(
			"⚠️ Job %s not found - already completed manually or by another agent, skipping assistant message",
			jobID,
		)
		return nil
	}

	job := maybeJob.MustGet()
	if job.DiscordPayload == nil {
		log.Printf("⚠️ Job %s has no Discord payload, skipping assistant message", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Validate that this agent is actually assigned to this job
	if err := d.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		return err
	}

	log.Printf(
		"📤 Sending assistant message to Discord thread %s",
		job.DiscordPayload.ThreadID,
	)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// Send assistant message to Discord - ThreadID contains the channel/thread info
	if err := d.sendDiscordMessage(
		ctx,
		discordIntegrationID,
		integration.DiscordGuildID,
		job.DiscordPayload.ThreadID,
		job.DiscordPayload.ThreadID,
		messageToSend,
	); err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Discord: %v", err)
	}

	// Update job timestamp to track activity
	if err := d.jobsService.UpdateJobTimestamp(ctx, job.ID, organizationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	// Update the ProcessedDiscordMessage status to COMPLETED
	utils.AssertInvariant(payload.ProcessedMessageID != "", "ProcessedMessageID is empty")

	messageID := payload.ProcessedMessageID

	updatedMessage, err := d.discordMessagesService.UpdateProcessedDiscordMessage(
		ctx,
		messageID,
		models.ProcessedDiscordMessageStatusCompleted,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed discord message status: %w", err)
	}

	// Add completed emoji reaction
	// For top-level messages (where DiscordMessageID equals DiscordThreadID), only set white_check_mark on job completion
	// For other messages, set white_check_mark immediately when processed
	isTopLevelMessage := updatedMessage.DiscordMessageID == job.DiscordPayload.MessageID
	if !isTopLevelMessage {
		reactionEmoji := DeriveMessageReactionFromStatus(models.ProcessedDiscordMessageStatusCompleted)
		if err := d.updateDiscordMessageReaction(
			ctx,
			updatedMessage.DiscordThreadID,
			updatedMessage.DiscordMessageID,
			reactionEmoji,
			discordIntegrationID,
		); err != nil {
			return fmt.Errorf("failed to update discord message reaction: %w", err)
		}
	}

	// Check if this is the latest message in the job and add hand emoji if waiting for next steps
	maybeLatestMsg, err := d.discordMessagesService.GetLatestProcessedMessageForJob(
		ctx,
		job.ID,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get latest message for job: %w", err)
	}

	if maybeLatestMsg.IsPresent() && maybeLatestMsg.MustGet().ID == messageID {
		// This is the latest message - agent is done processing, add hand emoji to top-level message
		if err := d.updateDiscordMessageReaction(
			ctx,
			job.DiscordPayload.ChannelID,
			job.DiscordPayload.MessageID,
			EmojiRaisedHand,
			discordIntegrationID,
		); err != nil {
			log.Printf("⚠️ Failed to add hand emoji to job %s thread: %v", job.ID, err)
			return fmt.Errorf("failed to add hand emoji to job thread: %w", err)
		}
		log.Printf("✋ Added hand emoji to job %s - agent waiting for next steps", job.ID)
	}

	log.Printf("📋 Completed successfully - sent assistant message to Discord thread %s", job.DiscordPayload.ThreadID)
	return nil
}

// CleanupFailedDiscordJob handles the cleanup of a failed Discord job including Discord notifications and database cleanup
// This is exported so core use case can call it when deregistering agents
func (d *DiscordUseCase) CleanupFailedDiscordJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", job.ID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID
	organizationID := job.OrganizationID

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get Discord integration: %v", err)
		// Continue with cleanup even if integration lookup fails
	}

	var guildID string
	if maybeIntegration.IsPresent() {
		guildID = maybeIntegration.MustGet().DiscordGuildID
	}

	// Send failure message to Discord thread
	if guildID != "" {
		if err := d.sendSystemMessage(ctx, discordIntegrationID, guildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, failureMessage); err != nil {
			log.Printf("❌ Failed to send failure message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
			// Continue with cleanup even if Discord message fails
		}
	}

	// Update the top-level message emoji to ❌
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiCrossMark, discordIntegrationID); err != nil {
		log.Printf("❌ Failed to update discord message reaction to ❌ for failed job %s: %v", job.ID, err)
		// Continue with cleanup even if reaction update fails
	}

	// Perform database operations within transaction
	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent ID is provided, unassign agent from job
		if agentID != "" {
			if err := d.agentsService.UnassignAgentFromJob(ctx, agentID, job.ID, organizationID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}
			log.Printf("🔗 Unassigned agent %s from job %s", agentID, job.ID)
		}

		// Delete the job (use the job's discord integration and organization from the job)
		if err := d.jobsService.DeleteJob(ctx, job.ID, organizationID); err != nil {
			log.Printf("❌ Failed to delete job %s: %v", job.ID, err)
			return fmt.Errorf("failed to delete job: %w", err)
		}

		log.Printf("🗑️ Deleted job %s", job.ID)
		return nil
	}); err != nil {
		return fmt.Errorf("failed to cleanup job %s in transaction: %w", job.ID, err)
	}

	return nil
}

// IsAgentErrorMessage determines if a system message from ccagent indicates an error or failure
func IsAgentErrorMessage(message string) bool {
	// Check if message starts with the specific error prefix from ccagent
	return strings.HasPrefix(message, "ccagent encountered error:")
}

// ProcessSystemMessage handles system messages from agents and sends them to Discord
func (d *DiscordUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process system message from client %s: %s", clientID, payload.Message)
	jobID := payload.JobID
	maybeJob, err := d.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s: %v", jobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf(
			"⚠️ Job %s not found - already completed manually or by another agent, skipping system message",
			jobID,
		)
		return nil
	}
	job := maybeJob.MustGet()
	if job.DiscordPayload == nil {
		log.Printf("⚠️ Job %s has no Discord payload, skipping system message", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Check if this is an error message from the agent
	if IsAgentErrorMessage(payload.Message) {
		log.Printf("❌ Detected agent error message for job %s: %s", job.ID, payload.Message)

		// Get the agent that encountered the error
		maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
		if err != nil {
			log.Printf("❌ Failed to find agent for error handling: %v", err)
			return fmt.Errorf("failed to find agent for error handling: %w", err)
		}

		var agentID string
		if maybeAgent.IsPresent() {
			agentID = maybeAgent.MustGet().ID
		}

		// Clean up the failed job
		errorMessage := fmt.Sprintf(
			"%s Agent encountered an error and cannot continue:\n%s",
			EmojiCrossMark,
			payload.Message,
		)
		if err := d.CleanupFailedDiscordJob(ctx, job, agentID, errorMessage); err != nil {
			return fmt.Errorf("failed to cleanup failed job: %w", err)
		}

		log.Printf("📋 Completed error handling - cleaned up failed job %s", job.ID)
		return nil
	}

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	log.Printf(
		"📤 Sending system message to Discord thread %s",
		job.DiscordPayload.ThreadID,
	)

	// Send system message (gear emoji will be added automatically)
	if err := d.sendSystemMessage(ctx, discordIntegrationID, integration.DiscordGuildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, payload.Message); err != nil {
		return fmt.Errorf("❌ Failed to send system message to Discord: %v", err)
	}

	// Update job timestamp to track activity
	if err := d.jobsService.UpdateJobTimestamp(ctx, job.ID, organizationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - sent system message to Discord thread %s", job.DiscordPayload.ThreadID)
	return nil
}

// ProcessJobComplete handles job completion from an agent
func (d *DiscordUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to process job complete from client %s: JobID: %s, Reason: %s",
		clientID,
		payload.JobID,
		payload.Reason,
	)

	jobID := payload.JobID
	maybeJob, err := d.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed manually or by another agent, skipping", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Get the agent by WebSocket connection ID to verify ownership (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}
	agent := maybeAgent.MustGet()

	// Validate that this agent is actually assigned to this job
	if err := d.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiCheckMark, discordIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Perform database operations within transaction
	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := d.agentsService.UnassignAgentFromJob(ctx, agent.ID, jobID, organizationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

		// Delete the job and its associated processed messages
		if err := d.jobsService.DeleteJob(ctx, jobID, organizationID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}
		log.Printf("🗑️ Deleted completed job %s", jobID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete job processing in transaction: %w", err)
	}

	// Get Discord integration to get guild ID for sending system message
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// Send completion message to Discord thread with reason
	if err := d.sendSystemMessage(ctx, discordIntegrationID, integration.DiscordGuildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, payload.Reason); err != nil {
		log.Printf("❌ Failed to send completion message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
		return fmt.Errorf("failed to send completion message to Discord: %w", err)
	}

	log.Printf("📤 Sent completion message to Discord thread %s: %s", job.DiscordPayload.ThreadID, payload.Reason)
	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}
