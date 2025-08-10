package core

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ccbackend/clients"
	"ccbackend/core"
	"ccbackend/models"
	"ccbackend/services"
	"ccbackend/usecases/agents"
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
}

// NewCoreUseCase creates a new instance of CoreUseCase
func NewCoreUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
	jobsService services.JobsService,
	slackIntegrationsService services.SlackIntegrationsService,
	organizationsService services.OrganizationsService,
	agentsUseCase *agents.AgentsUseCase,
) *CoreUseCase {
	return &CoreUseCase{
		wsClient:                 wsClient,
		agentsService:            agentsService,
		jobsService:              jobsService,
		slackIntegrationsService: slackIntegrationsService,
		organizationsService:     organizationsService,
		agentsUseCase:            agentsUseCase,
	}
}


// Agent Management Functions

// RegisterAgent registers a new agent connection in the system
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

// DeregisterAgent removes an agent from the system and cleans up its jobs
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

	// Process each job: route cleanup based on job type
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

		// TODO: Route cleanup to appropriate usecase based on job type
		log.Printf("⚠️ Job cleanup during agent deregistration not implemented for job type %s (job %s)", job.JobType, jobID)

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

// ProcessPing updates the last active timestamp for an agent
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

	log.Printf("📋 Completed successfully - updated ping timestamp for client %s", client.ID)
	return nil
}

const DefaultInactiveAgentTimeoutMinutes = 10

// CleanupInactiveAgents removes agents that have been inactive for more than the timeout period
func (s *CoreUseCase) CleanupInactiveAgents(ctx context.Context) error {
	log.Printf("📋 Starting to cleanup inactive agents (>%d minutes)", DefaultInactiveAgentTimeoutMinutes)

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
	inactiveThresholdMinutes := DefaultInactiveAgentTimeoutMinutes

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

// Authentication Functions

// ValidateAPIKey validates an API key and returns the organization ID if valid
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

// Background Processing Functions

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
		organizationID := organization.ID

		// Get connected agents for this organization using centralized service method
		connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
		if err != nil {
			return fmt.Errorf("failed to get connected agents for organization %s: %w", organizationID, err)
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
				return fmt.Errorf("failed to send CheckIdleJobs message to agent %s: %w", agent.ID, err)
			}
			log.Printf("📤 Sent CheckIdleJobs message to agent %s", agent.ID)
			totalAgentCount++
		}
	}

	log.Printf("📋 Completed successfully - broadcasted CheckIdleJobs to %d agents", totalAgentCount)
	return nil
}
