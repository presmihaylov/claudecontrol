package discord

import (
	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/salesnotif"
	"ccbackend/services"
	"ccbackend/usecases/agents"
	"ccbackend/utils"
	"context"
	"fmt"
	"log"
	"math/rand"
	"slices"
	"strings"
)

// DiscordUseCase handles all Discord-specific operations
type DiscordUseCase struct {
	discordClient              clients.DiscordClient
	wsClient                   clients.SocketIOClient
	agentsService              services.AgentsService
	jobsService                services.JobsService
	discordMessagesService     services.DiscordMessagesService
	discordIntegrationsService services.DiscordIntegrationsService
	txManager                  services.TransactionManager
	agentsUseCase              agents.AgentsUseCaseInterface
}

// NewDiscordUseCase creates a new instance of DiscordUseCase
func NewDiscordUseCase(
	discordClient clients.DiscordClient,
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	discordMessagesService services.DiscordMessagesService,
	discordIntegrationsService services.DiscordIntegrationsService,
	txManager services.TransactionManager,
	agentsUseCase agents.AgentsUseCaseInterface,
) *DiscordUseCase {
	return &DiscordUseCase{
		discordClient:              discordClient,
		wsClient:                   wsClient,
		agentsService:              agentsService,
		jobsService:                jobsService,
		discordMessagesService:     discordMessagesService,
		discordIntegrationsService: discordIntegrationsService,
		txManager:                  txManager,
		agentsUseCase:              agentsUseCase,
	}
}

func (d *DiscordUseCase) ProcessDiscordMessageEvent(
	ctx context.Context,
	event models.DiscordMessageEvent,
	discordIntegrationID string,
	orgID models.OrgID,
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
			orgID,
			*event.ThreadID,
			discordIntegrationID,
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
		orgID,
		event.MessageID,
		event.ChannelID,
		threadID,
		event.UserID,
		discordIntegrationID,
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
	connectedAgents, err := d.agentsService.GetConnectedActiveAgents(ctx, orgID, connectedClientIDs)
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
		clientID, err = d.agentsUseCase.GetOrAssignAgentForJob(ctx, job, threadID, orgID)
		if err != nil {
			return fmt.Errorf("failed to get or assign agent for job: %w", err)
		}
		messageStatus = models.ProcessedDiscordMessageStatusInProgress
	}

	// Store the Discord message as ProcessedDiscordMessage with appropriate status
	processedMessage, err := d.discordMessagesService.CreateProcessedDiscordMessage(
		ctx,
		orgID,
		job.ID,
		event.MessageID,
		threadID,
		event.Content,
		discordIntegrationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed Discord message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := deriveMessageReactionFromStatus(messageStatus)
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
	orgID models.OrgID,
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
	maybeJob, err := d.jobsService.GetJobByDiscordThread(ctx, orgID, threadID, discordIntegrationID)
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
	maybeAgent, err := d.agentsService.GetAgentByJobID(ctx, orgID, job.ID)
	if err != nil {
		log.Printf("❌ Failed to find agent for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to get agent by job id: %w", err)
	}

	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent is found, unassign them from the job
		if maybeAgent.IsPresent() {
			agent := maybeAgent.MustGet()
			if err := d.agentsService.UnassignAgentFromJob(ctx, orgID, agent.ID, job.ID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}

			log.Printf("✅ Unassigned agent %s from manually completed job %s", agent.ID, job.ID)
		}

		// Delete the job and its associated processed messages
		if err := d.jobsService.DeleteJob(ctx, orgID, job.ID); err != nil {
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
	
	// Send sales notification for manual job completion
	salesnotif.New(orgID, fmt.Sprintf("Manually completed job `%s`", job.ID))
	
	log.Printf("🗑️ Deleted manually completed job %s", job.ID)
	log.Printf("📋 Completed successfully - processed manual job completion for job %s", job.ID)
	return nil
}

func (d *DiscordUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to process processing discord message notification from client %s", clientID)

	messageID := payload.ProcessedMessageID

	// Get processed discord message directly using organization_id (optimization)
	maybeMessage, err := d.discordMessagesService.GetProcessedDiscordMessageByID(
		ctx,
		orgID,
		messageID,
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
	maybeJob, err := d.jobsService.GetJobByID(ctx, orgID, processedMessage.JobID)
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

// ProcessSystemMessage handles system messages from agents and sends them to Discord
func (d *DiscordUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to process system message from client %s: %s", clientID, payload.Message)
	jobID := payload.JobID
	maybeJob, err := d.jobsService.GetJobByID(ctx, orgID, jobID)
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
	if job.DiscordPayload == nil {
		log.Printf("⚠️ Job %s has no Discord payload, skipping system message", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Check if this is an error message from the agent
	if isAgentErrorMessage(payload.Message) {
		log.Printf("❌ Detected agent error message for job %s: %s", job.ID, payload.Message)

		// Get the agent that encountered the error
		maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, orgID, clientID)
		if err != nil {
			log.Printf("❌ Failed to find agent for error handling: %v", err)
			return fmt.Errorf("failed to find agent for error handling: %w", err)
		}

		var agentID string
		if maybeAgent.IsPresent() {
			agentID = maybeAgent.MustGet().ID
		}

		// Clean up the failed job
		errorMessage := fmt.Sprintf(
			"%s Agent encountered an error and cannot continue:\n%s",
			EmojiCrossMark,
			payload.Message,
		)
		if err := d.CleanupFailedDiscordJob(ctx, job, agentID, errorMessage); err != nil {
			return fmt.Errorf("failed to cleanup failed job: %w", err)
		}

		log.Printf("📋 Completed error handling - cleaned up failed job %s", job.ID)
		return nil
	}

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	log.Printf(
		"📤 Sending system message to Discord thread %s",
		job.DiscordPayload.ThreadID,
	)

	// Send system message (gear emoji will be added automatically)
	if err := d.sendSystemMessage(ctx, discordIntegrationID, integration.DiscordGuildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, payload.Message); err != nil {
		return fmt.Errorf("❌ Failed to send system message to Discord: %v", err)
	}

	// Update job timestamp to track activity
	if err := d.jobsService.UpdateJobTimestamp(ctx, orgID, job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	log.Printf("📋 Completed successfully - sent system message to Discord thread %s", job.DiscordPayload.ThreadID)
	return nil
}

// ProcessJobComplete handles job completion from an agent
func (d *DiscordUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	orgID models.OrgID,
) error {
	log.Printf(
		"📋 Starting to process job complete from client %s: JobID: %s, Reason: %s",
		clientID,
		payload.JobID,
		payload.Reason,
	)

	jobID := payload.JobID
	maybeJob, err := d.jobsService.GetJobByID(ctx, orgID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed manually or by another agent, skipping", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Get the agent by WebSocket connection ID to verify ownership (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, orgID, clientID)
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
	if err := d.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, orgID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiCheckMark, discordIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Perform database operations within transaction
	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// Unassign the agent from the job
		if err := d.agentsService.UnassignAgentFromJob(ctx, orgID, agent.ID, jobID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent from job: %w", err)
		}
		log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

		// Delete the job and its associated processed messages
		if err := d.jobsService.DeleteJob(ctx, orgID, jobID); err != nil {
			log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
			return fmt.Errorf("failed to delete completed job: %w", err)
		}
		log.Printf("🗑️ Deleted completed job %s", jobID)

		return nil
	}); err != nil {
		return fmt.Errorf("failed to complete job processing in transaction: %w", err)
	}

	// Get Discord integration to get guild ID for sending system message
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// Send completion message to Discord thread with reason
	if err := d.sendSystemMessage(ctx, discordIntegrationID, integration.DiscordGuildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, payload.Reason); err != nil {
		log.Printf("❌ Failed to send completion message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
		return fmt.Errorf("failed to send completion message to Discord: %w", err)
	}

	log.Printf("📤 Sent completion message to Discord thread %s: %s", job.DiscordPayload.ThreadID, payload.Reason)

	// Send sales notification for job completion
	salesnotif.New(orgID, fmt.Sprintf("Completed job `%s`", jobID))

	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}

// ProcessAssistantMessage handles assistant messages from agents and updates Discord accordingly
func (d *DiscordUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to process assistant message from client %s", clientID)

	// Get the agent by WebSocket connection ID (agents are organization-scoped)
	maybeAgent, err := d.agentsService.GetAgentByWSConnectionID(ctx, orgID, clientID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", clientID)
		return fmt.Errorf("no agent found for client: %s", clientID)
	}
	agent := maybeAgent.MustGet()

	// Get the specific job from the payload to find the Discord thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in AssistantMessage payload")

	jobID := payload.JobID

	// Get job directly using organization_id (optimization)
	maybeJob, err := d.jobsService.GetJobByID(ctx, orgID, jobID)
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
	if job.DiscordPayload == nil {
		log.Printf("⚠️ Job %s has no Discord payload, skipping assistant message", jobID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID

	// Validate that this agent is actually assigned to this job
	if err := d.agentsUseCase.ValidateJobBelongsToAgent(ctx, agent.ID, jobID, orgID); err != nil {
		return err
	}

	log.Printf(
		"📤 Sending assistant message to Discord thread %s",
		job.DiscordPayload.ThreadID,
	)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Discord integration: %w", err)
	}
	if !maybeIntegration.IsPresent() {
		return fmt.Errorf("discord integration not found: %s", discordIntegrationID)
	}
	integration := maybeIntegration.MustGet()

	// Send assistant message to Discord - ThreadID contains the channel/thread info
	if err := d.sendDiscordMessage(
		ctx,
		discordIntegrationID,
		integration.DiscordGuildID,
		job.DiscordPayload.ThreadID,
		job.DiscordPayload.ThreadID,
		messageToSend,
	); err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Discord: %v", err)
	}

	// Update job timestamp to track activity
	if err := d.jobsService.UpdateJobTimestamp(ctx, orgID, job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
		return fmt.Errorf("failed to update job timestamp: %w", err)
	}

	// Update the ProcessedDiscordMessage status to COMPLETED
	utils.AssertInvariant(payload.ProcessedMessageID != "", "ProcessedMessageID is empty")

	messageID := payload.ProcessedMessageID

	updatedMessage, err := d.discordMessagesService.UpdateProcessedDiscordMessage(
		ctx,
		orgID,
		messageID,
		models.ProcessedDiscordMessageStatusCompleted,
		discordIntegrationID,
	)
	if err != nil {
		return fmt.Errorf("failed to update processed discord message status: %w", err)
	}

	// Add completed emoji reaction
	// For top-level messages (where DiscordMessageID equals DiscordThreadID), only set white_check_mark on job completion
	// For other messages, set white_check_mark immediately when processed
	isTopLevelMessage := updatedMessage.DiscordMessageID == job.DiscordPayload.MessageID
	if !isTopLevelMessage {
		reactionEmoji := deriveMessageReactionFromStatus(models.ProcessedDiscordMessageStatusCompleted)
		if err := d.updateDiscordMessageReaction(
			ctx,
			updatedMessage.DiscordThreadID,
			updatedMessage.DiscordMessageID,
			reactionEmoji,
			discordIntegrationID,
		); err != nil {
			return fmt.Errorf("failed to update discord message reaction: %w", err)
		}
	}

	// Check if this is the latest message in the job and add hand emoji if waiting for next steps
	maybeLatestMsg, err := d.discordMessagesService.GetLatestProcessedMessageForJob(
		ctx,
		orgID,
		job.ID,
		discordIntegrationID,
	)
	if err != nil {
		return fmt.Errorf("failed to get latest message for job: %w", err)
	}

	if maybeLatestMsg.IsPresent() && maybeLatestMsg.MustGet().ID == messageID {
		// This is the latest message - agent is done processing, add hand emoji to top-level message
		if err := d.updateDiscordMessageReaction(
			ctx,
			job.DiscordPayload.ChannelID,
			job.DiscordPayload.MessageID,
			EmojiRaisedHand,
			discordIntegrationID,
		); err != nil {
			log.Printf("⚠️ Failed to add hand emoji to job %s thread: %v", job.ID, err)
			return fmt.Errorf("failed to add hand emoji to job thread: %w", err)
		}
		log.Printf("✋ Added hand emoji to job %s - agent waiting for next steps", job.ID)
	}

	log.Printf("📋 Completed successfully - sent assistant message to Discord thread %s", job.DiscordPayload.ThreadID)
	return nil
}

// CleanupFailedDiscordJob handles the cleanup of a failed Discord job including Discord notifications and database cleanup
// This is exported so core use case can call it when deregistering agents
func (d *DiscordUseCase) CleanupFailedDiscordJob(
	ctx context.Context,
	job *models.Job,
	agentID string,
	failureMessage string,
) error {
	if job.DiscordPayload == nil {
		log.Printf("❌ Job %s has no Discord payload", job.ID)
		return fmt.Errorf("job has no Discord payload")
	}
	discordIntegrationID := job.DiscordPayload.IntegrationID
	orgID := job.OrgID

	// Get Discord integration to get guild ID
	maybeIntegration, err := d.discordIntegrationsService.GetDiscordIntegrationByID(ctx, discordIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get Discord integration: %v", err)
		// Continue with cleanup even if integration lookup fails
	}

	var guildID string
	if maybeIntegration.IsPresent() {
		guildID = maybeIntegration.MustGet().DiscordGuildID
	}

	// Send failure message to Discord thread
	if guildID != "" {
		if err := d.sendSystemMessage(ctx, discordIntegrationID, guildID, job.DiscordPayload.ChannelID, job.DiscordPayload.ThreadID, failureMessage); err != nil {
			log.Printf("❌ Failed to send failure message to Discord thread %s: %v", job.DiscordPayload.ThreadID, err)
			// Continue with cleanup even if Discord message fails
		}
	}

	// Update the top-level message emoji to ❌
	if err := d.updateDiscordMessageReaction(ctx, job.DiscordPayload.ChannelID, job.DiscordPayload.MessageID, EmojiCrossMark, discordIntegrationID); err != nil {
		log.Printf("❌ Failed to update discord message reaction to ❌ for failed job %s: %v", job.ID, err)
		// Continue with cleanup even if reaction update fails
	}

	// Perform database operations within transaction
	if err := d.txManager.WithTransaction(ctx, func(ctx context.Context) error {
		// If agent ID is provided, unassign agent from job
		if agentID != "" {
			if err := d.agentsService.UnassignAgentFromJob(ctx, orgID, agentID, job.ID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from job %s: %v", agentID, job.ID, err)
				return fmt.Errorf("failed to unassign agent from job: %w", err)
			}
			log.Printf("🔗 Unassigned agent %s from job %s", agentID, job.ID)
		}

		// Delete the job (use the job's discord integration and organization from the job)
		if err := d.jobsService.DeleteJob(ctx, orgID, job.ID); err != nil {
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

		// Get queued messages for this integration
		queuedMessages, err := d.discordMessagesService.GetProcessedMessagesByStatus(
			ctx,
			integration.OrgID,
			models.ProcessedDiscordMessageStatusQueued,
			discordIntegrationID,
		)
		if err != nil {
			return fmt.Errorf("failed to get queued messages for integration %s: %w", discordIntegrationID, err)
		}

		if len(queuedMessages) == 0 {
			continue
		}

		log.Printf("🔍 Found %d queued messages for integration %s", len(queuedMessages), discordIntegrationID)

		// Group messages by job ID for efficient processing
		jobMessagesMap := groupDiscordMessagesByJobID(queuedMessages)

		// Try to assign each job with queued messages to an available agent
		for jobID, messages := range jobMessagesMap {
			// Only fetch job if we need job payload for processing
			maybeJob, err := d.jobsService.GetJobByID(ctx, integration.OrgID, jobID)
			if err != nil {
				return fmt.Errorf("failed to get job %s for integration %s: %w", jobID, discordIntegrationID, err)
			}
			if maybeJob.IsAbsent() {
				return fmt.Errorf("job %s not found for integration %s", jobID, discordIntegrationID)
			}
			job := maybeJob.MustGet()

			log.Printf("🔄 Processing %d queued messages for job %s", len(messages), job.ID)

			// Get organization ID for this integration
			orgID := integration.OrgID

			// Try to assign job to an available agent
			clientID, assigned, err := d.agentsUseCase.TryAssignJobToAgent(ctx, job.ID, orgID)
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
				integration.OrgID,
				job.ID,
				models.ProcessedDiscordMessageStatusQueued,
				discordIntegrationID,
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
					integration.OrgID,
					message.ID,
					models.ProcessedDiscordMessageStatusInProgress,
					discordIntegrationID,
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
