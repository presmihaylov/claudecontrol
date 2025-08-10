package discord

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

func (d *DiscordUseCase) ProcessDiscordMessageEvent(
	ctx context.Context,
	event models.DiscordMessageEvent,
	discordIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Discord message event from %s in %s: %s", event.User, event.ChannelID, event.Text)

	// For thread replies, validate that a job exists first (don't create new jobs)
	if event.ThreadID != "" {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadID, event.ChannelID)

		// Check if job exists for this thread - thread replies cannot create new jobs
		maybeJob, err := d.jobsService.GetJobByDiscordThread(
			ctx,
			event.ThreadID,
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
			return d.sendSystemMessage(ctx, discordIntegrationID, event.ChannelID, errorMessage)
		}
	} else {
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.ChannelID)
	}

	// Determine thread ID for job lookup/creation
	threadID := event.MessageID
	if event.ThreadID != "" {
		threadID = event.ThreadID
	}

	// Get or create job for this discord thread
	jobResult, err := d.jobsService.GetOrCreateJobForDiscordThread(
		ctx,
		threadID,
		event.ChannelID,
		event.User,
		discordIntegrationID,
		organizationID,
	)
	if err != nil {
		log.Printf("❌ Failed to get or create job for discord thread: %v", err)
		return fmt.Errorf("failed to get or create job for discord thread: %w", err)
	}

	job := jobResult.Job
	isNewConversation := jobResult.Status == models.JobCreationStatusCreated

	// Get organization ID from discord integration (agents are organization-scoped)
	maybeDiscordIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get discord integration: %v", err)
		return fmt.Errorf("failed to get discord integration: %w", err)
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
		event.ThreadID,
		event.Text,
		discordIntegrationID,
		organizationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed discord message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := DeriveMessageReactionFromStatus(messageStatus)
	if err := d.updateDiscordMessageReaction(ctx, event.ChannelID, event.MessageID, reactionEmoji, discordIntegrationID); err != nil {
		return fmt.Errorf("failed to update discord message reaction: %w", err)
	}

	// Always add eyes emoji to top-level message to show agent is processing
	if job.DiscordPayload == nil {
		return fmt.Errorf("job has no Discord payload")
	}
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, "👀", job.DiscordPayload.DiscordIntegrationID); err != nil {
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
	if isNewConversation {
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

func (d *DiscordUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageID, discordIntegrationID string,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to process reaction %s added by %s on message %s in channel %s",
		reactionName,
		userID,
		messageID,
		channelID,
	)

	// Only handle white check mark, check mark, or white tick reactions
	if reactionName != "white_check_mark" && reactionName != "heavy_check_mark" && reactionName != "white_tick" {
		log.Printf("⏭️ Ignoring reaction: %s (not a completion emoji)", reactionName)
		return nil
	}

	// Find the job by thread ID - the messageID is the thread root
	maybeJob, err := d.jobsService.GetJobByDiscordThread(ctx, messageID, discordIntegrationID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to get job for message %s in channel %s: %v", messageID, channelID, err)
		return fmt.Errorf("failed to get job for reaction: %w", err)
	}
	if !maybeJob.IsPresent() {
		// Job not found - this might be a reaction on a non-job message
		log.Printf("⏭️ No job found for message %s in channel %s - ignoring reaction", messageID, channelID)
		return nil
	}
	job := maybeJob.MustGet()

	// Check if the user who added the reaction is the same as the user who created the job
	if job.DiscordPayload == nil {
		log.Printf("⏭️ Job %s has no Discord payload", job.ID)
		return nil
	}
	if job.DiscordPayload.UserID != userID {
		log.Printf("⏭️ Reaction from %s ignored - job %s was created by %s", userID, job.ID, job.DiscordPayload.UserID)
		return nil
	}

	log.Printf("✅ Job completion reaction confirmed - user %s is the job creator", userID)

	// Get organization ID from discord integration (agents are organization-scoped)
	maybeDiscordIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get discord integration: %v", err)
		return fmt.Errorf("failed to get discord integration: %w", err)
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
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, "✅", job.DiscordPayload.DiscordIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update reaction for completed job %s: %v", job.ID, err)
		// Don't return error - this is not critical
	}

	// Send completion message to Discord thread
	if err := d.sendSystemMessage(ctx, job.DiscordPayload.DiscordIntegrationID, job.DiscordPayload.ChannelID, "Job manually marked as complete"); err != nil {
		log.Printf("❌ Failed to send completion message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
		return fmt.Errorf("failed to send completion message to Discord: %w", err)
	}

	log.Printf("📤 Sent completion message to Discord thread %s", job.DiscordPayload.ThreadID)
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

	// Update the discord message reaction to show agent is now processing (eyes emoji)
	if err := d.updateDiscordMessageReaction(ctx, processedMessage.DiscordMessageID, processedMessage.DiscordThreadID, "👀", discordIntegrationID); err != nil {
		return fmt.Errorf("failed to update discord message reaction to eyes: %w", err)
	}

	log.Printf("📋 Completed successfully - updated discord message emoji to eyes for message %s", messageID)
	return nil
}