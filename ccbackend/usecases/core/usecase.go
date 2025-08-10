package core

import (
	"context"
	"fmt"
	"log"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/agents"
	"ccbackend/usecases/slack"
)

// CoreUseCase orchestrates all core business operations
type CoreUseCase struct {
	wsClient                 clients.SocketIOClient
	agentsService            services.AgentsService
	jobsService              services.JobsService
	slackIntegrationsService services.SlackIntegrationsService
	organizationsService     services.OrganizationsService

	// Use case dependencies
	agentsUseCase *agents.AgentsUseCase
	slackUseCase  *slack.SlackUseCase
}

// NewCoreUseCase creates a new instance of CoreUseCase
func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	agentsUseCase *agents.AgentsUseCase,
	slackUseCase *slack.SlackUseCase,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		agentsUseCase:            agentsUseCase,
		slackUseCase:             slackUseCase,
	}
}

// Proxy methods to SlackUseCase

// ProcessSlackMessageEvent proxies to SlackUseCase
func (s *CoreUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	organizationID string,
) error {
	return s.slackUseCase.ProcessSlackMessageEvent(ctx, event, slackIntegrationID, organizationID)
}

// ProcessReactionAdded proxies to SlackUseCase
func (s *CoreUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	organizationID string,
) error {
	return s.slackUseCase.ProcessReactionAdded(
		ctx,
		reactionName,
		userID,
		channelID,
		messageTS,
		slackIntegrationID,
		organizationID,
	)
}

// ProcessProcessingMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessProcessingMessage(ctx, clientID, payload, organizationID)
}

// ProcessAssistantMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessAssistantMessage(ctx, clientID, payload, organizationID)
}

// ProcessSystemMessage proxies to SlackUseCase
func (s *CoreUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessSystemMessage(ctx, clientID, payload, organizationID)
}

// ProcessJobComplete proxies to SlackUseCase
func (s *CoreUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	organizationID string,
) error {
	return s.slackUseCase.ProcessJobComplete(ctx, clientID, payload, organizationID)
}

// ProcessQueuedJobs proxies to SlackUseCase
func (s *CoreUseCase) ProcessQueuedJobs(ctx context.Context) error {
	return s.slackUseCase.ProcessQueuedJobs(ctx)
}

// ProcessJobsInBackground processes pending jobs by assigning them to available agents
func (s *CoreUseCase) ProcessJobsInBackground(ctx context.Context) error {
	log.Printf("📋 Starting to process jobs in background")

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
	connectedClientIDs := s.wsClient.GetClientIDs()

	for _, integration := range integrations {
		organizationID := integration.OrganizationID

		// Get pending jobs for this organization
		jobs, err := s.jobsService.GetJobsByOrganizationID(ctx, organizationID)
		if err != nil {
			return fmt.Errorf("failed to get jobs for organization %s: %w", organizationID, err)
		}

		if len(jobs) == 0 {
			continue
		}

		// Get available agents for this organization
		availableAgents, err := s.agentsService.GetConnectedAvailableAgents(ctx, organizationID, connectedClientIDs)
		if err != nil {
			return fmt.Errorf("failed to get active agents for organization %s: %w", organizationID, err)
		}

		if len(availableAgents) == 0 {
			log.Printf("⚠️ No available agents found for organization %s", organizationID)
			continue
		}

		// Process all jobs (no status check since status field doesn't exist)
		for i, job := range jobs {
			// Round-robin assign to available agents
			agent := availableAgents[i%len(availableAgents)]

			// Assign job to agent
			if err := s.jobsService.AssignJobToAgent(ctx, job.ID, agent.ID, organizationID); err != nil {
				return fmt.Errorf("failed to assign job %s to agent %s: %w", job.ID, agent.ID, err)
			}

			// Send job message to agent
			if err := s.wsClient.SendMessage(agent.WSConnectionID, job); err != nil {
				return fmt.Errorf("failed to send job message to agent %s: %w", agent.ID, err)
			}

			log.Printf("✅ Assigned job %s to agent %s", job.ID, agent.ID)
			totalProcessedJobs++
		}
	}

	log.Printf("📋 Completed successfully - processed %d jobs", totalProcessedJobs)
	return nil
}
