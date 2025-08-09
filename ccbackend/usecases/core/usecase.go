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
	"ccbackend/services"
	"ccbackend/utils"
)

type CoreUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackIntegrationsService services.SlackIntegrationsService
	organizationsService     services.OrganizationsService
	txManager                services.TransactionManager
}

func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	txManager services.TransactionManager,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		txManager:                txManager,
	}
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

func (s *CoreUseCase) validateJobBelongsToAgent(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	agentJobs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agentID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get jobs for agent: %w", err)
	}
	if slices.Contains(agentJobs, jobID) {
		log.Printf("✅ Agent %s is assigned to job %s", agentID, jobID)
		return nil
	}

	log.Printf("❌ Agent %s is not assigned to job %s", agentID, jobID)
	return fmt.Errorf("agent %s is not assigned to job %s", agentID, jobID)
}

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

func (s *CoreUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process system message from client %s: %s", clientID, payload.Message)

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

	// Send system message (gear emoji will be added automatically)
	if err := s.sendSystemMessage(ctx, slackIntegrationID, processedMessage.SlackChannelID, job.SlackThreadTS, payload.Message); err != nil {
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

func (s *CoreUseCase) ProcessProcessingSlackMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingSlackMessagePayload,
	organizationID string,
) error {
	log.Printf("📋 Starting to process processing slack message notification from client %s", clientID)

	// Validate SlackMessageID is provided
	if payload.SlackMessageID == "" {
		log.Printf("⚠️ Processing slack message notification has no SlackMessageID")
		return fmt.Errorf("SlackMessageID is required")
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
			JobID:            message.JobID,
			Message:          resolvedText,
			SlackMessageID:   message.ID,
			SlackMessageLink: permalink,
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
			JobID:            message.JobID,
			Message:          resolvedText,
			SlackMessageID:   message.ID,
			SlackMessageLink: permalink,
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

func (s *CoreUseCase) getOrAssignAgentForJob(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	// Check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, job.ID, organizationID)
	if err != nil {
		// Some error occurred
		log.Printf("❌ Failed to check for existing agent assignment: %v", err)
		return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if !maybeExistingAgent.IsPresent() {
		// Job not assigned to any agent yet - need to assign to an available agent
		return s.assignJobToAvailableAgent(ctx, job, threadTS, organizationID)
	}

	existingAgent := maybeExistingAgent.MustGet()

	// Job is already assigned to an agent - verify it still has an active connection
	connectedClientIDs := s.wsClient.GetClientIDs()
	if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
		log.Printf(
			"🔄 Job %s already assigned to agent %s with active connection, routing message to existing agent",
			job.ID,
			existingAgent.ID,
		)
		return existingAgent.WSConnectionID, nil
	}

	// Existing agent doesn't have active connection - return error to signal no available agents
	log.Printf("⚠️ Job %s assigned to agent %s but no active WebSocket connection found", job.ID, existingAgent.ID)
	return "", fmt.Errorf("no active agents available for job assignment")
}

// assignJobToAvailableAgent attempts to assign a job to the least loaded available agent
// Returns the WebSocket client ID if successful, empty string if no agents available, or error on failure
func (s *CoreUseCase) assignJobToAvailableAgent(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	log.Printf("📝 Job %s not yet assigned, looking for any active agent", job.ID)

	clientID, assigned, err := s.tryAssignJobToAgent(ctx, job.ID, organizationID)
	if err != nil {
		return "", err
	}

	if !assigned {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", fmt.Errorf("no agents with active WebSocket connections available for job assignment")
	}

	log.Printf("✅ Assigned job %s to agent for slack thread %s (agent can handle multiple jobs)", job.ID, threadTS)
	return clientID, nil
}

// tryAssignJobToAgent is a reusable function that attempts to assign a job to the least loaded available agent
// Returns (clientID, wasAssigned, error) where:
// - clientID: WebSocket connection ID of assigned agent (empty if not assigned)
// - wasAssigned: true if job was successfully assigned to an agent, false if no agents available
// - error: any error that occurred during the assignment process
func (s *CoreUseCase) tryAssignJobToAgent(
	ctx context.Context,
	jobID string,
	organizationID string,
) (string, bool, error) {
	// First check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, jobID, organizationID)
	if err != nil {
		return "", false, fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if maybeExistingAgent.IsPresent() {
		existingAgent := maybeExistingAgent.MustGet()
		// Job is already assigned - check if agent still has active connection
		connectedClientIDs := s.wsClient.GetClientIDs()
		if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
			log.Printf("🔄 Job %s already assigned to agent %s with active connection", jobID, existingAgent.ID)
			return existingAgent.WSConnectionID, true, nil
		}
		// Agent no longer has active connection - job remains assigned but can't process
		log.Printf("⚠️ Job %s assigned to agent %s but no active connection", jobID, existingAgent.ID)
		return "", false, nil
	}

	// Job not assigned - proceed with assignment
	// Get active WebSocket connections first
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	// Get only agents with active connections using centralized service method
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to get connected active agents: %v", err)
		return "", false, fmt.Errorf("failed to get connected active agents: %w", err)
	}

	if len(connectedAgents) == 0 {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", false, nil
	}

	// Sort agents by load (number of assigned jobs) to select the least loaded agent
	sortedAgents, err := s.sortAgentsByLoad(ctx, connectedAgents, organizationID)
	if err != nil {
		log.Printf("❌ Failed to sort agents by load: %v", err)
		return "", false, fmt.Errorf("failed to sort agents by load: %w", err)
	}

	selectedAgent := sortedAgents[0].agent
	log.Printf("🎯 Selected agent %s with %d active jobs (least loaded)", selectedAgent.ID, sortedAgents[0].load)

	// Assign the job to the selected agent (agents can now handle multiple jobs simultaneously)
	if err := s.agentsService.AssignAgentToJob(ctx, selectedAgent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Failed to assign job %s to agent %s: %v", jobID, selectedAgent.ID, err)
		return "", false, fmt.Errorf("failed to assign job to agent: %w", err)
	}

	log.Printf("✅ Assigned job %s to agent %s", jobID, selectedAgent.ID)
	return selectedAgent.WSConnectionID, true, nil
}

func (s *CoreUseCase) RegisterAgent(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to register agent for client %s", client.ID)

	// Pass the agent ID to UpsertActiveAgent - use organization ID since agents are organization-scoped
	_, err := s.agentsService.UpsertActiveAgent(ctx, client.ID, client.OrganizationID, client.AgentID)
	if err != nil {
		return fmt.Errorf("failed to register agent for client %s: %w", client.ID, err)
	}

	log.Printf(
		"📋 Completed successfully - registered agent for client %s with organization %s",
		client.ID,
		client.OrganizationID,
	)
	return nil
}

func (s *CoreUseCase) DeregisterAgent(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to deregister agent for client %s", client.ID)

	// Find the agent directly using organization ID since agents are organization-scoped
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, client.ID, client.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", client.ID)
		return fmt.Errorf("no agent found for client: %s", client.ID)
	}

	agent := maybeAgent.MustGet()

	// Get active jobs for agent cleanup
	jobs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, client.OrganizationID)
	if err != nil {
		log.Printf("❌ Failed to get jobs for cleanup: %v", err)
		return fmt.Errorf("failed to get jobs for cleanup: %w", err)
	}

	// Clean up all job assignments - handle each job consistently
	log.Printf("🧹 Agent %s has %d assigned job(s), cleaning up all assignments", agent.ID, len(jobs))

	// Process each job: update Slack, unassign agent, delete job
	for _, jobID := range jobs {
		// Get job directly using organization_id (optimization)
		maybeJob, err := s.jobsService.GetJobByID(ctx, jobID, client.OrganizationID)
		if err != nil {
			log.Printf("❌ Failed to get job %s for cleanup: %v", jobID, err)
			return fmt.Errorf("failed to get job for cleanup: %w", err)
		}
		if !maybeJob.IsPresent() {
			log.Printf("❌ Job %s not found for cleanup", jobID)
			continue // Skip this job, it may have been deleted already
		}

		job := maybeJob.MustGet()

		// Clean up the abandoned job
		abandonmentMessage := ":x: The assigned agent was disconnected, abandoning job"
		if err := s.cleanupFailedJob(ctx, job, agent.ID, abandonmentMessage); err != nil {
			return fmt.Errorf("failed to cleanup abandoned job %s: %w", jobID, err)
		}
		log.Printf("✅ Cleaned up abandoned job %s", jobID)
	}

	// Delete the agent record (use organization ID since agents are organization-scoped)
	err = s.agentsService.DeleteActiveAgentByWsConnectionID(ctx, client.ID, client.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to deregister agent for client %s: %w", client.ID, err)
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", client.ID)
	return nil
}

func (s *CoreUseCase) BroadcastCheckIdleJobs(ctx context.Context) error {
	log.Printf("📋 Starting to broadcast CheckIdleJobs to all connected agents")

	// Get all organizations to broadcast to agents in each organization
	organizations, err := s.organizationsService.GetAllOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}

	if len(organizations) == 0 {
		log.Printf("📋 No organizations found")
		return nil
	}

	totalAgentCount := 0
	var broadcastErrors []string
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	for _, organization := range organizations {
		organizationID := organization.ID

		// Get connected agents for this organization using centralized service method
		connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
		if err != nil {
			broadcastErrors = append(
				broadcastErrors,
				fmt.Sprintf("failed to get connected agents for organization %s: %v", organizationID, err),
			)
			continue
		}

		if len(connectedAgents) == 0 {
			continue
		}

		log.Printf(
			"📡 Broadcasting CheckIdleJobs to %d connected agents for organization %s",
			len(connectedAgents),
			organizationID,
		)
		checkIdleJobsMessage := models.BaseMessage{
			ID:      core.NewID("msg"),
			Type:    models.MessageTypeCheckIdleJobs,
			Payload: models.CheckIdleJobsPayload{},
		}

		for _, agent := range connectedAgents {
			if err := s.wsClient.SendMessage(agent.WSConnectionID, checkIdleJobsMessage); err != nil {
				broadcastErrors = append(
					broadcastErrors,
					fmt.Sprintf("failed to send CheckIdleJobs message to agent %s: %v", agent.ID, err),
				)
				continue
			}
			log.Printf("📤 Sent CheckIdleJobs message to agent %s", agent.ID)
			totalAgentCount++
		}
	}

	log.Printf("📋 Completed broadcast - sent CheckIdleJobs to %d agents", totalAgentCount)

	// Return error if there were any broadcast failures
	if len(broadcastErrors) > 0 {
		return fmt.Errorf(
			"CheckIdleJobs broadcast encountered %d errors: %s",
			len(broadcastErrors),
			strings.Join(broadcastErrors, "; "),
		)
	}

	log.Printf("📋 Completed successfully - broadcasted CheckIdleJobs to %d agents", totalAgentCount)
	return nil
}

func (s *CoreUseCase) ProcessJobComplete(
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

	// Validate JobID is not empty
	if payload.JobID == "" {
		log.Printf("❌ Empty JobID from client %s", clientID)
		return fmt.Errorf("JobID cannot be empty")
	}

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := s.jobsService.GetJobByID(ctx, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed manually or by another agent, skipping", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	slackIntegrationID := job.SlackIntegrationID

	// Get the agent by WebSocket connection ID to verify ownership (agents are organization-scoped)
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

	// Validate that this agent is actually assigned to this job
	if err := s.validateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := s.updateSlackMessageReaction(ctx, job.SlackChannelID, job.SlackThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := s.agentsService.UnassignAgentFromJob(ctx, agent.ID, jobID, organizationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

		// Delete the job and its associated processed messages
		if err := s.jobsService.DeleteJob(ctx, jobID, slackIntegrationID, organizationID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}
		log.Printf("🗑️ Deleted completed job %s", jobID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete job processing in transaction: %w", err)
	}

	// Send completion message to Slack thread with reason
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackChannelID, job.SlackThreadTS, payload.Reason); err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s: %s", job.SlackThreadTS, payload.Reason)
	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}

func (s *CoreUseCase) ProcessQueuedJobs(ctx context.Context) error {
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
	var processingErrors []string

	for _, integration := range integrations {
		slackIntegrationID := integration.ID

		// Get jobs with queued messages for this integration
		queuedJobs, err := s.jobsService.GetJobsWithQueuedMessages(ctx, slackIntegrationID, integration.OrganizationID)
		if err != nil {
			processingErrors = append(
				processingErrors,
				fmt.Sprintf("failed to get queued jobs for integration %s: %v", slackIntegrationID, err),
			)
			continue
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
			clientID, assigned, err := s.tryAssignJobToAgent(ctx, job.ID, organizationID)
			if err != nil {
				processingErrors = append(
					processingErrors,
					fmt.Sprintf("failed to assign queued job %s: %v", job.ID, err),
				)
				continue
			}

			if !assigned {
				log.Printf("⚠️ Still no agents available for queued job %s", job.ID)
				continue
			}

			// Job was successfully assigned - get queued messages and send them to agent
			queuedMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(
				ctx,
				job.ID,
				models.ProcessedSlackMessageStatusQueued,
				slackIntegrationID,
				integration.OrganizationID,
			)
			if err != nil {
				processingErrors = append(
					processingErrors,
					fmt.Sprintf("failed to get queued messages for job %s: %v", job.ID, err),
				)
				continue
			}

			log.Printf("📨 Found %d queued messages for job %s", len(queuedMessages), job.ID)

			// Process each queued message
			for _, message := range queuedMessages {
				// Update message status to IN_PROGRESS
				updatedMessage, err := s.jobsService.UpdateProcessedSlackMessage(
					ctx,
					message.ID,
					models.ProcessedSlackMessageStatusInProgress,
					slackIntegrationID,
					organizationID,
				)
				if err != nil {
					processingErrors = append(
						processingErrors,
						fmt.Sprintf("failed to update message %s status: %v", message.ID, err),
					)
					continue
				}

				// Update Slack reaction to show processing (eyes emoji)
				if err := s.updateSlackMessageReaction(ctx, updatedMessage.SlackChannelID, updatedMessage.SlackTS, "eyes", slackIntegrationID); err != nil {
					processingErrors = append(
						processingErrors,
						fmt.Sprintf("failed to update slack reaction for message %s: %v", message.ID, err),
					)
					continue
				}

				// Determine if this is the first message in the job (new conversation)
				allMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(
					ctx,
					job.ID,
					models.ProcessedSlackMessageStatusCompleted,
					slackIntegrationID,
					integration.OrganizationID,
				)
				if err != nil {
					processingErrors = append(
						processingErrors,
						fmt.Sprintf("failed to check for existing messages in job %s: %v", job.ID, err),
					)
					continue
				}
				isNewConversation := len(allMessages) == 0

				// Send work to assigned agent
				if isNewConversation {
					if err := s.sendStartConversationToAgent(ctx, clientID, updatedMessage); err != nil {
						processingErrors = append(
							processingErrors,
							fmt.Sprintf("failed to send start conversation for message %s: %v", message.ID, err),
						)
						continue
					}
				} else {
					if err := s.sendUserMessageToAgent(ctx, clientID, updatedMessage); err != nil {
						processingErrors = append(processingErrors, fmt.Sprintf("failed to send user message %s: %v", message.ID, err))
						continue
					}
				}

				log.Printf("✅ Successfully assigned and sent queued message %s to agent", message.ID)
			}

			totalProcessedJobs++
			log.Printf("✅ Successfully processed queued job %s with %d messages", job.ID, len(queuedMessages))
		}
	}

	log.Printf("📋 Completed queue processing - processed %d jobs", totalProcessedJobs)

	// Return error if there were any processing failures
	if len(processingErrors) > 0 {
		return fmt.Errorf(
			"queued job processing encountered %d errors: %s",
			len(processingErrors),
			strings.Join(processingErrors, "; "),
		)
	}

	log.Printf("📋 Completed successfully - processed %d queued jobs", totalProcessedJobs)
	return nil
}

func (s *CoreUseCase) CleanupInactiveAgents(ctx context.Context) error {
	log.Printf("📋 Starting to cleanup inactive agents (>10 minutes)")

	// Get all slack integrations
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get slack integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No slack integrations found")
		return nil
	}

	totalInactiveAgents := 0
	var cleanupErrors []string
	inactiveThresholdMinutes := 10 // 10 minutes

	for _, integration := range integrations {
		slackIntegrationID := integration.ID
		organizationID := integration.OrganizationID

		// Get inactive agents for this organization (agents are organization-scoped)
		inactiveAgents, err := s.agentsService.GetInactiveAgents(ctx, organizationID, inactiveThresholdMinutes)
		if err != nil {
			cleanupErrors = append(
				cleanupErrors,
				fmt.Sprintf("failed to get inactive agents for integration %s: %v", slackIntegrationID, err),
			)
			continue
		}

		if len(inactiveAgents) == 0 {
			continue
		}

		log.Printf("🔍 Found %d inactive agents for integration %s", len(inactiveAgents), slackIntegrationID)

		// Delete each inactive agent
		for _, agent := range inactiveAgents {
			log.Printf(
				"🧹 Found inactive agent %s (last active: %s) - cleaning up",
				agent.ID,
				agent.LastActiveAt.Format("2006-01-02 15:04:05"),
			)

			// Delete the inactive agent - CASCADE DELETE will automatically clean up job assignments
			if err := s.agentsService.DeleteActiveAgent(ctx, agent.ID, organizationID); err != nil {
				cleanupErrors = append(
					cleanupErrors,
					fmt.Sprintf("failed to delete inactive agent %s: %v", agent.ID, err),
				)
				continue
			}

			log.Printf("✅ Deleted inactive agent %s (CASCADE DELETE cleaned up job assignments)", agent.ID)
			totalInactiveAgents++
		}
	}

	log.Printf("📋 Completed cleanup - removed %d inactive agents", totalInactiveAgents)

	// Return error if there were any cleanup failures
	if len(cleanupErrors) > 0 {
		return fmt.Errorf(
			"inactive agent cleanup encountered %d errors: %s",
			len(cleanupErrors),
			strings.Join(cleanupErrors, "; "),
		)
	}

	log.Printf("📋 Completed successfully - cleaned up %d inactive agents", totalInactiveAgents)
	return nil
}

func (s *CoreUseCase) ProcessPing(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to process ping from client %s", client.ID)

	// Check if agent exists for this client (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, client.ID, client.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", client.ID)
		return fmt.Errorf("no agent found for client: %s", client.ID)
	}

	// Update the agent's last_active_at timestamp (use organization ID since agents are organization-scoped)
	if err := s.agentsService.UpdateAgentLastActiveAt(ctx, client.ID, client.OrganizationID); err != nil {
		log.Printf("❌ Failed to update agent last_active_at for client %s: %v", client.ID, err)
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

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

func (s *CoreUseCase) ValidateAPIKey(ctx context.Context, apiKey string) (string, error) {
	log.Printf("📋 Starting to validate API key")

	maybeOrg, err := s.organizationsService.GetOrganizationBySecretKey(ctx, apiKey)
	if err != nil {
		return "", err
	}
	if !maybeOrg.IsPresent() {
		return "", fmt.Errorf("invalid API key")
	}
	organization := maybeOrg.MustGet()

	log.Printf("📋 Completed successfully - validated API key for organization %s", organization.ID)
	return organization.ID, nil
}

// cleanupFailedJob handles the cleanup of a failed job including Slack notifications and database cleanup
// This is used both when an agent encounters an error and when an agent is disconnected
func (s *CoreUseCase) cleanupFailedJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	slackIntegrationID := job.SlackIntegrationID
	organizationID := job.OrganizationID

	// Send failure message to Slack thread
	if err := s.sendSlackMessage(ctx, slackIntegrationID, job.SlackChannelID, job.SlackThreadTS, failureMessage); err != nil {
		log.Printf("❌ Failed to send failure message to Slack thread %s: %v", job.SlackThreadTS, err)
		// Continue with cleanup even if Slack message fails
	}

	// Update the top-level message emoji to :x:
	if err := s.updateSlackMessageReaction(ctx, job.SlackChannelID, job.SlackThreadTS, "x", slackIntegrationID); err != nil {
		log.Printf("❌ Failed to update slack message reaction to :x: for failed job %s: %v", job.ID, err)
		// Continue with cleanup even if reaction update fails
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent ID is provided, unassign agent from job
		if agentID != "" {
			if err := s.agentsService.UnassignAgentFromJob(ctx, agentID, job.ID, organizationID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}
			log.Printf("🔗 Unassigned agent %s from job %s", agentID, job.ID)
		}

		// Delete the job (use the job's slack integration and organization from the job)
		if err := s.jobsService.DeleteJob(ctx, job.ID, slackIntegrationID, organizationID); err != nil {
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
