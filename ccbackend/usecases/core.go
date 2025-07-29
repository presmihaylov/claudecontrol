package usecases

import (
	"fmt"
	"log"
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

func (s *CoreUseCase) getSlackClientForIntegration(slackIntegrationID string) (*slack.Client, error) {
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("invalid slack integration ID format: %w", err)
	}

	integration, err := s.slackIntegrationsService.GetSlackIntegrationByID(integrationUUID)
	if err != nil {
		return nil, fmt.Errorf("failed to get slack integration: %w", err)
	}

	return slack.New(integration.SlackAuthToken), nil
}

func (s *CoreUseCase) ProcessAssistantMessage(clientID string, payload models.AssistantMessagePayload, slackIntegrationID string) error {
	log.Printf("📋 Starting to process assistant message from client %s: %s", clientID, payload.Message)

	// Get the agent by WebSocket connection ID
	agent, err := s.agentsService.GetAgentByWSConnectionID(clientID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}

	// Get active jobs for agent
	jobs, err := s.agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get jobs for agent: %w", err)
	}
	if len(jobs) == 0 {
		log.Printf("⚠️ Agent %s has no assigned jobs, cannot determine Slack thread", agent.ID)
		return nil
	}

	// Get the job to find the Slack thread information (use first job for current single-job behavior)
	job, err := s.jobsService.GetJobByID(jobs[0], slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s for agent %s: %v", jobs[0], agent.ID, err)
		return fmt.Errorf("failed to get job for agent: %w", err)
	}

	log.Printf("📤 Sending assistant message to Slack thread %s in channel %s", job.SlackThreadTS, job.SlackChannelID)

	// Handle empty message from Claude
	messageToSend := payload.Message
	if strings.TrimSpace(messageToSend) == "" {
		messageToSend = "(agent sent empty response)"
		log.Printf("⚠️ Agent sent empty response, using fallback message")
	}

	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
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
	reactionEmoji := DeriveMessageReactionFromStatus(models.ProcessedSlackMessageStatusCompleted)
	if err := s.updateSlackMessageReaction(updatedMessage.SlackChannelID, updatedMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update slack message reaction: %w", err)
	}

	// Check for any queued messages and process the next one
	queuedMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusQueued, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to check for queued messages: %w", err)
	}

	if len(queuedMessages) > 0 {
		// Get the oldest queued message (first in the sorted list)
		nextMessage := queuedMessages[0]
		log.Printf("📨 Processing next queued message for job %s (SlackTS: %s)", job.ID, nextMessage.SlackTS)

		// Update the queued message to IN_PROGRESS
		updatedNextMessage, err := s.jobsService.UpdateProcessedSlackMessage(nextMessage.ID, models.ProcessedSlackMessageStatusInProgress, slackIntegrationID)
		if err != nil {
			return fmt.Errorf("failed to update queued message status to IN_PROGRESS: %w", err)
		}

		// Update emoji reaction to show processing
		reactionEmoji := DeriveMessageReactionFromStatus(models.ProcessedSlackMessageStatusInProgress)
		if err := s.updateSlackMessageReaction(updatedNextMessage.SlackChannelID, updatedNextMessage.SlackTS, reactionEmoji, slackIntegrationID); err != nil {
			return fmt.Errorf("failed to update slack message reaction for queued message: %w", err)
		}

		// Send the queued message to agent (always user message since start conversation only happens for new threads)
		if err := s.sendUserMessageToAgent(clientID, nextMessage); err != nil {
			return fmt.Errorf("failed to send queued user message to agent: %w", err)
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
	slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
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

	// Check if this job is already assigned to an agent or assign to new agent
	clientID, err := s.getOrAssignAgentForJob(job, threadTS, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to get or assign agent for job: %w", err)
	}

	if clientID == "" {
		log.Printf("⚠️ No available agents to handle Slack mention")

		// Get integration-specific Slack client
		slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
		if err != nil {
			return fmt.Errorf("failed to get Slack client for integration: %w", err)
		}

		// Send message to Slack informing that no agents are available
		_, _, err = slackClient.PostMessage(event.Channel,
			slack.MsgOptionText(utils.ConvertMarkdownToSlack("There are no available agents to handle this job"), false),
			slack.MsgOptionTS(threadTS),
		)
		if err != nil {
			log.Printf("❌ Failed to send 'no agents available' message to Slack: %v", err)
			return fmt.Errorf("failed to send 'no agents available' message to Slack: %w", err)
		}

		log.Printf("📤 Sent 'no agents available' message to Slack thread %s in channel %s", threadTS, event.Channel)
		return nil
	}

	// Check if there are any IN_PROGRESS messages for this job
	inProgressMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusInProgress, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to check for in progress messages: %w", err)
	}

	// Determine the status for the new message
	var messageStatus models.ProcessedSlackMessageStatus
	if len(inProgressMessages) > 0 {
		// There's already an IN_PROGRESS message, so queue this one
		messageStatus = models.ProcessedSlackMessageStatusQueued
		log.Printf("⏳ Message will be queued - found %d in progress message(s) for job %s", len(inProgressMessages), job.ID)
	} else {
		// No IN_PROGRESS messages, process immediately
		messageStatus = models.ProcessedSlackMessageStatusInProgress
	}

	// Store the Slack message as ProcessedSlackMessage
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

	// Only send to agent if status is IN_PROGRESS (not queued)
	if messageStatus == models.ProcessedSlackMessageStatusInProgress {
		// Send appropriate message to the assigned agent based on whether this is a new conversation
		if isNewConversation {
			if err := s.sendStartConversationToAgent(clientID, processedMessage); err != nil {
				return fmt.Errorf("failed to send start conversation message: %w", err)
			}
		} else {
			if err := s.sendUserMessageToAgent(clientID, processedMessage); err != nil {
				return fmt.Errorf("failed to send user message: %w", err)
			}
		}
	} else {
		log.Printf("📥 Message queued for job %s (SlackTS: %s)", job.ID, event.Ts)
	}

	log.Printf("📋 Completed successfully - processed Slack message event")
	return nil
}

func (s *CoreUseCase) sendStartConversationToAgent(clientID string, message *models.ProcessedSlackMessage) error {
	// Get integration-specific Slack client
	slackClient, err := s.getSlackClientForIntegration(message.SlackIntegrationID.String())
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

	startConversationMessage := models.UnknownMessage{
		Type: models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			Message:          message.TextContent,
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
	slackClient, err := s.getSlackClientForIntegration(message.SlackIntegrationID.String())
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

	userMessage := models.UnknownMessage{
		Type: models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			Message:          message.TextContent,
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
		return "eyes"
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
	slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
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
			log.Printf("📝 Job %s not yet assigned, looking for available agent", job.ID)

			availableAgents, err := s.agentsService.GetAvailableAgents(slackIntegrationID)
			if err != nil {
				log.Printf("❌ Failed to get available agents: %v", err)
				return "", fmt.Errorf("failed to get available agents: %w", err)
			}

			if len(availableAgents) == 0 {
				// Return empty string to signal no available agents
				return "", nil
			}

			firstAgent := availableAgents[0]

			// Assign the job to the selected agent
			if err := s.agentsService.AssignAgentToJob(firstAgent.ID, job.ID, slackIntegrationID); err != nil {
				log.Printf("❌ Failed to assign job %s to agent %s: %v", job.ID, firstAgent.ID, err)
				return "", fmt.Errorf("failed to assign job to agent: %w", err)
			}

			log.Printf("✅ Assigned job %s to agent %s for slack thread %s", job.ID, firstAgent.ID, threadTS)
			return firstAgent.WSConnectionID, nil
		}

		// Some other error occurred
		log.Printf("❌ Failed to check for existing agent assignment: %v", err)
		return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	// Job is already assigned to an agent - use that agent
	log.Printf("🔄 Job %s already assigned to agent %s, routing message to existing agent", job.ID, existingAgent.ID)
	return existingAgent.WSConnectionID, nil
}

func (s *CoreUseCase) RegisterAgent(clientID string) error {
	log.Printf("📋 Starting to register agent for client %s", clientID)

	// Get the slack integration ID from the WebSocket client
	slackIntegrationID := s.wsClient.GetSlackIntegrationIDByClientID(clientID)
	if slackIntegrationID == "" {
		log.Printf("❌ Failed to get slack integration ID for client %s", clientID)
		return fmt.Errorf("no slack integration ID found for client %s", clientID)
	}

	_, err := s.agentsService.CreateActiveAgent(clientID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to register agent for client %s: %w", clientID, err)
	}

	log.Printf("📋 Completed successfully - registered agent for client %s", clientID)
	return nil
}

func (s *CoreUseCase) DeregisterAgent(clientID string) error {
	log.Printf("📋 Starting to deregister agent for client %s", clientID)

	// Get the slack integration ID from the WebSocket client
	slackIntegrationID := s.wsClient.GetSlackIntegrationIDByClientID(clientID)
	if slackIntegrationID == "" {
		log.Printf("⚠️ No slack integration ID found for client %s, cannot deregister properly", clientID)
		return fmt.Errorf("no slack integration ID found for client %s", clientID)
	}

	// First, get the agent to check for assigned jobs
	agent, err := s.agentsService.GetAgentByWSConnectionID(clientID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find agent for client %s: %w", clientID, err)
	}

	// Get active jobs for agent cleanup
	jobs, err := s.agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get jobs for cleanup: %v", err)
		return fmt.Errorf("failed to get jobs for cleanup: %w", err)
	}
	if len(jobs) == 0 {
		// No jobs to clean up, just delete the agent
		if err := s.agentsService.DeleteActiveAgentByWsConnectionID(clientID, slackIntegrationID); err != nil {
			return fmt.Errorf("failed to delete agent: %w", err)
		}
		log.Printf("📋 Completed successfully - deregistered agent for client %s", clientID)
		return nil
	}

	// Clean up all job assignments and send notification for first job
	log.Printf("🧹 Agent %s has %d assigned job(s), cleaning up all assignments", agent.ID, len(jobs))

	// Get the first job for Slack notification
	job, err := s.jobsService.GetJobByID(jobs[0], slackIntegrationID)
	if err != nil {
		log.Printf("❌ Failed to get job %s for cleanup: %v", jobs[0], err)
	} else {
		slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
		if err != nil {
			log.Printf("❌ Failed to get Slack client for integration %s: %v", slackIntegrationID, err)
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
		if err := s.jobsService.DeleteJob(job.ID, slackIntegrationID); err != nil {
			log.Printf("❌ Failed to delete abandoned job %s: %v", job.ID, err)
		} else {
			log.Printf("🗑️ Deleted abandoned job %s", job.ID)
		}
	}

	// Unassign agent from all jobs
	for _, jobID := range jobs {
		if err := s.agentsService.UnassignAgentFromJob(agent.ID, jobID, slackIntegrationID); err != nil {
			log.Printf("❌ Failed to unassign agent %s from job %s: %v", agent.ID, jobID, err)
			return fmt.Errorf("failed to unassign agent %s from job %s: %w", agent.ID, jobID, err)
		}

		log.Printf("🔗 Unassigned agent %s from job %s", agent.ID, jobID)
	}

	// Delete the agent record
	err = s.agentsService.DeleteActiveAgentByWsConnectionID(clientID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to deregister agent for client %s: %w", clientID, err)
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", clientID)
	return nil
}

func (s *CoreUseCase) CleanupIdleJobs() error {
	log.Printf("📋 Starting to cleanup idle jobs older than 5 minutes")

	// Get jobs that haven't been updated in the last 5 minutes across all integrations
	idleJobs, err := s.jobsService.GetIdleJobs(5)
	if err != nil {
		return fmt.Errorf("failed to get idle jobs: %w", err)
	}

	if len(idleJobs) == 0 {
		log.Printf("📋 No idle jobs found")
		return nil
	}

	log.Printf("🧹 Found %d idle jobs to cleanup across all integrations", len(idleJobs))

	var cleanupErrors []string
	for _, job := range idleJobs {
		slackIntegrationID := job.SlackIntegrationID.String()

		// Check if this job has an assigned agent
		assignedAgent, err := s.agentsService.GetAgentByJobID(job.ID, slackIntegrationID)
		if err != nil {
			if !strings.Contains(fmt.Sprintf("%v", err), "not found") {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to check agent assignment for job %s: %v", job.ID, err))
				continue
			}
		} else {
			if err := s.agentsService.UnassignAgentFromJob(assignedAgent.ID, job.ID, slackIntegrationID); err != nil {
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to unassign agent %s from idle job %s: %v", assignedAgent.ID, job.ID, err))
				continue
			}
			log.Printf("✅ Unassigned agent %s from idle job %s", assignedAgent.ID, job.ID)

			// Send JobUnassigned message to the agent after successful unassignment
			jobUnassignedMessage := models.UnknownMessage{
				Type:    models.MessageTypeJobUnassigned,
				Payload: models.JobUnassignedPayload{},
			}

			if err := s.wsClient.SendMessage(assignedAgent.WSConnectionID, jobUnassignedMessage); err != nil {
				log.Printf("⚠️ Failed to send JobUnassigned message to agent %s: %v", assignedAgent.ID, err)
			} else {
				log.Printf("📤 Sent JobUnassigned message to agent %s", assignedAgent.ID)
			}
		}

		// Delete the idle job
		if err := s.jobsService.DeleteJob(job.ID, slackIntegrationID); err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete idle job %s: %v", job.ID, err))
			continue
		}

		log.Printf("🗑️ Deleted idle job %s (thread: %s)", job.ID, job.SlackThreadTS)

		// Send completion message to Slack thread
		slackClient, err := s.getSlackClientForIntegration(slackIntegrationID)
		if err != nil {
			log.Printf("⚠️ Failed to get Slack client for integration: %v", err)
			continue
		}

		completionMessage := "This job is now complete"
		_, _, err = slackClient.PostMessage(job.SlackChannelID,
			slack.MsgOptionText(utils.ConvertMarkdownToSlack(completionMessage), false),
			slack.MsgOptionTS(job.SlackThreadTS),
		)
		if err != nil {
			log.Printf("⚠️ Failed to send completion message to Slack thread %s: %v", job.SlackThreadTS, err)
		} else {
			log.Printf("📤 Sent completion message to Slack thread %s", job.SlackThreadTS)
		}
	}

	log.Printf("📋 Completed cleanup - processed %d idle jobs", len(idleJobs))

	// Return error if there were any cleanup failures
	if len(cleanupErrors) > 0 {
		return fmt.Errorf("idle job cleanup encountered %d errors: %s", len(cleanupErrors), strings.Join(cleanupErrors, "; "))
	}

	log.Printf("📋 Completed successfully - cleaned up %d idle jobs", len(idleJobs))
	return nil
}

func (s *CoreUseCase) CleanupStaleActiveAgents() error {
	log.Printf("📋 Starting to cleanup stale active agents")

	// Get all slack integrations
	integrations, err := s.slackIntegrationsService.GetAllSlackIntegrations()
	if err != nil {
		return fmt.Errorf("failed to get slack integrations: %w", err)
	}

	if len(integrations) == 0 {
		log.Printf("📋 No slack integrations found")
		return nil
	}

	totalStaleAgents := 0
	var cleanupErrors []string
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	// Create a map for faster lookup
	connectedClientsMap := make(map[string]bool)
	for _, clientID := range connectedClientIDs {
		connectedClientsMap[clientID] = true
	}

	for _, integration := range integrations {
		slackIntegrationID := integration.ID.String()
		
		// Get all active agents for this integration
		agents, err := s.agentsService.GetAllActiveAgents(slackIntegrationID)
		if err != nil {
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to get active agents for integration %s: %v", slackIntegrationID, err))
			continue
		}

		if len(agents) == 0 {
			continue
		}

		log.Printf("🔍 Checking %d active agents for integration %s", len(agents), slackIntegrationID)

		// Check each agent to see if their WebSocket connection still exists
		for _, agent := range agents {
			if !connectedClientsMap[agent.WSConnectionID] {
				log.Printf("🧹 Found stale agent %s (WebSocket ID: %s) - no corresponding connection", agent.ID, agent.WSConnectionID)
				
				// Get job assignments for notification purposes (optional - don't fail cleanup if this fails)
				jobs, err := s.agentsService.GetActiveAgentJobAssignments(agent.ID, slackIntegrationID)
				if err != nil {
					log.Printf("⚠️ Failed to get jobs for stale agent %s notification: %v", agent.ID, err)
				} else if len(jobs) > 0 {
					log.Printf("📤 Stale agent %s had %d assigned job(s) - jobs will become unassigned", agent.ID, len(jobs))
					
					// Send notification to first job's Slack thread (best effort - don't fail cleanup if this fails)
					if firstJob, err := s.jobsService.GetJobByID(jobs[0], slackIntegrationID); err == nil {
						if slackClient, err := s.getSlackClientForIntegration(slackIntegrationID); err == nil {
							abandonmentMessage := ":warning: Agent disconnected - job is now unassigned and available for pickup"
							slackClient.PostMessage(firstJob.SlackChannelID,
								slack.MsgOptionText(utils.ConvertMarkdownToSlack(abandonmentMessage), false),
								slack.MsgOptionTS(firstJob.SlackThreadTS),
							)
							log.Printf("📤 Sent stale agent notification to Slack thread %s", firstJob.SlackThreadTS)
						}
					}
				}
				
				// Delete the stale agent - CASCADE DELETE will automatically clean up job assignments
				if err := s.agentsService.DeleteActiveAgent(agent.ID, slackIntegrationID); err != nil {
					cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to delete stale agent %s: %v", agent.ID, err))
				} else {
					log.Printf("✅ Deleted stale agent %s (CASCADE DELETE cleaned up job assignments)", agent.ID)
					totalStaleAgents++
				}
			}
		}
	}

	log.Printf("📋 Completed cleanup - removed %d stale active agents", totalStaleAgents)

	// Return error if there were any cleanup failures
	if len(cleanupErrors) > 0 {
		return fmt.Errorf("stale agent cleanup encountered %d errors: %s", len(cleanupErrors), strings.Join(cleanupErrors, "; "))
	}

	log.Printf("📋 Completed successfully - cleaned up %d stale active agents", totalStaleAgents)
	return nil
}
