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
	slackClient   *slack.Client
	wsClient      *clients.WebSocketClient
	agentsService *services.AgentsService
	jobsService   *services.JobsService
}

func NewCoreUseCase(slackClient *slack.Client, wsClient *clients.WebSocketClient, agentsService *services.AgentsService, jobsService *services.JobsService) *CoreUseCase {
	return &CoreUseCase{
		slackClient:   slackClient,
		wsClient:      wsClient,
		agentsService: agentsService,
		jobsService:   jobsService,
	}
}

func (s *CoreUseCase) ProcessAssistantMessage(clientID string, payload models.AssistantMessagePayload) error {
	log.Printf("📋 Starting to process assistant message from client %s: %s", clientID, payload.Message)

	// Get the agent by WebSocket connection ID
	agent, err := s.agentsService.GetAgentByWSConnectionID(clientID)
	if err != nil {
		log.Printf("❌ Failed to find agent for client %s: %v", clientID, err)
		return fmt.Errorf("failed to find agent for client: %w", err)
	}

	// Check if agent has an assigned job
	if agent.AssignedJobID == nil {
		log.Printf("⚠️ Agent %s has no assigned job, cannot determine Slack thread", agent.ID)
		return nil
	}

	// Get the job to find the Slack thread information
	job, err := s.jobsService.GetJobByID(*agent.AssignedJobID)
	if err != nil {
		log.Printf("❌ Failed to get job %s for agent %s: %v", *agent.AssignedJobID, agent.ID, err)
		return fmt.Errorf("failed to get job for agent: %w", err)
	}

	log.Printf("📤 Sending assistant message to Slack thread %s in channel %s", job.SlackThreadTS, job.SlackChannelID)

	_, _, err = s.slackClient.PostMessage(job.SlackChannelID,
		slack.MsgOptionText(utils.ConvertMarkdownToSlack(payload.Message), false),
		slack.MsgOptionTS(job.SlackThreadTS),
	)
	if err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
	}

	// Update the ProcessedSlackMessage status to COMPLETED
	utils.AssertInvariant(payload.SlackMessageID != "", "SlackMessageID is empty")
	utils.AssertInvariant(uuid.Validate(payload.SlackMessageID) == nil, "SlackMessageID is not in UUID format")
	
	messageID := uuid.MustParse(payload.SlackMessageID)
	
	if err := s.jobsService.UpdateProcessedSlackMessage(messageID, models.ProcessedSlackMessageStatusCompleted); err != nil {
		return fmt.Errorf("failed to update processed slack message status: %w", err)
	}

	// Check for any queued messages and process the next one
	queuedMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusQueued)
	if err != nil {
		return fmt.Errorf("failed to check for queued messages: %w", err)
	}

	if len(queuedMessages) > 0 {
		// Get the oldest queued message (first in the sorted list)
		nextMessage := queuedMessages[0]
		log.Printf("📨 Processing next queued message for job %s (SlackTS: %s)", job.ID, nextMessage.SlackTS)
		
		// Update the queued message to IN_PROGRESS
		if err := s.jobsService.UpdateProcessedSlackMessage(nextMessage.ID, models.ProcessedSlackMessageStatusInProgress); err != nil {
			return fmt.Errorf("failed to update queued message status to IN_PROGRESS: %w", err)
		}

		// Send the queued message to agent (always user message since start conversation only happens for new threads)
		if err := s.sendUserMessageToAgent(clientID, nextMessage); err != nil {
			return fmt.Errorf("failed to send queued user message to agent: %w", err)
		}
	}

	log.Printf("📋 Completed successfully - sent assistant message to Slack thread %s", job.SlackThreadTS)
	return nil
}

func (s *CoreUseCase) ProcessSlackMessageEvent(event models.SlackMessageEvent) error {
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
	job, err := s.jobsService.GetOrCreateJobForSlackThread(threadTS, event.Channel)
	if err != nil {
		log.Printf("❌ Failed to get or create job for slack thread: %v", err)
		return fmt.Errorf("failed to get or create job for slack thread: %w", err)
	}

	// Check if this job is already assigned to an agent or assign to new agent
	clientID, err := s.getOrAssignAgentForJob(job, threadTS)
	if err != nil {
		return fmt.Errorf("failed to get or assign agent for job: %w", err)
	}

	if clientID == "" {
		log.Printf("⚠️ No available agents to handle Slack mention")

		// Send message to Slack informing that no agents are available
		_, _, err := s.slackClient.PostMessage(event.Channel,
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
	inProgressMessages, err := s.jobsService.GetProcessedMessagesByJobIDAndStatus(job.ID, models.ProcessedSlackMessageStatusInProgress)
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
		messageStatus,
	)
	if err != nil {
		return fmt.Errorf("failed to create processed slack message: %w", err)
	}

	// Only send to agent if status is IN_PROGRESS (not queued)
	if messageStatus == models.ProcessedSlackMessageStatusInProgress {
		// Send appropriate message to the assigned agent
		if event.ThreadTs == "" {
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
	startConversationMessage := models.UnknownMessage{
		Type:    models.MessageTypeStartConversation,
		Payload: models.StartConversationPayload{
			Message:        message.TextContent,
			SlackMessageID: message.ID.String(),
		},
	}

	if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
		return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
	}
	log.Printf("🚀 Sent start conversation message to client %s", clientID)
	return nil
}

func (s *CoreUseCase) sendUserMessageToAgent(clientID string, message *models.ProcessedSlackMessage) error {
	userMessage := models.UnknownMessage{
		Type:    models.MessageTypeUserMessage,
		Payload: models.UserMessagePayload{
			Message:        message.TextContent,
			SlackMessageID: message.ID.String(),
		},
	}

	if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
		return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
	}
	log.Printf("💬 Sent user message to client %s", clientID)
	return nil
}

func (s *CoreUseCase) getOrAssignAgentForJob(job *models.Job, threadTS string) (string, error) {
	// Check if this job is already assigned to an agent
	existingAgent, err := s.agentsService.GetAgentByJobID(job.ID)
	if err != nil {
		// Job not assigned to any agent yet - need to assign to an available agent
		if strings.Contains(fmt.Sprintf("%v", err), "not found") {
			log.Printf("📝 Job %s not yet assigned, looking for available agent", job.ID)

			availableAgents, err := s.agentsService.GetAvailableAgents()
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
			if err := s.agentsService.AssignJobToAgent(firstAgent.ID, job.ID); err != nil {
				log.Printf("❌ Failed to assign job %s to agent %s: %v", job.ID, firstAgent.ID, err)
				return "", fmt.Errorf("failed to assign job to agent: %w", err)
			}

			log.Printf("✅ Assigned job %s to agent %s for slack thread %s", job.ID, firstAgent.ID, threadTS)
			return firstAgent.WSConnectionID, nil
		} else {
			// Some other error occurred
			log.Printf("❌ Failed to check for existing agent assignment: %v", err)
			return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
		}
	} else {
		// Job is already assigned to an agent - use that agent
		log.Printf("🔄 Job %s already assigned to agent %s, routing message to existing agent", job.ID, existingAgent.ID)
		return existingAgent.WSConnectionID, nil
	}
}

func (s *CoreUseCase) RegisterAgent(clientID string) {
	log.Printf("📋 Starting to register agent for client %s", clientID)

	_, err := s.agentsService.CreateActiveAgent(clientID, nil)
	if err != nil {
		log.Printf("❌ Failed to register agent for client %s: %v", clientID, err)
		return
	}

	log.Printf("📋 Completed successfully - registered agent for client %s", clientID)
}

func (s *CoreUseCase) DeregisterAgent(clientID string) {
	log.Printf("📋 Starting to deregister agent for client %s", clientID)

	err := s.agentsService.DeleteActiveAgentByWsConnectionID(clientID)
	if err != nil {
		log.Printf("❌ Failed to deregister agent for client %s: %v", clientID, err)
		return
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", clientID)
}

func (s *CoreUseCase) CleanupIdleJobs() {
	log.Printf("📋 Starting to cleanup idle jobs older than 2 minutes")

	// Get jobs that haven't been updated in the last 2 minutes
	idleJobs, err := s.jobsService.GetIdleJobs(5)
	if err != nil {
		log.Printf("❌ Failed to get idle jobs: %v", err)
		return
	}

	if len(idleJobs) == 0 {
		log.Printf("📋 No idle jobs found")
		return
	}

	log.Printf("🧹 Found %d idle jobs to cleanup", len(idleJobs))

	for _, job := range idleJobs {
		// Check if this job has an assigned agent
		assignedAgent, err := s.agentsService.GetAgentByJobID(job.ID)
		if err != nil {
			if !strings.Contains(fmt.Sprintf("%v", err), "not found") {
				log.Printf("❌ Failed to check agent assignment for job %s: %v", job.ID, err)
				continue
			}
		} else {
			if err := s.agentsService.UnassignJobFromAgent(assignedAgent.ID); err != nil {
				log.Printf("❌ Failed to unassign agent %s from idle job %s: %v", assignedAgent.ID, job.ID, err)
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
		if err := s.jobsService.DeleteJob(job.ID); err != nil {
			log.Printf("❌ Failed to delete idle job %s: %v", job.ID, err)
			continue
		}

		log.Printf("🗑️ Deleted idle job %s (thread: %s)", job.ID, job.SlackThreadTS)

		// Send completion message to Slack thread
		completionMessage := "This job is now complete"
		_, _, err = s.slackClient.PostMessage(job.SlackChannelID,
			slack.MsgOptionText(utils.ConvertMarkdownToSlack(completionMessage), false),
			slack.MsgOptionTS(job.SlackThreadTS),
		)
		if err != nil {
			log.Printf("⚠️ Failed to send completion message to Slack thread %s: %v", job.SlackThreadTS, err)
		} else {
			log.Printf("📤 Sent completion message to Slack thread %s", job.SlackThreadTS)
		}
	}

	log.Printf("📋 Completed successfully - cleaned up %d idle jobs", len(idleJobs))
}

