package core

import (
	"context"
	"fmt"
	"log"
	"slices"
	"strings"

	"github.com/samber/mo"

	"ccbackend/clients"
	slackclient "ccbackend/clients/slack"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/utils"
)

func (s *CoreUseCase) ProcessSlackMessageEvent(
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
		clientID, err = s.getOrAssignAgentForJob(ctx, job, threadTS, organizationID)
		if err != nil {
			return fmt.Errorf("failed to get or assign agent for job: %w", err)
		}
		messageStatus = models.ProcessedSlackMessageStatusInProgress
	}

	// Store the Slack message as ProcessedSlackMessage with appropriate status
	processedMessage, err := s.jobsService.CreateProcessedSlackMessage(
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
	if err := s.updateSlackMessageReaction(ctx, job.SlackChannelID, job.SlackThreadTS, "eyes", slackIntegrationID); err != nil {
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

func (s *CoreUseCase) ProcessReactionAdded(
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
	if job.SlackUserID != userID {
		log.Printf("⏭️ Reaction from %s ignored - job %s was created by %s", userID, job.ID, job.SlackUserID)
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
	if err := s.updateSlackMessageReaction(ctx, job.SlackChannelID, job.SlackThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update reaction for completed job %s: %v", job.ID, err)
		// Don't return error - this is not critical
	}

	// Send completion message to Slack thread
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackChannelID, job.SlackThreadTS, "Job manually marked as complete"); err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s", job.SlackThreadTS)
	log.Printf("🗑️ Deleted manually completed job %s", job.ID)
	log.Printf("📋 Completed successfully - processed manual job completion for job %s", job.ID)
	return nil
}

func (s *CoreUseCase) ProcessProcessingSlackMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingSlackMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process processing slack message notification from client %s", clientID)

	messageID := payload.ProcessedMessageID

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

func (s *CoreUseCase) getSlackClientForIntegration(
	ctx context.Context,
	slackIntegrationID string,
) (clients.SlackClient, error) {
	maybeSlackInt, err := s.slackIntegrationsService.GetSlackIntegrationByID(ctx, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration: %w", err)
	}
	if !maybeSlackInt.IsPresent() {
		return nil, fmt.Errorf("slack integration not found: %s", slackIntegrationID)
	}
	integration := maybeSlackInt.MustGet()

	return slackclient.NewSlackClient(integration.SlackAuthToken), nil
}

func (s *CoreUseCase) sendStartConversationToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedSlackMessage,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get job to access thread timestamp
	maybeJob, err := s.jobsService.GetJobByID(ctx, message.JobID, message.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate permalink for the thread's first message
	permalink, err := slackClient.GetPermalink(&clients.SlackPermalinkParameters{
		Channel: message.SlackChannelID,
		TS:      job.SlackThreadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := slackClient.ResolveMentionsInMessage(ctx, message.TextContent)
	startConversationMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
		return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
	}
	log.Printf("🚀 Sent start conversation message to client %s", clientID)
	return nil
}

func (s *CoreUseCase) sendUserMessageToAgent(
	ctx context.Context,
	clientID string,
	message *models.ProcessedSlackMessage,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get job to access thread timestamp
	maybeJob, err := s.jobsService.GetJobByID(ctx, message.JobID, message.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		return fmt.Errorf("job not found: %s", message.JobID)
	}
	job := maybeJob.MustGet()

	// Generate permalink for the thread's first message
	permalink, err := slackClient.GetPermalink(&clients.SlackPermalinkParameters{
		Channel: message.SlackChannelID,
		TS:      job.SlackThreadTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := slackClient.ResolveMentionsInMessage(ctx, message.TextContent)
	userMessage := models.BaseMessage{
		ID:   core.NewID("msg"),
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			JobID:              message.JobID,
			Message:            resolvedText,
			ProcessedMessageID: message.ID,
			MessageLink:        permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
		return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
	}
	log.Printf("💬 Sent user message to client %s", clientID)
	return nil
}

func (s *CoreUseCase) updateSlackMessageReaction(
	ctx context.Context,
	channelID, messageTS, newEmoji, slackIntegrationID string,
) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Get only reactions added by our bot
	botReactions, err := s.getBotReactionsOnMessage(ctx, channelID, messageTS, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get bot reactions: %w", err)
	}

	// Only remove reactions that are incompatible with the new state
	reactionsToRemove := getOldReactions(newEmoji)
	for _, emoji := range reactionsToRemove {
		if slices.Contains(botReactions, emoji) {
			if err := slackClient.RemoveReaction(emoji, clients.SlackItemRef{
				Channel:   channelID,
				Timestamp: messageTS,
			}); err != nil {
				return fmt.Errorf("failed to remove %s reaction: %w", emoji, err)
			}
		}
	}

	// Add new reaction if not already there
	if newEmoji != "" && !slices.Contains(botReactions, newEmoji) {
		if err := slackClient.AddReaction(newEmoji, clients.SlackItemRef{
			Channel:   channelID,
			Timestamp: messageTS,
		}); err != nil {
			return fmt.Errorf("failed to add %s reaction: %w", newEmoji, err)
		}
	}

	return nil
}

func (s *CoreUseCase) sendSlackMessage(
	ctx context.Context,
	slackIntegrationID, channelID, threadTS, message string,
) error {
	log.Printf("📋 Starting to send message to channel %s, thread %s: %s", channelID, threadTS, message)

	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Send message to Slack
	params := clients.SlackMessageParams{
		Text: utils.ConvertMarkdownToSlack(message),
	}
	if threadTS != "" {
		params.ThreadTS = mo.Some(threadTS)
	}
	_, err = slackClient.PostMessage(channelID, params)
	if err != nil {
		return fmt.Errorf("failed to send message to Slack: %w", err)
	}

	log.Printf("📋 Completed successfully - sent message to channel %s, thread %s", channelID, threadTS)
	return nil
}

func (s *CoreUseCase) sendSystemMessage(
	ctx context.Context,
	slackIntegrationID, channelID, threadTS, message string,
) error {
	log.Printf("📋 Starting to send system message to channel %s, thread %s: %s", channelID, threadTS, message)

	// Prepend gear emoji to message
	systemMessage := ":gear: " + message

	// Use the base sendSlackMessage function
	return s.sendSlackMessage(ctx, slackIntegrationID, channelID, threadTS, systemMessage)
}

func (s *CoreUseCase) getBotUserID(ctx context.Context, slackIntegrationID string) (string, error) {
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return "", fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	authTest, err := slackClient.AuthTest()
	if err != nil {
		return "", fmt.Errorf("failed to get bot user ID: %w", err)
	}
	return authTest.UserID, nil
}

func (s *CoreUseCase) getBotReactionsOnMessage(
	ctx context.Context,
	channelID, messageTS string,
	slackIntegrationID string,
) ([]string, error) {
	slackClient, err := s.getSlackClientForIntegration(ctx, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	botUserID, err := s.getBotUserID(ctx, slackIntegrationID)
	if err != nil {
		return nil, err
	}

	// Get reactions directly using GetReactions - much less rate limited
	reactions, err := slackClient.GetReactions(clients.SlackItemRef{
		Channel:   channelID,
		Timestamp: messageTS,
	}, clients.SlackGetReactionsParameters{})
	if err != nil {
		return nil, fmt.Errorf("failed to get reactions: %w", err)
	}

	var botReactions []string
	for _, reaction := range reactions {
		// Check if bot added this reaction
		if slices.Contains(reaction.Users, botUserID) {
			botReactions = append(botReactions, reaction.Name)
		}
	}

	return botReactions, nil
}

func DeriveMessageReactionFromStatus(status models.ProcessedSlackMessageStatus) string {
	switch status {
	case models.ProcessedSlackMessageStatusInProgress:
		return "hourglass"
	case models.ProcessedSlackMessageStatusQueued:
		return "hourglass"
	case models.ProcessedSlackMessageStatusCompleted:
		return "white_check_mark"
	default:
		utils.AssertInvariant(false, "invalid status received")
		return ""
	}
}

// IsAgentErrorMessage determines if a system message from ccagent indicates an error or failure
func IsAgentErrorMessage(message string) bool {
	// Check if message starts with the specific error prefix from ccagent
	return strings.HasPrefix(message, "ccagent encountered error:")
}

func getOldReactions(newEmoji string) []string {
	allReactions := []string{"hourglass", "eyes", "white_check_mark", "hand", "x"}

	var result []string
	for _, reaction := range allReactions {
		if reaction != newEmoji {
			result = append(result, reaction)
		}
	}

	return result
}
