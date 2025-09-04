package core

import (
	"context"
	"fmt"
	"log"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/salesnotif"
	"ccbackend/services"
	"ccbackend/usecases/discord"
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
	slackUseCase   *slack.SlackUseCase
	discordUseCase *discord.DiscordUseCase
}

// NewCoreUseCase creates a new instance of CoreUseCase
func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	slackUseCase *slack.SlackUseCase,
	discordUseCase *discord.DiscordUseCase,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		slackUseCase:             slackUseCase,
		discordUseCase:           discordUseCase,
	}
}

// Proxy methods to SlackUseCase

// ProcessSlackMessageEvent proxies to SlackUseCase
func (s *CoreUseCase) ProcessSlackMessageEvent(
	ctx context.Context,
	event models.SlackMessageEvent,
	slackIntegrationID string,
	orgID models.OrgID,
) error {
	return s.slackUseCase.ProcessSlackMessageEvent(ctx, event, slackIntegrationID, orgID)
}

// ProcessReactionAdded proxies to SlackUseCase
func (s *CoreUseCase) ProcessReactionAdded(
	ctx context.Context,
	reactionName, userID, channelID, messageTS, slackIntegrationID string,
	orgID models.OrgID,
) error {
	return s.slackUseCase.ProcessReactionAdded(
		ctx,
		reactionName,
		userID,
		channelID,
		messageTS,
		slackIntegrationID,
		orgID,
	)
}

// ProcessProcessingMessage routes to appropriate usecase based on job type
func (s *CoreUseCase) ProcessProcessingMessage(
	ctx context.Context,
	clientID string,
	payload models.ProcessingMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to route processing message from client %s", clientID)
	jobID := payload.JobID

	// Get job to determine the platform
	maybeJob, err := s.jobsService.GetJobByID(ctx, orgID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed, skipping processing message", jobID)
		return nil
	}

	job := maybeJob.MustGet()
	switch job.JobType {
	case models.JobTypeSlack:
		log.Printf("🔀 Routing processing message to Slack usecase for job %s", jobID)
		return s.slackUseCase.ProcessProcessingMessage(ctx, clientID, payload, orgID)
	case models.JobTypeDiscord:
		log.Printf("🔀 Routing processing message to Discord usecase for job %s", jobID)
		return s.discordUseCase.ProcessProcessingMessage(ctx, clientID, payload, orgID)
	default:
		return fmt.Errorf("unsupported job type: %s", job.JobType)
	}
}

// ProcessAssistantMessage routes to appropriate usecase based on job type
func (s *CoreUseCase) ProcessAssistantMessage(
	ctx context.Context,
	clientID string,
	payload models.AssistantMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to route assistant message from client %s", clientID)

	// Get the job to determine the type
	jobID := payload.JobID
	if jobID == "" {
		return fmt.Errorf("JobID is empty in AssistantMessage payload")
	}

	// Get job to determine the platform
	maybeJob, err := s.jobsService.GetJobByID(ctx, orgID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed, skipping assistant message", jobID)
		return nil
	}

	job := maybeJob.MustGet()

	// Route based on job type
	switch job.JobType {
	case models.JobTypeSlack:
		log.Printf("🔀 Routing assistant message to Slack usecase for job %s", jobID)
		return s.slackUseCase.ProcessAssistantMessage(ctx, clientID, payload, orgID)
	case models.JobTypeDiscord:
		log.Printf("🔀 Routing assistant message to Discord usecase for job %s", jobID)
		return s.discordUseCase.ProcessAssistantMessage(ctx, clientID, payload, orgID)
	default:
		return fmt.Errorf("unsupported job type: %s", job.JobType)
	}
}

// ProcessSystemMessage routes to appropriate usecase based on job type
func (s *CoreUseCase) ProcessSystemMessage(
	ctx context.Context,
	clientID string,
	payload models.SystemMessagePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to route system message from client %s", clientID)

	// Get the job ID from the payload to determine the type
	jobID := payload.JobID
	if jobID == "" {
		return fmt.Errorf("JobID is empty in SystemMessage payload")
	}

	// Get job to determine the platform
	maybeJob, err := s.jobsService.GetJobByID(ctx, orgID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed, skipping system message", jobID)
		return nil
	}

	job := maybeJob.MustGet()

	// Send sales notification for system message
	salesnotif.New(fmt.Sprintf("Job %s received ccagent system event: %s", jobID, payload.Message))

	// Route based on job type
	switch job.JobType {
	case models.JobTypeSlack:
		log.Printf("🔀 Routing system message to Slack usecase for job %s", jobID)
		return s.slackUseCase.ProcessSystemMessage(ctx, clientID, payload, orgID)
	case models.JobTypeDiscord:
		log.Printf("🔀 Routing system message to Discord usecase for job %s", jobID)
		return s.discordUseCase.ProcessSystemMessage(ctx, clientID, payload, orgID)
	default:
		return fmt.Errorf("unsupported job type: %s", job.JobType)
	}
}

// ProcessJobComplete routes to appropriate usecase based on job type
func (s *CoreUseCase) ProcessJobComplete(
	ctx context.Context,
	clientID string,
	payload models.JobCompletePayload,
	orgID models.OrgID,
) error {
	log.Printf("📋 Starting to route job complete from client %s", clientID)

	jobID := payload.JobID
	maybeJob, err := s.jobsService.GetJobByID(ctx, orgID, jobID)
	if err != nil {
		return fmt.Errorf("failed to get job: %w", err)
	}
	if !maybeJob.IsPresent() {
		log.Printf("⚠️ Job %s not found - already completed, skipping job complete message", jobID)
		return nil
	}

	job := maybeJob.MustGet()

	// Route based on job type
	switch job.JobType {
	case models.JobTypeSlack:
		log.Printf("🔀 Routing job complete to Slack usecase for job %s", jobID)
		return s.slackUseCase.ProcessJobComplete(ctx, clientID, payload, orgID)
	case models.JobTypeDiscord:
		log.Printf("🔀 Routing job complete to Discord usecase for job %s", jobID)
		return s.discordUseCase.ProcessJobComplete(ctx, clientID, payload, orgID)
	default:
		return fmt.Errorf("unsupported job type: %s", job.JobType)
	}
}

// ProcessQueuedJobs processes queued jobs for all platforms
func (s *CoreUseCase) ProcessQueuedJobs(ctx context.Context) error {
	log.Printf("📋 Starting to process queued jobs for all platforms")

	// Process Slack queued jobs
	if err := s.slackUseCase.ProcessQueuedJobs(ctx); err != nil {
		return fmt.Errorf("failed to process Slack queued jobs: %w", err)
	}

	// Process Discord queued jobs
	if err := s.discordUseCase.ProcessQueuedJobs(ctx); err != nil {
		return fmt.Errorf("failed to process Discord queued jobs: %w", err)
	}

	log.Printf("📋 Completed successfully - processed queued jobs for all platforms")
	return nil
}

// RegisterAgent registers a new agent connection in the system
func (s *CoreUseCase) RegisterAgent(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to register agent for client %s", client.ID)

	// Pass the agent ID to UpsertActiveAgent - use organization ID since agents are organization-scoped
	_, err := s.agentsService.UpsertActiveAgent(ctx, client.OrgID, client.ID, client.AgentID)
	if err != nil {
		return fmt.Errorf("failed to register agent for client %s: %w", client.ID, err)
	}

	log.Printf(
		"📋 Completed successfully - registered agent for client %s with organization %s",
		client.ID,
		client.OrgID,
	)
	return nil
}

// DeregisterAgent removes an agent from the system and cleans up its jobs
func (s *CoreUseCase) DeregisterAgent(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to deregister agent for client %s", client.ID)

	// Find the agent directly using organization ID since agents are organization-scoped
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, client.OrgID, client.ID)
	if err != nil {
		return fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", client.ID)
		return fmt.Errorf("no agent found for client: %s", client.ID)
	}

	agent := maybeAgent.MustGet()

	// Get active jobs for agent cleanup
	jobs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, client.OrgID, agent.ID)
	if err != nil {
		log.Printf("❌ Failed to get jobs for cleanup: %v", err)
		return fmt.Errorf("failed to get jobs for cleanup: %w", err)
	}

	// Clean up all job assignments - handle each job consistently
	log.Printf("🧹 Agent %s has %d assigned job(s), cleaning up all assignments", agent.ID, len(jobs))

	// Process each job: route cleanup based on job type
	for _, jobID := range jobs {
		// Get job directly using organization_id (optimization)
		maybeJob, err := s.jobsService.GetJobByID(ctx, client.OrgID, jobID)
		if err != nil {
			log.Printf("❌ Failed to get job %s for cleanup: %v", jobID, err)
			return fmt.Errorf("failed to get job for cleanup: %w", err)
		}
		if !maybeJob.IsPresent() {
			log.Printf("❌ Job %s not found for cleanup", jobID)
			continue // Skip this job, it may have been deleted already
		}

		job := maybeJob.MustGet()

		// Route cleanup based on job type
		switch job.JobType {
		case models.JobTypeSlack:
			abandonmentMessage := ":x: The assigned agent was disconnected, abandoning job"
			if err := s.slackUseCase.CleanupFailedSlackJob(ctx, job, agent.ID, abandonmentMessage); err != nil {
				return fmt.Errorf("failed to cleanup abandoned Slack job %s: %w", jobID, err)
			}
		case models.JobTypeDiscord:
			abandonmentMessage := "❌ The assigned agent was disconnected, abandoning job"
			if err := s.discordUseCase.CleanupFailedDiscordJob(ctx, job, agent.ID, abandonmentMessage); err != nil {
				return fmt.Errorf("failed to cleanup abandoned Discord job %s: %w", jobID, err)
			}
		default:
			log.Printf("⚠️ Unknown job type %s for job %s, skipping cleanup", job.JobType, jobID)
			continue
		}

		log.Printf("✅ Cleaned up abandoned job %s", jobID)
	}

	// Delete the agent record (use organization ID since agents are organization-scoped)
	err = s.agentsService.DeleteActiveAgentByWsConnectionID(ctx, client.OrgID, client.ID)
	if err != nil {
		return fmt.Errorf("failed to deregister agent for client %s: %w", client.ID, err)
	}

	log.Printf("📋 Completed successfully - deregistered agent for client %s", client.ID)
	return nil
}

// ProcessPing updates the last active timestamp for an agent
func (s *CoreUseCase) ProcessPing(ctx context.Context, client *clients.Client) error {
	log.Printf("📋 Starting to process ping from client %s", client.ID)

	// Check if agent exists for this client (agents are organization-scoped)
	maybeAgent, err := s.agentsService.GetAgentByWSConnectionID(ctx, client.OrgID, client.ID)
	if err != nil {
		return fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	if !maybeAgent.IsPresent() {
		log.Printf("❌ No agent found for client %s", client.ID)
		return fmt.Errorf("no agent found for client: %s", client.ID)
	}

	// Update the agent's last_active_at timestamp (use organization ID since agents are organization-scoped)
	if err := s.agentsService.UpdateAgentLastActiveAt(ctx, client.OrgID, client.ID); err != nil {
		log.Printf("❌ Failed to update agent last_active_at for client %s: %v", client.ID, err)
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated ping timestamp for client %s", client.ID)
	return nil
}

const DefaultInactiveAgentTimeoutMinutes = 10

// CleanupInactiveAgents removes agents that have been inactive for more than the timeout period
func (s *CoreUseCase) CleanupInactiveAgents(ctx context.Context) error {
	log.Printf("📋 Starting to cleanup inactive agents")
	organizations, err := s.organizationsService.GetAllOrganizations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get organizations: %w", err)
	}
	if len(organizations) == 0 {
		log.Printf("📋 No organizations found")
		return nil
	}

	totalInactiveAgents := 0
	inactiveThresholdMinutes := DefaultInactiveAgentTimeoutMinutes
	for _, organization := range organizations {
		orgID := organization.ID

		// Get inactive agents for this organization (agents are organization-scoped)
		inactiveAgents, err := s.agentsService.GetInactiveAgents(
			ctx,
			models.OrgID(orgID),
			inactiveThresholdMinutes,
		)
		if err != nil {
			return fmt.Errorf("failed to get inactive agents for organization %s: %w", orgID, err)
		}

		if len(inactiveAgents) == 0 {
			continue
		}

		log.Printf("🔍 Found %d inactive agents for organization %s", len(inactiveAgents), orgID)

		// Delete each inactive agent
		for _, agent := range inactiveAgents {
			log.Printf(
				"🧹 Found inactive agent %s (last active: %s) - cleaning up",
				agent.ID,
				agent.LastActiveAt.Format("2006-01-02 15:04:05"),
			)

			// Delete the inactive agent - CASCADE DELETE will automatically clean up job assignments
			if err := s.agentsService.DeleteActiveAgent(ctx, models.OrgID(orgID), agent.ID); err != nil {
				return fmt.Errorf("failed to delete inactive agent %s: %w", agent.ID, err)
			}

			log.Printf("✅ Deleted inactive agent %s (CASCADE DELETE cleaned up job assignments)", agent.ID)
			totalInactiveAgents++
		}
	}

	log.Printf("📋 Completed successfully - cleaned up %d inactive agents", totalInactiveAgents)
	return nil
}

// BroadcastCheckIdleJobs sends a CheckIdleJobs message to all connected agents
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
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	for _, organization := range organizations {
		orgID := organization.ID

		// Get connected agents for this organization using centralized service method
		connectedAgents, err := s.agentsService.GetConnectedActiveAgents(
			ctx,
			models.OrgID(orgID),
			connectedClientIDs,
		)
		if err != nil {
			return fmt.Errorf("failed to get connected agents for organization %s: %w", orgID, err)
		}

		if len(connectedAgents) == 0 {
			continue
		}

		log.Printf(
			"📡 Broadcasting CheckIdleJobs to %d connected agents for organization %s",
			len(connectedAgents),
			orgID,
		)
		checkIdleJobsMessage := models.BaseMessage{
			ID:      core.NewID("msg"),
			Type:    models.MessageTypeCheckIdleJobs,
			Payload: models.CheckIdleJobsPayload{},
		}

		for _, agent := range connectedAgents {
			if err := s.wsClient.SendMessage(agent.WSConnectionID, checkIdleJobsMessage); err != nil {
				return fmt.Errorf("failed to send CheckIdleJobs message to agent %s: %w", agent.ID, err)
			}
			log.Printf("📤 Sent CheckIdleJobs message to agent %s", agent.ID)
			totalAgentCount++
		}
	}

	log.Printf("📋 Completed successfully - broadcasted CheckIdleJobs to %d agents", totalAgentCount)
	return nil
}
