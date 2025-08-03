package usecases

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/utils"

	"github.com/google/uuid"
	"github.com/slack-go/slack"
)

type CoreUseCase struct {
	wsClient                 *clients.WebSocketClient
	agentsService            *services.AgentsService
	jobsService              *services.JobsService
	slackIntegrationsService *services.SlackIntegrationsService
}

func NewCoreUseCase(wsClient *clients.WebSocketClient, agentsService *services.AgentsService, jobsService *services.JobsService, slackIntegrationsService *services.SlackIntegrationsService) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
	}
}

func (s *CoreUseCase) getSlackClientForIntegration(slackIntegrationID uuid.UUID) (*slack.Client, error) {
	integration, err := s.slackIntegrationsService.GetSlackIntegrationByID(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration: %w", err)
	}

	return slack.New(integration.SlackAuthToken), nil
}

func (s *CoreUseCase) validateJobBelongsToAgent(agentID, jobID uuid.UUID, slackIntegrationID string) error {
	agentJobs, err := s.agentsService.GetActiveAgentJobAssignments(agentID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get jobs for agent: %w", err)
	}

	for _, assignedJobID := range agentJobs {
		if assignedJobID == jobID {
			return nil
		}
	}

	log.Printf("❌ Agent %s is not assigned to job %s", agentID, jobID)
	return fmt.Errorf("agent %s is not assigned to job %s", agentID, jobID)
}

func (s *CoreUseCase) ProcessAssistantMessage(clientID string, payload models.AssistantMessagePayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process assistant message from client %s: %s", clientID, payload.Message)

	// Get the agent by WebSocket connection ID
	agent, err := s.agentsService.GetAgentByWSConnectionID(clientID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}

	// Get the specific job from the payload to find the Slack thread information
	utils.AssertInvariant(payload.JobID != "", "JobID is empty in AssistantMessage payload")
	utils.AssertInvariant(uuid.Validate(payload.JobID) == nil, "JobID is not in UUID format")

	jobID := uuid.MustParse(payload.JobID)
	job, err := s.jobsService.GetJobByID(jobID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s: %v", jobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Validate that this agent is actually assigned to this job
	if err := s.validateJobBelongsToAgent(agent.ID, jobID, slackIntegrationID); err != nil {
		return err
	}

	log.Printf("📤 Sending assistant message to Slack thread %s in channel %s", job.SlackThreadTS, job.SlackChannelID)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Get integration-specific Slack client
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return fmt.Errorf("invalid slack integration ID format: %w", err)
	}
	slackClient, err := s.getSlackClientForIntegration(integrationUUID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	_, _, err = slackClient.PostMessage(job.SlackChannelID,
		slack.MsgOptionText(utils.ConvertMarkdownToSlack(messageToSend), false),
		slack.MsgOptionTS(job.SlackThreadTS),
	)
	if err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(job.ID, slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
	}

	// Update the ProcessedSlackMessage status to COMPLETED
	utils.AssertInvariant(payload.SlackMessageID != "", "SlackMessageID is empty")
	utils.AssertInvariant(uuid.Validate(payload.SlackMessageID) == nil, "SlackMessageID is not in UUID format")

	messageID := uuid.MustParse(payload.SlackMessageID)

	updatedMessage, err := s.jobsService.UpdateProcessedSlackMessage(messageID, models.ProcessedSlackMessageStatusCompleted, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	// Add completed emoji reaction
	// For top-level messages (where SlackTS equals SlackThreadTS), only set white_check_mark on job completion
	// For other messages, set white_check_mark immediately when processed
	isTopLevelMessage := updatedMessage.SlackTS == job.SlackThreadTS
	if !isTopLevelMessage {
		reactionEmoji := DeriveMessageReactionFromStatus(models.ProcessedSlackMessageStatusCompleted)
		if err := s.updateSlackMessageReaction(updatedMessage.SlackChannelID, updatedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
			return fmt.Errorf("failed to update slack message reaction: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - sent assistant message to Slack thread %s", job.SlackThreadTS)
	return nil
}

func (s *CoreUseCase) ProcessSystemMessage(clientID string, payload models.SystemMessagePayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process system message from client %s: %s", clientID, payload.Message)

	// Validate SlackMessageID is provided
	if payload.SlackMessageID == "" {
		log.Printf("⚠️ System message has no SlackMessageID, cannot determine target thread")
		return nil
	}

	utils.AssertInvariant(uuid.Validate(payload.SlackMessageID) == nil, "SlackMessageID is not in UUID format")
	messageID := uuid.MustParse(payload.SlackMessageID)

	// Get the ProcessedSlackMessage to find the correct channel and thread
	processedMessage, err := s.jobsService.GetProcessedSlackMessageByID(messageID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get processed slack message %s: %v", messageID, err)
		return fmt.Errorf("failed to get processed slack message: %w", err)
	}

	// Get the job to find the thread timestamp
	job, err := s.jobsService.GetJobByID(processedMessage.JobID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s: %v", processedMessage.JobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}

	log.Printf("📤 Sending system message to Slack thread %s in channel %s", job.SlackThreadTS, processedMessage.SlackChannelID)

	// Get integration-specific Slack client
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return fmt.Errorf("invalid slack integration ID format: %w", err)
	}
	slackClient, err := s.getSlackClientForIntegration(integrationUUID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Send system message with :gear: emoji prepended
	systemMessage := ":gear: " + payload.Message
	_, _, err = slackClient.PostMessage(processedMessage.SlackChannelID,
		slack.MsgOptionText(utils.ConvertMarkdownToSlack(systemMessage), false),
		slack.MsgOptionTS(job.SlackThreadTS),
	)
	if err != nil {
		return fmt.Errorf("❌ Failed to send system message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(job.ID, slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
	}

	log.Printf("📋 Completed successfully - sent system message to Slack thread %s", job.SlackThreadTS)
	return nil
}

func (s *CoreUseCase) ProcessProcessingSlackMessage(clientID string, payload models.ProcessingSlackMessagePayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process processing slack message notification from client %s", clientID)

	// Validate SlackMessageID is provided
	if payload.SlackMessageID == "" {
		log.Printf("⚠️ Processing slack message notification has no SlackMessageID")
		return fmt.Errorf("SlackMessageID is required")
	}

	utils.AssertInvariant(uuid.Validate(payload.SlackMessageID) == nil, "SlackMessageID is not in UUID format")
	messageID := uuid.MustParse(payload.SlackMessageID)

	// Get the ProcessedSlackMessage to find the correct channel and update emoji
	processedMessage, err := s.jobsService.GetProcessedSlackMessageByID(messageID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get processed slack message %s: %v", messageID, err)
		return fmt.Errorf("failed to get processed slack message: %w", err)
	}

	// Update the slack message reaction to show agent is now processing (eyes emoji)
	if err := s.updateSlackMessageReaction(processedMessage.SlackChannelID, processedMessage.SlackTS, "eyes", slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update slack message reaction to eyes: %w", err)
	}

	log.Printf("📋 Completed successfully - updated slack message emoji to eyes for message %s", messageID)
	return nil
}

func (s *CoreUseCase) ProcessSlackMessageEvent(event models.SlackMessageEvent, slackIntegrationID string) error {
	log.Printf("📋 Starting to process Slack message event from %s in %s: %s", event.User, event.Channel, event.Text)

	// Determine the thread timestamp to use for job lookup/creation
	var threadTS string
	if event.ThreadTs == "" {
		// New thread - use the message timestamp
		threadTS = event.Ts
		log.Printf("🆕 Bot mentioned at start of new thread in channel %s", event.Channel)
	} else {
		// Existing thread - use the thread timestamp
		threadTS = event.ThreadTs
		log.Printf("💬 Bot mentioned in ongoing thread %s in channel %s", event.ThreadTs, event.Channel)
	}

	// Create or get existing job for this slack thread
	jobResult, err := s.jobsService.GetOrCreateJobForSlackThread(threadTS, event.Channel, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get or create job for slack thread: %v", err)
		return fmt.Errorf("failed to get or create job for slack thread: %w", err)
	}

	job := jobResult.Job
	isNewConversation := jobResult.Status == models.JobCreationStatusCreated

	// Check if agents are available first
	connectedClientIDs := s.wsClient.GetClientIDs()
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(slackIntegrationID, connectedClientIDs)
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
		clientID, err = s.getOrAssignAgentForJob(job, threadTS, slackIntegrationID)
		if err != nil {
			return fmt.Errorf("failed to get or assign agent for job: %w", err)
		}
		messageStatus = models.ProcessedSlackMessageStatusInProgress
	}

	// Store the Slack message as ProcessedSlackMessage with appropriate status
	processedMessage, err := s.jobsService.CreateProcessedSlackMessage(
		job.ID,
		event.Channel,
		event.Ts,
		event.Text,
		slackIntegrationID,
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	// Add emoji reaction based on message status
	reactionEmoji := DeriveMessageReactionFromStatus(messageStatus)
	if err := s.updateSlackMessageReaction(processedMessage.SlackChannelID, processedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update slack message reaction: %w", err)
	}

	// If message was queued, don't send to agent yet - background processor will handle it
	if messageStatus == models.ProcessedSlackMessageStatusQueued {
		log.Printf("📋 Message queued for background processing - job %s", job.ID)
		log.Printf("📋 Completed successfully - processed Slack message event (queued)")
		return nil
	}

	// Send work to assigned agent
	if isNewConversation {
		if err := s.sendStartConversationToAgent(clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message: %w", err)
		}
	} else {
		if err := s.sendUserMessageToAgent(clientID, processedMessage); err != nil {
			return fmt.Errorf("failed to send user message: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - processed Slack message event")
	return nil
}

func (s *CoreUseCase) sendStartConversationToAgent(clientID string, message *models.ProcessedSlackMessage) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Generate permalink for the Slack message
	permalink, err := slackClient.GetPermalink(&slack.PermalinkParameters{
		Channel: message.SlackChannelID,
		Ts:      message.SlackTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := utils.ResolveMentionsInSlackMessage(context.Background(), message.TextContent, slackClient)

	startConversationMessage := models.UnknownMessage{
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			JobID:            message.JobID.String(),
			Message:          resolvedText,
			SlackMessageID:   message.ID.String(),
			SlackMessageLink: permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
		return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
	}
	log.Printf("🚀 Sent start conversation message to client %s", clientID)
	return nil
}

func (s *CoreUseCase) sendUserMessageToAgent(clientID string, message *models.ProcessedSlackMessage) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(message.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	permalink, err := slackClient.GetPermalink(&slack.PermalinkParameters{
		Channel: message.SlackChannelID,
		Ts:      message.SlackTS,
	})
	if err != nil {
		return fmt.Errorf("failed to get permalink for slack message: %w", err)
	}

	// Resolve user mentions in the message text before sending to agent
	resolvedText := utils.ResolveMentionsInSlackMessage(context.Background(), message.TextContent, slackClient)

	userMessage := models.UnknownMessage{
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			JobID:            message.JobID.String(),
			Message:          resolvedText,
			SlackMessageID:   message.ID.String(),
			SlackMessageLink: permalink,
		},
	}

	if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
		return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
	}
	log.Printf("💬 Sent user message to client %s", clientID)
	return nil
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

func (s *CoreUseCase) updateSlackMessageReaction(channelID, messageTS, newEmoji, slackIntegrationID string) error {
	// Get integration-specific Slack client
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return fmt.Errorf("invalid slack integration ID format: %w", err)
	}
	slackClient, err := s.getSlackClientForIntegration(integrationUUID)
	if err != nil {
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	// Remove existing reactions
	reactionsToRemove := []string{"eyes", "hourglass", "white_check_mark"}
	for _, emoji := range reactionsToRemove {
		if err := slackClient.RemoveReaction(emoji, slack.ItemRef{
			Channel:   channelID,
			Timestamp: messageTS,
		}); err != nil {
			// Check if it's a no_reaction error (reaction doesn't exist)
			if strings.Contains(err.Error(), "no_reaction") {
				// Ignore no_reaction error and continue
				log.Printf("Note: %s reaction not found on message %s, skipping removal", emoji, messageTS)
				continue
			}
			return fmt.Errorf("failed to remove %s reaction: %w", emoji, err)
		}
	}

	// Add the new reaction
	if newEmoji != "" {
		if err := slackClient.AddReaction(newEmoji, slack.ItemRef{
			Channel:   channelID,
			Timestamp: messageTS,
		}); err != nil {
			return fmt.Errorf("failed to add %s reaction: %w", newEmoji, err)
		}
	}

	return nil
}

func (s *CoreUseCase) getOrAssignAgentForJob(job *models.Job, threadTS, slackIntegrationID string) (string, error) {
	// Check if this job is already assigned to an agent
	existingAgent, err := s.agentsService.GetAgentByJobID(job.ID, slackIntegrationID)
	if err != nil {
		// Job not assigned to any agent yet - need to assign to an available agent
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			return s.assignJobToAvailableAgent(job, threadTS, slackIntegrationID)
		}

		// Some other error occurred
		log.Printf("❌ Failed to check for existing agent assignment: %v", err)
		return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	// Job is already assigned to an agent - verify it still has an active connection
	connectedClientIDs := s.wsClient.GetClientIDs()
	if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
		log.Printf("🔄 Job %s already assigned to agent %s with active connection, routing message to existing agent", job.ID, existingAgent.ID)
		return existingAgent.WSConnectionID, nil
	}

	// Existing agent doesn't have active connection - return error to signal no available agents
	log.Printf("⚠️ Job %s assigned to agent %s but no active WebSocket connection found", job.ID, existingAgent.ID)
	return "", fmt.Errorf("no active agents available for job assignment")
}

// assignJobToAvailableAgent attempts to assign a job to the least loaded available agent
// Returns the WebSocket client ID if successful, empty string if no agents available, or error on failure
func (s *CoreUseCase) assignJobToAvailableAgent(job *models.Job, threadTS, slackIntegrationID string) (string, error) {
	log.Printf("📝 Job %s not yet assigned, looking for any active agent", job.ID)

	clientID, assigned, err := s.tryAssignJobToAgent(job.ID, slackIntegrationID)
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
func (s *CoreUseCase) tryAssignJobToAgent(jobID uuid.UUID, slackIntegrationID string) (string, bool, error) {
	// Get active WebSocket connections first
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	// Get only agents with active connections using centralized service method
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(slackIntegrationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to get connected active agents: %v", err)
		return "", false, fmt.Errorf("failed to get connected active agents: %w", err)
	}

	if len(connectedAgents) == 0 {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", false, nil
	}

	// Sort agents by load (number of assigned jobs) to select the least loaded agent
	sortedAgents, err := s.sortAgentsByLoad(connectedAgents, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to sort agents by load: %v", err)
		return "", false, fmt.Errorf("failed to sort agents by load: %w", err)
	}

	selectedAgent := sortedAgents[0].agent
	log.Printf("🎯 Selected agent %s with %d active messages (least loaded)", selectedAgent.ID, sortedAgents[0].load)

	// Assign the job to the selected agent (agents can now handle multiple jobs simultaneously)
	if err := s.agentsService.AssignAgentToJob(selectedAgent.ID, jobID, slackIntegrationID); err != nil {
		log.Printf("❌ Failed to assign job %s to agent %s: %v", jobID, selectedAgent.ID, err)
		return "", false, fmt.Errorf("failed to assign job to agent: %w", err)
	}

	log.Printf("✅ Assigned job %s to agent %s", jobID, selectedAgent.ID)
	return selectedAgent.WSConnectionID, true, nil
}

type agentWithLoad struct {
	agent *models.ActiveAgent
	load  int
}

func (s *CoreUseCase) sortAgentsByLoad(agents []*models.ActiveAgent, slackIntegrationID string) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := s.agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job assignments for agent %s: %w", agent.ID, err)
		}
		
		// Get count of active messages (IN_PROGRESS or QUEUED) for these jobs
		activeMessageCount, err := s.jobsService.GetActiveMessageCountForJobs(jobIDs, slackIntegrationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get active message count for agent %s: %w", agent.ID, err)
		}
		
		agentsWithLoad = append(agentsWithLoad, agentWithLoad{agent: agent, load: activeMessageCount})
	}

	// Sort by load (ascending - least loaded first)
	sort.Slice(agentsWithLoad, func(i, j int) bool {
		return agentsWithLoad[i].load < agentsWithLoad[j].load
	})

	return agentsWithLoad, nil
}

func (s *CoreUseCase) RegisterAgent(client *clients.Client) error {
	log.Printf("📋 Starting to register agent for client %s", client.ID)

	// Pass the agent ID to CreateActiveAgent
	_, err := s.agentsService.CreateActiveAgent(client.ID, client.SlackIntegrationID, client.AgentID)
	if err != nil {
		return fmt.Errorf("failed to register agent for client %s: %w", client.ID, err)
	}

	log.Printf("📋 Completed successfully - registered agent for client %s", client.ID)
	return nil
}

func (s *CoreUseCase) DeregisterAgent(client *clients.Client) error {
	log.Printf("📋 Starting to deregister agent for client %s", client.ID)

	// First, get the agent to check for assigned jobs
	agent, err := s.agentsService.GetAgentByWSConnectionID(client.ID, client.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find agent for client %s: %w", client.ID, err)
	}

	// Get active jobs for agent cleanup
	jobs, err := s.agentsService.GetActiveAgentJobAssignments(agent.ID, client.SlackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get jobs for cleanup: %v", err)
		return fmt.Errorf("failed to get jobs for cleanup: %w", err)
	}
	if len(jobs) == 0 {
		// No jobs to clean up, just delete the agent
		if err := s.agentsService.DeleteActiveAgentByWsConnectionID(client.ID, client.SlackIntegrationID); err != nil {
			return fmt.Errorf("failed to delete agent: %w", err)
		}
		log.Printf("📋 Completed successfully - deregistered agent for client %s", client.ID)
		return nil
	}

	// Clean up all job assignments and send notification for first job
	log.Printf("🧹 Agent %s has %d assigned job(s), cleaning up all assignments", agent.ID, len(jobs))

	// Get the first job for Slack notification
	job, err := s.jobsService.GetJobByID(jobs[0], client.SlackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s for cleanup: %v", jobs[0], err)
	} else {
		integrationUUID, err := uuid.Parse(client.SlackIntegrationID)
		if err != nil {
			log.Printf("❌ Invalid slack integration ID format %s: %v", client.SlackIntegrationID, err)
			return fmt.Errorf("invalid slack integration ID format: %w", err)
		}
		slackClient, err := s.getSlackClientForIntegration(integrationUUID)
		if err != nil {
			log.Printf("❌ Failed to get Slack client for integration %s: %v", client.SlackIntegrationID, err)
			return fmt.Errorf("failed to get Slack client for integration: %w", err)
		}

		// Send abandonment message to Slack thread
		abandonmentMessage := ":x: The assigned agent was disconnected, abandoning job"
		_, _, err = slackClient.PostMessage(job.SlackChannelID,
			slack.MsgOptionText(utils.ConvertMarkdownToSlack(abandonmentMessage), false),
			slack.MsgOptionTS(job.SlackThreadTS),
		)
		if err != nil {
			log.Printf("⚠️ Failed to send abandonment message to Slack thread %s: %v", job.SlackThreadTS, err)
		} else {
			log.Printf("📤 Sent abandonment message to Slack thread %s", job.SlackThreadTS)
		}

		// Delete the job
		if err := s.jobsService.DeleteJob(job.ID, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to delete abandoned job %s: %v", job.ID, err)
		} else {
			log.Printf("🗑️ Deleted abandoned job %s", job.ID)
		}
	}

	// Unassign agent from all jobs
	for _, jobID := range jobs {
		if err := s.agentsService.UnassignAgentFromJob(agent.ID, jobID, client.SlackIntegrationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent %s from job %s: %w", agent.ID, jobID, err)
		}

		log.Printf("🔗 Unassigned agent %s from job %s", agent.ID, jobID)
	}

	// Delete the agent record
	err = s.agentsService.DeleteActiveAgentByWsConnectionID(client.ID, client.SlackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to deregister agent for client %s: %w", client.ID, err)
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", client.ID)
	return nil
}

func (s *CoreUseCase) BroadcastCheckIdleJobs() error {
	log.Printf("📋 Starting to broadcast CheckIdleJobs to all connected agents")

	// Get all slack integrations to broadcast to agents in each integration
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations()
	if err != nil {
		return fmt.Errorf("failed to get slack integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No slack integrations found")
		return nil
	}

	totalAgentCount := 0
	var broadcastErrors []string
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	for _, integration := range integrations {
		slackIntegrationID := integration.ID.String()

		// Get connected agents for this integration using centralized service method
		connectedAgents, err := s.agentsService.GetConnectedActiveAgents(slackIntegrationID, connectedClientIDs)
		if err != nil {
			broadcastErrors = append(broadcastErrors, fmt.Sprintf("failed to get connected agents for integration %s: %v", slackIntegrationID, err))
			continue
		}

		if len(connectedAgents) == 0 {
			continue
		}

		log.Printf("📡 Broadcasting CheckIdleJobs to %d connected agents for integration %s", len(connectedAgents), slackIntegrationID)

		// Send CheckIdleJobs message to each connected agent
		checkIdleJobsMessage := models.UnknownMessage{
			Type:    models.MessageTypeCheckIdleJobs,
			Payload: models.CheckIdleJobsPayload{},
		}

		for _, agent := range connectedAgents {
			if err := s.wsClient.SendMessage(agent.WSConnectionID, checkIdleJobsMessage); err != nil {
				broadcastErrors = append(broadcastErrors, fmt.Sprintf("failed to send CheckIdleJobs message to agent %s: %v", agent.ID, err))
				continue
			}
			log.Printf("📤 Sent CheckIdleJobs message to agent %s", agent.ID)
			totalAgentCount++
		}
	}

	log.Printf("📋 Completed broadcast - sent CheckIdleJobs to %d agents", totalAgentCount)

	// Return error if there were any broadcast failures
	if len(broadcastErrors) > 0 {
		return fmt.Errorf("CheckIdleJobs broadcast encountered %d errors: %s", len(broadcastErrors), strings.Join(broadcastErrors, "; "))
	}

	log.Printf("📋 Completed successfully - broadcasted CheckIdleJobs to %d agents", totalAgentCount)
	return nil
}

func (s *CoreUseCase) ProcessJobComplete(clientID string, payload models.JobCompletePayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process job complete from client %s: JobID: %s, Reason: %s", clientID, payload.JobID, payload.Reason)

	// Validate JobID format
	jobID, err := uuid.Parse(payload.JobID)
	if err != nil {
		log.Printf("❌ Invalid JobID format from client %s: %v", clientID, err)
		return fmt.Errorf("invalid JobID format: %w", err)
	}

	// Get the job to find the Slack thread information
	job, err := s.jobsService.GetJobByID(jobID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s: %v", jobID, err)
		return fmt.Errorf("failed to get job: %w", err)
	}

	// Get the agent by WebSocket connection ID to verify ownership
	agent, err := s.agentsService.GetAgentByWSConnectionID(clientID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}

	// Validate that this agent is actually assigned to this job
	if err := s.validateJobBelongsToAgent(agent.ID, jobID, slackIntegrationID); err != nil {
		log.Printf("❌ Agent %s not assigned to job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("agent not assigned to job: %w", err)
	}

	// Set white_check_mark emoji on the top-level message to indicate job completion
	if err := s.updateSlackMessageReaction(job.SlackChannelID, job.SlackThreadTS, "white_check_mark", slackIntegrationID); err != nil {
		log.Printf("⚠️ Failed to update top-level message reaction for completed job %s: %v", jobID, err)
		// Don't return error - this is not critical to job completion
	}

	// Unassign the agent from the job
	if err := s.agentsService.UnassignAgentFromJob(agent.ID, jobID, slackIntegrationID); err != nil {
		log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}
	log.Printf("✅ Unassigned agent %s from completed job %s", agent.ID, jobID)

	// Delete the job and its associated processed messages
	if err := s.jobsService.DeleteJob(jobID, slackIntegrationID); err != nil {
		log.Printf("❌ Failed to delete completed job %s: %v", jobID, err)
		return fmt.Errorf("failed to delete completed job: %w", err)
	}
	log.Printf("🗑️ Deleted completed job %s", jobID)

	// Send completion message to Slack thread with reason
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		log.Printf("❌ Invalid slack integration ID format: %v", err)
		return fmt.Errorf("invalid slack integration ID format: %w", err)
	}
	slackClient, err := s.getSlackClientForIntegration(integrationUUID)
	if err != nil {
		log.Printf("❌ Failed to get Slack client for integration: %v", err)
		return fmt.Errorf("failed to get Slack client for integration: %w", err)
	}

	completionMessage := fmt.Sprintf(":gear: %s", payload.Reason)
	_, _, err = slackClient.PostMessage(job.SlackChannelID,
		slack.MsgOptionText(utils.ConvertMarkdownToSlack(completionMessage), false),
		slack.MsgOptionTS(job.SlackThreadTS),
	)
	if err != nil {
		log.Printf("❌ Failed to send completion message to Slack thread %s: %v", job.SlackThreadTS, err)
		return fmt.Errorf("failed to send completion message to Slack: %w", err)
	}

	log.Printf("📤 Sent completion message to Slack thread %s: %s", job.SlackThreadTS, completionMessage)
	log.Printf("📋 Completed successfully - processed job complete for job %s", jobID)
	return nil
}


func (s *CoreUseCase) ProcessHealthcheckAck(clientID string, payload models.HealthcheckAckPayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process healthcheck ack from client %s", clientID)

	// Update the last_active_at timestamp for this agent
	if err := s.agentsService.UpdateAgentLastActiveAt(clientID, slackIntegrationID); err != nil {
		log.Printf("❌ Failed to update last_active_at for client %s: %v", clientID, err)
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated last_active_at for client %s", clientID)
	return nil
}

func (s *CoreUseCase) SendHealthcheckAck(clientID string, slackIntegrationID string) error {
	log.Printf("📋 Starting to send healthcheck ack to client %s", clientID)
	
	// Create healthcheck ack message
	healthcheckAckMsg := models.UnknownMessage{
		Type:    models.MessageTypeHealthcheckAck,
		Payload: models.HealthcheckAckPayload{},
	}
	
	// Send the message to the client
	if err := s.wsClient.SendMessage(clientID, healthcheckAckMsg); err != nil {
		log.Printf("❌ Failed to send healthcheck ack to client %s: %v", clientID, err)
		return fmt.Errorf("failed to send healthcheck ack: %w", err)
	}
	
	log.Printf("📋 Completed successfully - sent healthcheck ack to client %s", clientID)
	return nil
}


func (s *CoreUseCase) BroadcastHealthcheck() error {
	log.Printf("📋 Starting to broadcast healthcheck to all connected agents")

	// Get all slack integrations to broadcast to agents in each integration
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations()
	if err != nil {
		return fmt.Errorf("failed to get slack integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No slack integrations found")
		return nil
	}

	totalAgentCount := 0
	var broadcastErrors []string
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	for _, integration := range integrations {
		slackIntegrationID := integration.ID.String()

		// Get connected agents for this integration using centralized service method
		connectedAgents, err := s.agentsService.GetConnectedActiveAgents(slackIntegrationID, connectedClientIDs)
		if err != nil {
			broadcastErrors = append(broadcastErrors, fmt.Sprintf("failed to get connected agents for integration %s: %v", slackIntegrationID, err))
			continue
		}

		if len(connectedAgents) == 0 {
			continue
		}

		log.Printf("💓 Broadcasting healthcheck to %d connected agents for integration %s", len(connectedAgents), slackIntegrationID)

		// Send healthcheck message to each connected agent
		healthcheckMessage := models.UnknownMessage{
			Type:    models.MessageTypeHealthcheckCheck,
			Payload: models.HealthcheckCheckPayload{},
		}

		for _, agent := range connectedAgents {
			if err := s.wsClient.SendMessage(agent.WSConnectionID, healthcheckMessage); err != nil {
				broadcastErrors = append(broadcastErrors, fmt.Sprintf("failed to send healthcheck message to agent %s: %v", agent.ID, err))
				continue
			}
			log.Printf("💓 Sent healthcheck message to agent %s", agent.ID)
			totalAgentCount++
		}
	}

	log.Printf("📋 Completed broadcast - sent healthcheck to %d agents", totalAgentCount)

	// Return error if there were any broadcast failures
	if len(broadcastErrors) > 0 {
		return fmt.Errorf("healthcheck broadcast encountered %d errors: %s", len(broadcastErrors), strings.Join(broadcastErrors, "; "))
	}

	log.Printf("📋 Completed successfully - broadcasted healthcheck to %d agents", totalAgentCount)
	return nil
}

func (s *CoreUseCase) ProcessQueuedJobs() error {
	log.Printf("📋 Starting to process queued jobs")

	// Get all slack integrations
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations()
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
		slackIntegrationID := integration.ID.String()

		// Get jobs with queued messages for this integration
		queuedJobs, err := s.jobsService.GetJobsWithQueuedMessages(slackIntegrationID)
		if err != nil {
			processingErrors = append(processingErrors, fmt.Sprintf("failed to get queued jobs for integration %s: %v", slackIntegrationID, err))
			continue
		}

		if len(queuedJobs) == 0 {
			continue
		}

		log.Printf("🔍 Found %d jobs with queued messages for integration %s", len(queuedJobs), slackIntegrationID)

		// Try to assign each queued job to an available agent
		for _, job := range queuedJobs {
			log.Printf("🔄 Processing queued job %s", job.ID)

			// Try to assign job to an available agent
			clientID, assigned, err := s.tryAssignJobToAgent(job.ID, slackIntegrationID)
			if err != nil {
				processingErrors = append(processingErrors, fmt.Sprintf("failed to assign queued job %s: %v", job.ID, err))
				continue
			}

			if !assigned {
				log.Printf("⚠️ Still no agents available for queued job %s", job.ID)
				continue
			}

			// Job was successfully assigned - get queued messages and send them to agent
			queuedMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusQueued, slackIntegrationID)
			if err != nil {
				processingErrors = append(processingErrors, fmt.Sprintf("failed to get queued messages for job %s: %v", job.ID, err))
				continue
			}

			log.Printf("📨 Found %d queued messages for job %s", len(queuedMessages), job.ID)

			// Process each queued message
			for _, message := range queuedMessages {
				// Update message status to IN_PROGRESS
				updatedMessage, err := s.jobsService.UpdateProcessedSlackMessage(message.ID, models.ProcessedSlackMessageStatusInProgress, slackIntegrationID)
				if err != nil {
					processingErrors = append(processingErrors, fmt.Sprintf("failed to update message %s status: %v", message.ID, err))
					continue
				}

				// Update Slack reaction to show processing (eyes emoji)
				if err := s.updateSlackMessageReaction(updatedMessage.SlackChannelID, updatedMessage.SlackTS, "eyes", slackIntegrationID); err != nil {
					log.Printf("⚠️ Failed to update slack reaction for message %s: %v", message.ID, err)
				}

				// Determine if this is the first message in the job (new conversation)
				allMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusCompleted, slackIntegrationID)
				if err != nil {
					log.Printf("⚠️ Failed to check for existing messages in job %s: %v", job.ID, err)
				}
				isNewConversation := len(allMessages) == 0

				// Send work to assigned agent
				if isNewConversation {
					if err := s.sendStartConversationToAgent(clientID, updatedMessage); err != nil {
						processingErrors = append(processingErrors, fmt.Sprintf("failed to send start conversation for message %s: %v", message.ID, err))
						continue
					}
				} else {
					if err := s.sendUserMessageToAgent(clientID, updatedMessage); err != nil {
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
		return fmt.Errorf("queued job processing encountered %d errors: %s", len(processingErrors), strings.Join(processingErrors, "; "))
	}

	log.Printf("📋 Completed successfully - processed %d queued jobs", totalProcessedJobs)
	return nil
}

func (s *CoreUseCase) CleanupInactiveAgents() error {
	log.Printf("📋 Starting to cleanup inactive agents (>10 minutes)")

	// Get all slack integrations
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations()
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
		slackIntegrationID := integration.ID.String()

		// Get inactive agents for this integration
		inactiveAgents, err := s.agentsService.GetInactiveAgents(slackIntegrationID, inactiveThresholdMinutes)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to get inactive agents for integration %s: %v", slackIntegrationID, err))
			continue
		}

		if len(inactiveAgents) == 0 {
			continue
		}

		log.Printf("🔍 Found %d inactive agents for integration %s", len(inactiveAgents), slackIntegrationID)

		// Delete each inactive agent
		for _, agent := range inactiveAgents {
			log.Printf("🧹 Found inactive agent %s (last active: %s) - cleaning up", agent.ID, agent.LastActiveAt.Format("2006-01-02 15:04:05"))

			// Delete the inactive agent - CASCADE DELETE will automatically clean up job assignments
			if err := s.agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete inactive agent %s: %v", agent.ID, err))
				continue
			}

			log.Printf("✅ Deleted inactive agent %s (CASCADE DELETE cleaned up job assignments)", agent.ID)
			totalInactiveAgents++
		}
	}

	log.Printf("📋 Completed cleanup - removed %d inactive agents", totalInactiveAgents)

	// Return error if there were any cleanup failures
	if len(cleanupErrors) > 0 {
		return fmt.Errorf("inactive agent cleanup encountered %d errors: %s", len(cleanupErrors), strings.Join(cleanupErrors, "; "))
	}

	log.Printf("📋 Completed successfully - cleaned up %d inactive agents", totalInactiveAgents)
	return nil
}
