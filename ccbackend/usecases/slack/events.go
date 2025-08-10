package slack

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

func (s *SlackUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to process Slack message event from %s in %s: %s", event.User, event.Channel, event.Text)

	// For thread replies, validate that a job exists first (don't create new jobs)
	if event.ThreadTS != "" {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadTS, event.Channel)

		// Check if job exists for this thread - thread replies cannot create new jobs
		maybeJob, err := s.jobsService.GetJobBySlackThread(
			ctx,
			event.ThreadTS,
			event.Channel,
			slackIntegrationID,
			organizationID,
		)
		if err != nil {
			// Error occurred - propagate upstream
			log.Printf("❌ Failed to get job for thread reply in %s: %v", event.Channel, err)
			return fmt.Errorf("failed to get job for thread reply: %w", err)
		}
		if !maybeJob.IsPresent() {
			// Job not found for thread reply - send error message
			log.Printf("❌ No existing job found for thread reply in %s", event.Channel)
			errorMessage := "Error: new jobs can only be started from top-level messages"
			return s.sendSystemMessage(ctx, slackIntegrationID, event.Channel, event.TS, errorMessage)
		}
	} else {
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.Channel)
	}

	// Determine thread timestamp for job lookup/creation
	threadTS := event.TS
	if event.ThreadTS != "" {
		threadTS = event.ThreadTS
	}

	// Get or create job for this slack thread
	jobResult, err := s.jobsService.GetOrCreateJobForSlackThread(
		ctx,
		threadTS,
		event.Channel,
		event.User,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		log.Printf("❌ Failed to get or create job for slack thread: %v", err)
		return fmt.Errorf("failed to get or create job for slack thread: %w", err)
	}

	job := jobResult.Job
	isNewConversation := jobResult.Status == models.JobCreationStatusCreated

	// Get organization ID from slack integration (agents are organization-scoped)
	maybeSlackIntegration, err := s.slackIntegrationsService.GetSlackIntegrationByID(ctx, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get slack integration: %v", err)
		return fmt.Errorf("failed to get slack integration: %w", err)
	}
	if !maybeSlackIntegration.IsPresent() {
		log.Printf("❌ Slack integration not found: %s", slackIntegrationID)
		return fmt.Errorf("slack integration not found: %s", slackIntegrationID)
	}
	// Verify the organization ID matches (already passed as parameter)

	// Check if agents are available first
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("📋 Retrieved %d active client IDs", len(connectedClientIDs))
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to check for connected agents: %v", err)
		return fmt.Errorf("failed to check for connected agents: %w", err)
	}

	var clientID string
	var messageStatus models.ProcessedSlackMessageStatus

	if len(connectedAgents) == 0 {
		// No agents available - queue the message
		log.Printf("⚠️ No available agents to handle Slack mention - queuing message")
		messageStatus = models.ProcessedSlackMessageStatusQueued
		clientID = "" // No agent assigned
	} else {
		// Agents available - assign job to agent
		clientID, err = s.agentsUseCase.GetOrAssignAgentForJob(ctx, job, threadTS, organizationID)
		if err != nil {
			return fmt.Errorf("failed to get or assign agent for job: %w", err)
		}
		messageStatus = models.ProcessedSlackMessageStatusInProgress
	}

	// Store the Slack message as ProcessedSlackMessage with appropriate status
	processedMessage, err := s.slackMessagesService.CreateProcessedSlackMessage(
		ctx,
		job.ID,
		event.Channel,
		event.TS,
		event.Text,
		slackIntegrationID,
		organizationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := DeriveMessageReactionFromStatus(messageStatus)
	if err := s.updateSlackMessageReaction(ctx, processedMessage.SlackChannelID, processedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update slack message reaction: %w", err)
	}

	// Always add eyes emoji to top-level message to show agent is processing
	if job.SlackPayload == nil {
		return fmt.Errorf("job has no Slack payload")
	}
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "eyes", slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update top-level message reaction: %w", err)
	}
	log.Printf("👀 Updated top-level message with eyes emoji for job %s - agent processing message", job.ID)

	// If message was queued, don't send to agent yet - background processor will handle it
	if messageStatus == models.ProcessedSlackMessageStatusQueued {
		log.Printf("📋 Message queued for background processing - job %s", job.ID)
		log.Printf("📋 Completed successfully - processed Slack message event (queued)")
		return nil
	}

	// Send work to assigned agent
	if isNewConversation {
		if err := s.sendStartConversationToAgent(ctx, clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message: %w", err)
		}
	} else {
		if err := s.sendUserMessageToAgent(ctx, clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send user message: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - processed Slack message event")
	return nil
}

func (s *SlackUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to process reaction %s added by %s on message %s in channel %s",
		reactionName,
		userID,
		messageTS,
		channelID,
	)

	// Only handle white check mark, check mark, or white tick reactions
	if reactionName != "white_check_mark" && reactionName != "heavy_check_mark" && reactionName != "white_tick" {
		log.Printf("⏭️ Ignoring reaction: %s (not a completion emoji)", reactionName)
		return nil
	}

	// Find the job by thread TS and channel - the messageTS is the thread root
	maybeJob, err := s.jobsService.GetJobBySlackThread(ctx, messageTS, channelID, slackIntegrationID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to get job for message %s in channel %s: %v", messageTS, channelID, err)
		return fmt.Errorf("failed to get job for reaction: %w", err)
	}
	if !maybeJob.IsPresent() {
		// Job not found - this might be a reaction on a non-job message
		log.Printf("⏭️ No job found for message %s in channel %s - ignoring reaction", messageTS, channelID)
		return nil
	}
	job := maybeJob.MustGet()

	// Check if the user who added the reaction is the same as the user who created the job
	if job.SlackPayload == nil {
		log.Printf("⏭️ Job %s has no Slack payload", job.ID)
		return nil
	}
	if job.SlackPayload.UserID != userID {
		log.Printf("⏭️ Reaction from %s ignored - job %s was created by %s", userID, job.ID, job.SlackPayload.UserID)
		return nil
	}

	log.Printf("✅ Job completion reaction confirmed - user %s is the job creator", userID)

	// Get organization ID from slack integration (agents are organization-scoped)
	maybeSlackIntegration, err := s.slackIntegrationsService.GetSlackIntegrationByID(ctx, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get slack integration: %v", err)
		return fmt.Errorf("failed to get slack integration: %w", err)
	}
	if !maybeSlackIntegration.IsPresent() {
		log.Printf("❌ Slack integration not found: %s", slackIntegrationID)
		return fmt.Errorf("slack integration not found: %s", slackIntegrationID)
	}
	// Verify the organization ID matches (already passed as parameter)

	// Get the assigned agent for this job to unassign them
	maybeAgent, err := s.agentsService.GetAgentByJobID(ctx, job.ID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to get agent by job id: %w", err)
	}

	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent is found, unassign them from the job
		if maybeAgent.IsPresent() {
			agent := maybeAgent.MustGet()
			if err := s.agentsService.UnassignAgentFromJob(ctx, agent.ID, job.ID, organizationID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}

			log.Printf("✅ Unassigned agent %s from manually completed job %s", agent.ID, job.ID)
		}

		// Delete the job and its associated processed messages
		if err := s.jobsService.DeleteJob(ctx, job.ID, slackIntegrationID, organizationID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", job.ID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete manual job completion in transaction: %w", err)
	}

	// Update Slack reactions - remove eyes emoji and add white_check_mark
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update reaction for completed job %s: %v", job.ID, err)
		// Don't return error - this is not critical
	}

	// Send completion message to Slack thread
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "Job manually marked as complete"); err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackPayload.ThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s", job.SlackPayload.ThreadTS)
	log.Printf("🗑️ Deleted manually completed job %s", job.ID)
	log.Printf("📋 Completed successfully - processed manual job completion for job %s", job.ID)
	return nil
}

func (s *SlackUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process processing slack message notification from client %s", clientID)

	messageID := payload.ProcessedMessageID

	// Get processed slack message directly using organization_id (optimization)
	maybeMessage, err := s.slackMessagesService.GetProcessedSlackMessageByID(
		ctx,
		messageID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get processed slack message: %w", err)
	}
	if !maybeMessage.IsPresent() {
		log.Printf(
			"⚠️ Processed slack message %s not found - job may have been completed manually, skipping processing message",
			messageID,
		)
		return nil
	}

	processedMessage := maybeMessage.MustGet()
	slackIntegrationID := processedMessage.SlackIntegrationID

	// Update the slack message reaction to show agent is now processing (eyes emoji)
	if err := s.updateSlackMessageReaction(ctx, processedMessage.SlackChannelID, processedMessage.SlackTS, "eyes", slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update slack message reaction to eyes: %w", err)
	}

	log.Printf("📋 Completed successfully - updated slack message emoji to eyes for message %s", messageID)
	return nil
}
