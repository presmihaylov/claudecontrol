package discord

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"slices"

	"ccbackend/models"
)

func (d *DiscordUseCase) ProcessDiscordMessageEvent(
	ctx context.Context,
	event models.DiscordMessageEvent,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Discord message event from user %s in guild %s, channel %s",
		event.UserID, event.GuildID, event.ChannelID)

	// Step 1: Get bot user information
	botUser, err := d.discordClient.GetBotUser()
	if err != nil {
		log.Printf("❌ Failed to get bot user: %v", err)
		return err
	}
	log.Printf("🤖 Bot user retrieved: %s (%s)", botUser.Username, botUser.ID)

	// Step 2: Check if bot was mentioned
	botMentioned := slices.Contains(event.Mentions, botUser.ID)

	if !botMentioned {
		log.Printf("🔍 Bot not mentioned in message from user %s - ignoring message", event.UserID)
		return nil
	}

	log.Printf("🤖 Bot %s (%s) mentioned in message from user %s", botUser.Username, botUser.ID, event.UserID)

	// For thread replies, validate that a job exists first (don't create new jobs)
	if event.ThreadID != nil {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", *event.ThreadID, event.ChannelID)

		// Check if job exists for this thread - thread replies cannot create new jobs
		maybeJob, err := d.jobsService.GetJobByDiscordThread(
			ctx,
			*event.ThreadID,
			discordIntegrationID,
			organizationID,
		)
		if err != nil {
			// Error occurred - propagate upstream
			log.Printf("❌ Failed to get job for thread reply in %s: %v", event.ChannelID, err)
			return fmt.Errorf("failed to get job for thread reply: %w", err)
		}
		if !maybeJob.IsPresent() {
			// Job not found for thread reply - send error message
			log.Printf("❌ No existing job found for thread reply in %s", event.ChannelID)
			errorMessage := "Error: new jobs can only be started from top-level messages"
			return d.sendSystemMessage(
				ctx,
				discordIntegrationID,
				event.GuildID,
				event.ChannelID,
				*event.ThreadID,
				errorMessage,
			)
		}
	} else {
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.ChannelID)
	}

	// Determine thread ID for job lookup/creation
	var threadID string
	if event.ThreadID != nil {
		// Reply in existing thread - use the thread ID
		threadID = *event.ThreadID
	} else {
		// New conversation - create a public thread from the message
		log.Printf("🧵 Creating new Discord thread for message %s in channel %s", event.MessageID, event.ChannelID)

		//nolint:gosec // We don't care about using secure random numbers here
		randomNumber := rand.Intn(9000) + 1000 // Generates number between 1000-9999
		threadName := fmt.Sprintf("CC Sesh #%d", randomNumber)

		threadResponse, err := d.discordClient.CreatePublicThread(event.ChannelID, event.MessageID, threadName)
		if err != nil {
			log.Printf("❌ Failed to create Discord thread: %v", err)
			return fmt.Errorf("failed to create Discord thread: %w", err)
		}

		threadID = threadResponse.ThreadID
		log.Printf("✅ Created Discord thread %s with name '%s'", threadID, threadResponse.ThreadName)
	}

	// Get or create job for this Discord thread
	jobResult, err := d.jobsService.GetOrCreateJobForDiscordThread(
		ctx,
		event.MessageID,
		event.ChannelID,
		threadID,
		event.UserID,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		log.Printf("❌ Failed to get or create job for Discord thread: %v", err)
		return fmt.Errorf("failed to get or create job for Discord thread: %w", err)
	}

	job := jobResult.Job

	// Get organization ID from Discord integration (agents are organization-scoped)
	maybeDiscordIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get Discord integration: %v", err)
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeDiscordIntegration.IsPresent() {
		log.Printf("❌ Discord integration not found: %s", discordIntegrationID)
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	// Verify the organization ID matches (already passed as parameter)

	// Check if agents are available first
	connectedClientIDs := d.wsClient.GetClientIDs()
	log.Printf("📋 Retrieved %d active client IDs", len(connectedClientIDs))
	connectedAgents, err := d.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to check for connected agents: %v", err)
		return fmt.Errorf("failed to check for connected agents: %w", err)
	}

	var clientID string
	var messageStatus models.ProcessedDiscordMessageStatus

	if len(connectedAgents) == 0 {
		// No agents available - queue the message
		log.Printf("⚠️ No available agents to handle Discord mention - queuing message")
		messageStatus = models.ProcessedDiscordMessageStatusQueued
		clientID = "" // No agent assigned
	} else {
		// Agents available - assign job to agent
		clientID, err = d.agentsUseCase.GetOrAssignAgentForJob(ctx, job, threadID, organizationID)
		if err != nil {
			return fmt.Errorf("failed to get or assign agent for job: %w", err)
		}
		messageStatus = models.ProcessedDiscordMessageStatusInProgress
	}

	// Store the Discord message as ProcessedDiscordMessage with appropriate status
	processedMessage, err := d.discordMessagesService.CreateProcessedDiscordMessage(
		ctx,
		job.ID,
		event.MessageID,
		threadID,
		event.Content,
		discordIntegrationID,
		organizationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed Discord message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := DeriveMessageReactionFromStatus(messageStatus)
	if err := d.updateDiscordMessageReaction(ctx, event.ChannelID, processedMessage.DiscordMessageID, reactionEmoji, discordIntegrationID); err != nil {
		return fmt.Errorf("failed to update Discord message reaction: %w", err)
	}

	// Always add eyes emoji to top-level message to show agent is processing
	if job.DiscordPayload == nil {
		return fmt.Errorf("job has no Discord payload")
	}

	// For Discord reactions, we always want to react to the original message in the original channel
	// job.DiscordPayload.MessageID contains the original message ID that triggered the job
	// job.DiscordPayload.ChannelID contains the channel where the original message was posted (for threads, this is the parent channel)
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiEyes, discordIntegrationID); err != nil {
		return fmt.Errorf("failed to update top-level message reaction: %w", err)
	}
	log.Printf("👀 Updated top-level message with eyes emoji for job %s - agent processing message", job.ID)

	// If message was queued, don't send to agent yet - background processor will handle it
	if messageStatus == models.ProcessedDiscordMessageStatusQueued {
		log.Printf("📋 Message queued for background processing - job %s", job.ID)
		log.Printf("📋 Completed successfully - processed Discord message event (queued)")
		return nil
	}

	// Send work to assigned agent
	if event.ThreadID == nil {
		if err := d.sendStartConversationToAgent(ctx, clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message: %w", err)
		}
	} else {
		if err := d.sendUserMessageToAgent(ctx, clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send user message: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - processed Discord message event")
	return nil
}

func (d *DiscordUseCase) ProcessDiscordReactionEvent(
	ctx context.Context,
	event models.DiscordReactionEvent,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Discord reaction event: %s by user %s on message %s in guild %s, channel %s",
		event.EmojiName, event.UserID, event.MessageID, event.GuildID, event.ChannelID)

	// Only handle white check mark, check mark, or similar completion reactions
	if event.EmojiName != EmojiCheckMark && event.EmojiName != "white_check_mark" &&
		event.EmojiName != "heavy_check_mark" {
		log.Printf("⏭️ Ignoring reaction: %s (not a completion emoji)", event.EmojiName)
		return nil
	}

	log.Printf("✅ Completion reaction detected: %s by user %s on message %s",
		event.EmojiName, event.UserID, event.MessageID)

	// Determine thread ID for job lookup
	threadID := event.MessageID
	if event.ThreadID != nil {
		threadID = *event.ThreadID
	}

	// Find the job by thread ID and channel - the messageID is the thread root
	maybeJob, err := d.jobsService.GetJobByDiscordThread(ctx, threadID, discordIntegrationID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to get job for message %s in channel %s: %v", event.MessageID, event.ChannelID, err)
		return fmt.Errorf("failed to get job for reaction: %w", err)
	}
	if !maybeJob.IsPresent() {
		// Job not found - this might be a reaction on a non-job message
		log.Printf("⏭️ No job found for message %s in channel %s - ignoring reaction", event.MessageID, event.ChannelID)
		return nil
	}
	job := maybeJob.MustGet()

	// Check if the user who added the reaction is the same as the user who created the job
	if job.DiscordPayload == nil {
		log.Printf("⏭️ Job %s has no Discord payload", job.ID)
		return nil
	}
	if job.DiscordPayload.UserID != event.UserID {
		log.Printf(
			"⏭️ Reaction from %s ignored - job %s was created by %s",
			event.UserID,
			job.ID,
			job.DiscordPayload.UserID,
		)
		return nil
	}

	log.Printf("✅ Job completion reaction confirmed - user %s is the job creator", event.UserID)

	// Get organization ID from Discord integration (agents are organization-scoped)
	maybeDiscordIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get Discord integration: %v", err)
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeDiscordIntegration.IsPresent() {
		log.Printf("❌ Discord integration not found: %s", discordIntegrationID)
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	// Verify the organization ID matches (already passed as parameter)

	// Get the assigned agent for this job to unassign them
	maybeAgent, err := d.agentsService.GetAgentByJobID(ctx, job.ID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to get agent by job id: %w", err)
	}

	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent is found, unassign them from the job
		if maybeAgent.IsPresent() {
			agent := maybeAgent.MustGet()
			if err := d.agentsService.UnassignAgentFromJob(ctx, agent.ID, job.ID, organizationID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}

			log.Printf("✅ Unassigned agent %s from manually completed job %s", agent.ID, job.ID)
		}

		// Delete the job and its associated processed messages
		if err := d.jobsService.DeleteJob(ctx, job.ID, organizationID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", job.ID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete manual job completion in transaction: %w", err)
	}

	// Update Discord reactions - remove eyes emoji and add white_check_mark
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiCheckMark, discordIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update reaction for completed job %s: %v", job.ID, err)
		// Don't return error - this is not critical
	}

	// Send completion message to Discord thread
	threadChannelID := job.DiscordPayload.ThreadID
	if event.ThreadID != nil {
		threadChannelID = *event.ThreadID
	}

	if err := d.sendSystemMessage(ctx, discordIntegrationID, event.GuildID, event.ChannelID, threadChannelID, "Job manually marked as complete"); err != nil {
		log.Printf("❌ Failed to send completion message to Discord thread %s: %v", threadChannelID, err)
		return fmt.Errorf("failed to send completion message to Discord: %w", err)
	}

	log.Printf("📤 Sent completion message to Discord thread %s", threadChannelID)
	log.Printf("🗑️ Deleted manually completed job %s", job.ID)
	log.Printf("📋 Completed successfully - processed manual job completion for job %s", job.ID)
	return nil
}

func (d *DiscordUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process processing discord message notification from client %s", clientID)

	messageID := payload.ProcessedMessageID

	// Get processed discord message directly using organization_id (optimization)
	maybeMessage, err := d.discordMessagesService.GetProcessedDiscordMessageByID(
		ctx,
		messageID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get processed discord message: %w", err)
	}
	if !maybeMessage.IsPresent() {
		log.Printf(
			"⚠️ Processed discord message %s not found - job may have been completed manually, skipping processing message",
			messageID,
		)
		return nil
	}

	processedMessage := maybeMessage.MustGet()
	discordIntegrationID := processedMessage.DiscordIntegrationID

	// Get job to determine if this is a top-level message
	maybeJob, err := d.jobsService.GetJobByID(ctx, processedMessage.JobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - may have been completed manually", processedMessage.JobID)
		return nil
	}
	job := maybeJob.MustGet()

	// Determine if this is the top-level message (original message that started the job)
	isTopLevelMessage := processedMessage.DiscordMessageID == job.DiscordPayload.MessageID

	var reactionChannelID, reactionMessageID string
	if isTopLevelMessage {
		// For top-level message, use original channel and message from job payload
		reactionChannelID = job.DiscordPayload.ChannelID
		reactionMessageID = job.DiscordPayload.MessageID
	} else {
		// For thread messages, use thread channel and message from processed message
		reactionChannelID = processedMessage.DiscordThreadID
		reactionMessageID = processedMessage.DiscordMessageID
	}

	// Update the discord message reaction to show agent is now processing (eyes emoji)
	if err := d.updateDiscordMessageReaction(ctx, reactionChannelID, reactionMessageID, EmojiEyes, discordIntegrationID); err != nil {
		return fmt.Errorf("failed to update discord message reaction to eyes: %w", err)
	}

	log.Printf("📋 Completed successfully - updated discord message emoji to eyes for message %s", messageID)
	return nil
}
