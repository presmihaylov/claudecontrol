package discord

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

// CleanupFailedDiscordJob cleans up a failed Discord job by unassigning the agent and sending a failure message
func (d *DiscordUseCase) CleanupFailedDiscordJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	abandonmentMessage string,
) error {
	log.Printf("📋 Starting to cleanup failed Discord job %s for agent %s", job.ID, agentID)

	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := d.agentsService.UnassignAgentFromJob(ctx, agentID, job.ID, job.OrganizationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from failed job %s", agentID, job.ID)

		// Get all active processed messages for this job to mark them as failed
		activeMessages, err := d.discordMessagesService.GetProcessedMessagesByJobIDAndStatus(
			ctx,
			job.ID,
			models.ProcessedDiscordMessageStatusInProgress,
			job.DiscordPayload.DiscordIntegrationID,
			job.OrganizationID,
		)
		if err != nil {
			log.Printf("❌ Failed to get active messages for job %s: %v", job.ID, err)
			return fmt.Errorf("failed to get active messages for job: %w", err)
		}

		// Mark all active messages as failed
		for _, message := range activeMessages {
			_, err := d.discordMessagesService.UpdateProcessedDiscordMessage(
				ctx,
				message.ID,
				models.ProcessedDiscordMessageStatusFailed,
				job.DiscordPayload.DiscordIntegrationID,
				job.OrganizationID,
			)
			if err != nil {
				log.Printf("❌ Failed to mark message %s as failed: %v", message.ID, err)
				return fmt.Errorf("failed to mark message as failed: %w", err)
			}
			log.Printf("✅ Marked message %s as failed for job %s", message.ID, job.ID)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to cleanup Discord job in transaction: %w", err)
	}

	// Send abandonment message to Discord thread
	if job.DiscordPayload != nil {
		if err := d.sendSystemMessage(
			ctx,
			job.DiscordPayload.DiscordIntegrationID,
			job.DiscordPayload.ChannelID,
			abandonmentMessage,
		); err != nil {
			log.Printf("❌ Failed to send abandonment message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
			return fmt.Errorf("failed to send abandonment message to Discord: %w", err)
		}
		log.Printf("📤 Sent abandonment message to Discord thread %s", job.DiscordPayload.ThreadID)

		// Update Discord message reaction to show failure
		if err := d.updateDiscordMessageReaction(
			ctx,
			job.DiscordPayload.ChannelID,
			job.DiscordPayload.ThreadID,
			"❌",
			job.DiscordPayload.DiscordIntegrationID,
		); err != nil {
			log.Printf("⚠️ Failed to update reaction for failed job %s: %v", job.ID, err)
			// Don't return error - this is not critical
		}
	}

	log.Printf("📋 Completed successfully - cleaned up failed Discord job %s", job.ID)
	return nil
}