package slack

import (
	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/agents"
	"ccbackend/utils"
	"context"
	"fmt"
	"log"
	"strings"
)

// SlackClientFactory creates a Slack client given an auth token
type SlackClientFactory func(authToken string) clients.SlackClient

// SlackUseCase handles all Slack-specific operations
type SlackUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackMessagesService     services.SlackMessagesService
	slackIntegrationsService services.SlackIntegrationsService
	txManager                services.TransactionManager
	agentsUseCase            agents.AgentsUseCaseInterface
	slackClientFactory       SlackClientFactory
}

// NewSlackUseCase creates a new instance of SlackUseCase
func NewSlackUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackMessagesService services.SlackMessagesService,
	slackIntegrationsService services.SlackIntegrationsService,
	txManager services.TransactionManager,
	agentsUseCase agents.AgentsUseCaseInterface,
	slackClientFactory SlackClientFactory,
) *SlackUseCase {
	return &SlackUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackMessagesService:     slackMessagesService,
		slackIntegrationsService: slackIntegrationsService,
		txManager:                txManager,
		agentsUseCase:            agentsUseCase,
		slackClientFactory:       slackClientFactory,
	}
}

func (s *SlackUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	organizationID models.OrgID,
) error {
	log.Printf("📋 Starting to process Slack message event from %s in %s: %s", event.User, event.Channel, event.Text)

	// For thread replies, validate that a job exists first (don't create new jobs)
	if event.ThreadTS != "" {
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadTS, event.Channel)

		// Check if job exists for this thread - thread replies cannot create new jobs
		maybeJob, err := s.jobsService.GetJobBySlackThread(
			ctx,
			organizationID,
			event.ThreadTS,
			event.Channel,
			slackIntegrationID,
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
		organizationID,
		threadTS,
		event.Channel,
		event.User,
		slackIntegrationID,
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
		organizationID,
		job.ID,
		event.Channel,
		event.TS,
		event.Text,
		slackIntegrationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := deriveMessageReactionFromStatus(messageStatus)
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
	organizationID models.OrgID,
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
	maybeJob, err := s.jobsService.GetJobBySlackThread(ctx, organizationID, messageTS, channelID, slackIntegrationID)
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
	maybeAgent, err := s.agentsService.GetAgentByJobID(ctx, organizationID, job.ID)
	if err != nil {
		log.Printf("❌ Failed to find agent for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to get agent by job id: %w", err)
	}

	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent is found, unassign them from the job
		if maybeAgent.IsPresent() {
			agent := maybeAgent.MustGet()
			if err := s.agentsService.UnassignAgentFromJob(ctx, organizationID, agent.ID, job.ID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}

			log.Printf("✅ Unassigned agent %s from manually completed job %s", agent.ID, job.ID)
		}

		// Delete the job and its associated processed messages
		if err := s.jobsService.DeleteJob(ctx, organizationID, job.ID); err != nil {
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
	organizationID models.OrgID,
) error {
	log.Printf("📋 Starting to process processing slack message notification from client %s", clientID)

	messageID := payload.ProcessedMessageID

	// Get processed slack message directly using organization_id (optimization)
	maybeMessage, err := s.slackMessagesService.GetProcessedSlackMessageByID(
		ctx,
		organizationID,
		messageID,
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

		// Get queued messages for this integration
		queuedMessages, err := s.slackMessagesService.GetProcessedMessagesByStatus(
			ctx,
			integration.OrgID,
			models.ProcessedSlackMessageStatusQueued,
			slackIntegrationID,
		)
		if err != nil {
			return fmt.Errorf("failed to get queued messages for integration %s: %w", slackIntegrationID, err)
		}

		if len(queuedMessages) == 0 {
			continue
		}

		log.Printf("🔍 Found %d queued messages for integration %s", len(queuedMessages), slackIntegrationID)

		// Group messages by job ID for efficient processing
		jobMessagesMap := groupMessagesByJobID(queuedMessages)

		// Try to assign each job with queued messages to an available agent
		for jobID, messages := range jobMessagesMap {
			// Only fetch job if we need job payload for processing
			maybeJob, err := s.jobsService.GetJobByID(ctx, integration.OrgID, jobID)
			if err != nil {
				log.Printf("❌ Failed to get job %s: %v", jobID, err)
				continue
			}
			if maybeJob.IsNone() {
				log.Printf("❌ Job %s not found, skipping messages", jobID)
				continue
			}
			job := maybeJob.MustGet()

			log.Printf("🔄 Processing %d queued messages for job %s", len(messages), job.ID)

			// Get organization ID for this integration
			organizationID := integration.OrgID

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
				integration.OrgID,
				job.ID,
				models.ProcessedSlackMessageStatusQueued,
				slackIntegrationID,
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
					integration.OrgID,
					message.ID,
					models.ProcessedSlackMessageStatusInProgress,
					slackIntegrationID,
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

// ProcessJobComplete handles job completion from an agent
func (s *SlackUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	organizationID models.OrgID,
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
	maybeJob, err := s.jobsService.GetJobByID(ctx, organizationID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed manually or by another agent, skipping", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	if job.SlackPayload == nil {
		log.Printf("❌ Job %s has no Slack payload", jobID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID

	// Get the agent by WebSocket connection ID to verify ownership (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, organizationID, clientID)
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
	if err := s.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := s.agentsService.UnassignAgentFromJob(ctx, organizationID, agent.ID, jobID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

		// Delete the job and its associated processed messages
		if err := s.jobsService.DeleteJob(ctx, organizationID, jobID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}
		log.Printf("🗑️ Deleted completed job %s", jobID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete job processing in transaction: %w", err)
	}

	// Send completion message to Slack thread with reason
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, payload.Reason); err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackPayload.ThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s: %s", job.SlackPayload.ThreadTS, payload.Reason)
	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}

// CleanupFailedSlackJob handles the cleanup of a failed Slack job including Slack notifications and database cleanup
// This is exported so core use case can call it when deregistering agents
func (s *SlackUseCase) CleanupFailedSlackJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	if job.SlackPayload == nil {
		log.Printf("❌ Job %s has no Slack payload", job.ID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID
	organizationID := job.OrgID

	// Send failure message to Slack thread
	if err := s.sendSlackMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, failureMessage); err != nil {
		log.Printf("❌ Failed to send failure message to Slack thread %s: %v", job.SlackPayload.ThreadTS, err)
		// Continue with cleanup even if Slack message fails
	}

	// Update the top-level message emoji to :x:
	if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "x", slackIntegrationID); err != nil {
		log.Printf("❌ Failed to update slack message reaction to :x: for failed job %s: %v", job.ID, err)
		// Continue with cleanup even if reaction update fails
	}

	// Perform database operations within transaction
	if err := s.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent ID is provided, unassign agent from job
		if agentID != "" {
			if err := s.agentsService.UnassignAgentFromJob(ctx, organizationID, agentID, job.ID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}
			log.Printf("🔗 Unassigned agent %s from job %s", agentID, job.ID)
		}

		// Delete the job (use the job's slack integration and organization from the job)
		if err := s.jobsService.DeleteJob(ctx, organizationID, job.ID); err != nil {
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

// ProcessAssistantMessage handles assistant messages from agents and updates Slack accordingly
func (s *SlackUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID models.OrgID,
) error {
	log.Printf("📋 Starting to process assistant message from client %s", clientID)

	// Get the agent by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, organizationID, clientID)
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
	maybeJob, err := s.jobsService.GetJobByID(ctx, organizationID, jobID)
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
	if job.SlackPayload == nil {
		log.Printf("⚠️ Job %s has no Slack payload, skipping assistant message", jobID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID

	// Validate that this agent is actually assigned to this job
	if err := s.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, organizationID); err != nil {
		return err
	}

	log.Printf(
		"📤 Sending assistant message to Slack thread %s in channel %s",
		job.SlackPayload.ThreadTS,
		job.SlackPayload.ChannelID,
	)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Send assistant message to Slack
	if err := s.sendSlackMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, messageToSend); err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(ctx, organizationID, job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	// Update the ProcessedSlackMessage status to COMPLETED
	utils.AssertInvariant(payload.ProcessedMessageID != "", "ProcessedMessageID is empty")

	messageID := payload.ProcessedMessageID

	updatedMessage, err := s.slackMessagesService.UpdateProcessedSlackMessage(
		ctx,
		organizationID,
		messageID,
		models.ProcessedSlackMessageStatusCompleted,
		slackIntegrationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	// Add completed emoji reaction
	// For top-level messages (where SlackTS equals SlackThreadTS), only set white_check_mark on job completion
	// For other messages, set white_check_mark immediately when processed
	isTopLevelMessage := updatedMessage.SlackTS == job.SlackPayload.ThreadTS
	if !isTopLevelMessage {
		reactionEmoji := deriveMessageReactionFromStatus(models.ProcessedSlackMessageStatusCompleted)
		if err := s.updateSlackMessageReaction(ctx, updatedMessage.SlackChannelID, updatedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
			return fmt.Errorf("failed to update slack message reaction: %w", err)
		}
	}

	// Check if this is the latest message in the job and add hand emoji if waiting for next steps
	maybeLatestMsg, err := s.slackMessagesService.GetLatestProcessedMessageForJob(
		ctx,
		organizationID,
		job.ID,
		slackIntegrationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get latest message for job: %w", err)
	}

	if maybeLatestMsg.IsPresent() && maybeLatestMsg.MustGet().ID == messageID {
		// This is the latest message - agent is done processing, add hand emoji to top-level message
		if err := s.updateSlackMessageReaction(ctx, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, "hand", slackIntegrationID); err != nil {
			log.Printf("⚠️ Failed to add hand emoji to job %s thread: %v", job.ID, err)
			return fmt.Errorf("failed to add hand emoji to job thread: %w", err)
		}
		log.Printf("✋ Added hand emoji to job %s - agent waiting for next steps", job.ID)
	}

	log.Printf("📋 Completed successfully - sent assistant message to Slack thread %s", job.SlackPayload.ThreadTS)
	return nil
}

// ProcessSystemMessage handles system messages from agents and sends them to Slack
func (s *SlackUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID models.OrgID,
) error {
	log.Printf("📋 Starting to process system message from client %s: %s", clientID, payload.Message)
	jobID := payload.JobID
	maybeJob, err := s.jobsService.GetJobByID(ctx, organizationID, jobID)
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
	if job.SlackPayload == nil {
		log.Printf("⚠️ Job %s has no Slack payload, skipping assistant message", jobID)
		return fmt.Errorf("job has no Slack payload")
	}
	slackIntegrationID := job.SlackPayload.IntegrationID

	// Check if this is an error message from the agent
	if isAgentErrorMessage(payload.Message) {
		log.Printf("❌ Detected agent error message for job %s: %s", job.ID, payload.Message)

		// Get the agent that encountered the error
		maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, organizationID, clientID)
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
		if err := s.CleanupFailedSlackJob(ctx, job, agentID, errorMessage); err != nil {
			return fmt.Errorf("failed to cleanup failed job: %w", err)
		}

		log.Printf("📋 Completed error handling - cleaned up failed job %s", job.ID)
		return nil
	}

	log.Printf(
		"📤 Sending system message to Slack thread %s in channel %s",
		job.SlackPayload.ThreadTS,
		job.SlackPayload.ChannelID,
	)

	// Send system message (gear emoji will be added automatically)
	if err := s.sendSystemMessage(ctx, slackIntegrationID, job.SlackPayload.ChannelID, job.SlackPayload.ThreadTS, payload.Message); err != nil {
		return fmt.Errorf("❌ Failed to send system message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(ctx, organizationID, job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - sent system message to Slack thread %s", job.SlackPayload.ThreadTS)
	return nil
}
