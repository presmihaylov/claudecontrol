package discord

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

// ProcessQueuedJobs processes jobs that are queued waiting for available agents
func (d *DiscordUseCase) ProcessQueuedJobs(ctx context.Context) error {
	log.Printf("📋 Starting to process queued Discord jobs")

	// Get all discord integrations
	integrations, err := d.discordIntegrationsService.GetAllDiscordIntegrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get discord integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No discord integrations found")
		return nil
	}

	totalProcessedJobs := 0

	for _, integration := range integrations {
		discordIntegrationID := integration.ID

		// Get jobs with queued messages for this integration
		queuedJobs, err := d.jobsService.GetJobsWithQueuedMessages(
			ctx,
			models.JobTypeDiscord,
			discordIntegrationID,
			integration.OrganizationID,
		)
		if err != nil {
			return fmt.Errorf("failed to get queued jobs for integration %s: %w", discordIntegrationID, err)
		}

		if len(queuedJobs) == 0 {
			continue
		}

		log.Printf("🔍 Found %d jobs with queued messages for integration %s", len(queuedJobs), discordIntegrationID)

		// Try to assign each queued job to an available agent
		for _, job := range queuedJobs {
			log.Printf("🔄 Processing queued job %s", job.ID)

			// Get organization ID for this integration
			organizationID := integration.OrganizationID

			// Try to assign job to an available agent
			clientID, assigned, err := d.agentsUseCase.TryAssignJobToAgent(ctx, job.ID, organizationID)
			if err != nil {
				return fmt.Errorf("failed to assign queued job %s: %w", job.ID, err)
			}

			if !assigned {
				log.Printf("⚠️ Still no agents available for queued job %s", job.ID)
				continue
			}

			// Job was successfully assigned - get queued messages and send them to agent
			queuedMessages, err := d.discordMessagesService.GetProcessedMessagesByJobIDAndStatus(
				ctx,
				job.ID,
				models.ProcessedDiscordMessageStatusQueued,
				discordIntegrationID,
				integration.OrganizationID,
			)
			if err != nil {
				return fmt.Errorf("failed to get queued messages for job %s: %w", job.ID, err)
			}

			log.Printf("📨 Found %d queued messages for job %s", len(queuedMessages), job.ID)

			// Process each queued message
			for _, message := range queuedMessages {
				// Update message status to IN_PROGRESS
				updatedMessage, err := d.discordMessagesService.UpdateProcessedDiscordMessage(
					ctx,
					message.ID,
					models.ProcessedDiscordMessageStatusInProgress,
					discordIntegrationID,
					organizationID,
				)
				if err != nil {
					return fmt.Errorf("failed to update message %s status: %w", message.ID, err)
				}

				// Determine if this is the first message in the job (new conversation)
				// Check if this message's ID matches the job's message ID (i.e., it's the top-level message)
				isNewConversation := false
				if job.DiscordPayload != nil {
					isNewConversation = updatedMessage.DiscordMessageID == job.DiscordPayload.MessageID
				}

				// Update Discord reaction to show processing (eyes emoji)
				// For top-level messages, use the original channel and message ID from job payload
				// For reply messages, use the thread and message ID from the processed message
				var reactionChannelID, reactionMessageID string
				if isNewConversation {
					// Top-level message: use original channel and message ID
					reactionChannelID = job.DiscordPayload.ChannelID
					reactionMessageID = job.DiscordPayload.MessageID
				} else {
					// Reply message: use thread and message ID
					reactionChannelID = updatedMessage.DiscordThreadID
					reactionMessageID = updatedMessage.DiscordMessageID
				}

				if err := d.updateDiscordMessageReaction(ctx, reactionChannelID, reactionMessageID, EmojiEyes, discordIntegrationID); err != nil {
					return fmt.Errorf("failed to update discord reaction for message %s: %w", message.ID, err)
				}

				// Send work to assigned agent
				if isNewConversation {
					log.Printf("📬 Sending start conversation message for job %s to client %s", job.ID, clientID)
					if err := d.sendStartConversationToAgent(ctx, clientID, updatedMessage); err != nil {
						return fmt.Errorf("failed to send start conversation for message %s: %w", message.ID, err)
					}
				} else {
					log.Printf("📬 Sending user message %s to client %s", message.ID, clientID)
					if err := d.sendUserMessageToAgent(ctx, clientID, updatedMessage); err != nil {
						return fmt.Errorf("failed to send user message %s: %w", message.ID, err)
					}
				}

				log.Printf("✅ Successfully assigned and sent queued message %s to agent", message.ID)
			}

			totalProcessedJobs++
			log.Printf("✅ Successfully processed queued job %s with %d messages", job.ID, len(queuedMessages))
		}
	}

	log.Printf("📋 Completed successfully - processed %d queued Discord jobs", totalProcessedJobs)
	return nil
}
