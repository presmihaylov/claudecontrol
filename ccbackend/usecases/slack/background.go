package slack

import (
	"context"
	"fmt"
	"log"

	"ccbackend/models"
)

// ProcessQueuedJobs processes jobs that are queued waiting for available agents
func (s *SlackUseCase) ProcessQueuedJobs(ctx context.Context) error {
	log.Printf("📋 Starting to process queued jobs")

	// Get all slack integrations
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get slack integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No slack integrations found")
		return nil
	}

	totalProcessedJobs := 0

	for _, integration := range integrations {
		slackIntegrationID := integration.ID

		// Get jobs with queued messages for this integration
		queuedJobs, err := s.jobsService.GetJobsWithQueuedMessages(ctx, slackIntegrationID, integration.OrganizationID)
		if err != nil {
			return fmt.Errorf("failed to get queued jobs for integration %s: %w", slackIntegrationID, err)
		}

		if len(queuedJobs) == 0 {
			continue
		}

		log.Printf("🔍 Found %d jobs with queued messages for integration %s", len(queuedJobs), slackIntegrationID)

		// Try to assign each queued job to an available agent
		for _, job := range queuedJobs {
			log.Printf("🔄 Processing queued job %s", job.ID)

			// Get organization ID for this integration
			organizationID := integration.OrganizationID

			// Try to assign job to an available agent
			clientID, assigned, err := s.agentsUseCase.TryAssignJobToAgent(ctx, job.ID, organizationID)
			if err != nil {
				return fmt.Errorf("failed to assign queued job %s: %w", job.ID, err)
			}

			if !assigned {
				log.Printf("⚠️ Still no agents available for queued job %s", job.ID)
				continue
			}

			// Job was successfully assigned - get queued messages and send them to agent
			queuedMessages, err := s.slackMessagesService.GetProcessedMessagesByJobIDAndStatus(
				ctx,
				job.ID,
				models.ProcessedSlackMessageStatusQueued,
				slackIntegrationID,
				integration.OrganizationID,
			)
			if err != nil {
				return fmt.Errorf("failed to get queued messages for job %s: %w", job.ID, err)
			}

			log.Printf("📨 Found %d queued messages for job %s", len(queuedMessages), job.ID)

			// Process each queued message
			for _, message := range queuedMessages {
				// Update message status to IN_PROGRESS
				updatedMessage, err := s.slackMessagesService.UpdateProcessedSlackMessage(
					ctx,
					message.ID,
					models.ProcessedSlackMessageStatusInProgress,
					slackIntegrationID,
					organizationID,
				)
				if err != nil {
					return fmt.Errorf("failed to update message %s status: %w", message.ID, err)
				}

				// Update Slack reaction to show processing (eyes emoji)
				if err := s.updateSlackMessageReaction(ctx, updatedMessage.SlackChannelID, updatedMessage.SlackTS, "eyes", slackIntegrationID); err != nil {
					return fmt.Errorf("failed to update slack reaction for message %s: %w", message.ID, err)
				}

				// Determine if this is the first message in the job (new conversation)
				// Check if this message's timestamp matches the job's thread timestamp (i.e., it's the top-level message)
				isNewConversation := false
				if job.SlackPayload != nil {
					isNewConversation = updatedMessage.SlackTS == job.SlackPayload.ThreadTS
				}

				// Send work to assigned agent
				if isNewConversation {
					if err := s.sendStartConversationToAgent(ctx, clientID, updatedMessage); err != nil {
						return fmt.Errorf("failed to send start conversation for message %s: %w", message.ID, err)
					}
				} else {
					if err := s.sendUserMessageToAgent(ctx, clientID, updatedMessage); err != nil {
						return fmt.Errorf("failed to send user message %s: %w", message.ID, err)
					}
				}

				log.Printf("✅ Successfully assigned and sent queued message %s to agent", message.ID)
			}

			totalProcessedJobs++
			log.Printf("✅ Successfully processed queued job %s with %d messages", job.ID, len(queuedMessages))
		}
	}

	log.Printf("📋 Completed successfully - processed %d queued jobs", totalProcessedJobs)
	return nil
}
