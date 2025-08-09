package core

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ccbackend/models"
	"ccbackend/utils"
)

// ProcessAssistantMessage handles assistant messages from agents and updates Slack accordingly
func (s *CoreUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process assistant message from client %s", clientID)

	// Get the agent by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}
	agent := maybeAgent.MustGet()

	// Get the specific job from the payload to find the Slack thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in AssistantMessage payload")

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := s.jobsService.GetJobByID(ctx, jobID, organizationID)
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
	slackIntegrationID := job.SlackIntegrationID

	// Validate that this agent is actually assigned to this job
	if err := s.validateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		return err
	}

	log.Printf("📤 Sending assistant message to Slack thread %s in channel %s", job.SlackThreadTS, job.SlackChannelID)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Send assistant message to Slack
	if err := s.sendSlackMessage(ctx, slackIntegrationID, job.SlackChannelID, job.SlackThreadTS, messageToSend); err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(ctx, job.ID, slackIntegrationID, organizationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	// Update the ProcessedSlackMessage status to COMPLETED
	utils.AssertInvariant(payload.SlackMessageID != "", "SlackMessageID is empty")

	messageID := payload.SlackMessageID

	updatedMessage, err := s.jobsService.UpdateProcessedSlackMessage(
		ctx,
		messageID,
		models.ProcessedSlackMessageStatusCompleted,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	// Add completed emoji reaction
	// For top-level messages (where SlackTS equals SlackThreadTS), only set white_check_mark on job completion
	// For other messages, set white_check_mark immediately when processed
	isTopLevelMessage := updatedMessage.SlackTS == job.SlackThreadTS
	if !isTopLevelMessage {
		reactionEmoji := DeriveMessageReactionFromStatus(models.ProcessedSlackMessageStatusCompleted)
		if err := s.updateSlackMessageReaction(ctx, updatedMessage.SlackChannelID, updatedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
			return fmt.Errorf("failed to update slack message reaction: %w", err)
		}
	}

	// Check if this is the latest message in the job and add hand emoji if waiting for next steps
	maybeLatestMsg, err := s.jobsService.GetLatestProcessedMessageForJob(
		ctx,
		job.ID,
		slackIntegrationID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get latest message for job: %w", err)
	}

	if maybeLatestMsg.IsPresent() && maybeLatestMsg.MustGet().ID == messageID {
		// This is the latest message - agent is done processing, add hand emoji to top-level message
		if err := s.updateSlackMessageReaction(ctx, job.SlackChannelID, job.SlackThreadTS, "hand", slackIntegrationID); err != nil {
			log.Printf("⚠️ Failed to add hand emoji to job %s thread: %v", job.ID, err)
			return fmt.Errorf("failed to add hand emoji to job thread: %w", err)
		}
		log.Printf("✋ Added hand emoji to job %s - agent waiting for next steps", job.ID)
	}

	log.Printf("📋 Completed successfully - sent assistant message to Slack thread %s", job.SlackThreadTS)
	return nil
}

// ProcessSystemMessage handles system messages from agents and sends them to Slack
func (s *CoreUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	log.Printf(
		"📋 Starting to process system message from client %s for job %s: %s",
		clientID,
		payload.JobID,
		payload.Message,
	)

	// Validate SlackMessageID is provided
	if payload.SlackMessageID == "" {
		log.Printf("⚠️ System message has no SlackMessageID, cannot determine target thread")
		return nil
	}

	messageID := payload.SlackMessageID

	// Get processed slack message directly using organization_id (optimization)
	maybeMessage, err := s.jobsService.GetProcessedSlackMessageByID(
		ctx,
		messageID,
		organizationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get processed slack message: %w", err)
	}
	if !maybeMessage.IsPresent() {
		log.Printf(
			"⚠️ Processed slack message %s not found - job may have been completed manually, skipping system message",
			messageID,
		)
		return nil
	}

	processedMessage := maybeMessage.MustGet()
	slackIntegrationID := processedMessage.SlackIntegrationID

	// Get the job to find the thread timestamp (should be in the same slack integration)
	maybeJob, err := s.jobsService.GetJobByID(ctx, processedMessage.JobID, organizationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s: %v", processedMessage.JobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf(
			"⚠️ Job %s not found - already completed manually or by another agent, skipping system message",
			processedMessage.JobID,
		)
		return nil
	}
	job := maybeJob.MustGet()

	// Check if this is an error message from the agent
	if IsAgentErrorMessage(payload.Message) {
		log.Printf("❌ Detected agent error message for job %s: %s", job.ID, payload.Message)

		// Get the agent that encountered the error
		maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, clientID, organizationID)
		if err != nil {
			log.Printf("❌ Failed to find agent for error handling: %v", err)
			return fmt.Errorf("failed to find agent for error handling: %w", err)
		}

		var agentID string
		if maybeAgent.IsPresent() {
			agentID = maybeAgent.MustGet().ID
		}

		// Clean up the failed job
		errorMessage := fmt.Sprintf(":x: Agent encountered an error and cannot continue:\n%s", payload.Message)
		if err := s.cleanupFailedJob(ctx, job, agentID, errorMessage); err != nil {
			return fmt.Errorf("failed to cleanup failed job: %w", err)
		}

		log.Printf("📋 Completed error handling - cleaned up failed job %s", job.ID)
		return nil
	}

	log.Printf(
		"📤 Sending system message to Slack thread %s in channel %s",
		job.SlackThreadTS,
		processedMessage.SlackChannelID,
	)

	// Format message with Job ID and send system message (gear emoji will be added automatically)
	messageWithJobID := payload.Message
	if payload.JobID != "" {
		// Extract last 7 characters of job ID for brevity
		shortJobID := payload.JobID
		if len(shortJobID) > 7 {
			shortJobID = shortJobID[len(shortJobID)-7:]
		}
		messageWithJobID = fmt.Sprintf("[job_%s] %s", shortJobID, payload.Message)
	}
	if err := s.sendSystemMessage(ctx, slackIntegrationID, processedMessage.SlackChannelID, job.SlackThreadTS, messageWithJobID); err != nil {
		return fmt.Errorf("❌ Failed to send system message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(ctx, job.ID, slackIntegrationID, organizationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - sent system message to Slack thread %s", job.SlackThreadTS)
	return nil
}
