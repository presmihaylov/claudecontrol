package usecases

import (
	"fmt"
	"log"
	"strings"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"

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
		slack.MsgOptionText(payload.Message, false),
		slack.MsgOptionTS(job.SlackThreadTS),
	)
	if err != nil {
		return fmt.Errorf("❌ Failed to send assistant message to Slack: %v", err)
	}

	// Update job timestamp to track activity
	if err := s.jobsService.UpdateJobTimestamp(job.ID); err != nil {
		log.Printf("⚠️ Failed to update job timestamp for job %s: %v", job.ID, err)
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
			slack.MsgOptionText("There are no available agents to handle this job", false),
			slack.MsgOptionTS(threadTS),
		)
		if err != nil {
			log.Printf("❌ Failed to send 'no agents available' message to Slack: %v", err)
			return fmt.Errorf("failed to send 'no agents available' message to Slack: %w", err)
		}
		
		log.Printf("📤 Sent 'no agents available' message to Slack thread %s in channel %s", threadTS, event.Channel)
		return nil
	}

	// Send appropriate message to the assigned agent
	if event.ThreadTs == "" {
		startConversationMessage := models.UnknownMessage{
			Type:    models.MessageTypeStartConversation,
			Payload: models.StartConversationPayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(clientID, startConversationMessage); err != nil {
			return fmt.Errorf("failed to send start conversation message to client %s: %v", clientID, err)
		}
		log.Printf("🚀 Sent start conversation message to client %s", clientID)
	} else {
		userMessage := models.UnknownMessage{
			Type:    models.MessageTypeUserMessage,
			Payload: models.UserMessagePayload{Message: event.Text},
		}

		if err := s.wsClient.SendMessage(clientID, userMessage); err != nil {
			return fmt.Errorf("failed to send user message to client %s: %v", clientID, err)
		}
		log.Printf("💬 Sent user message to client %s", clientID)
	}

	log.Printf("📋 Completed successfully - processed Slack message event")
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
	idleJobs, err := s.jobsService.GetIdleJobs(2)
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
	}

	log.Printf("📋 Completed successfully - cleaned up %d idle jobs", len(idleJobs))
}