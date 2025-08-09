package core

import (
	"context"
	"fmt"
	"log"
	"strings"

	"ccbackend/clients"
	"ccbackend/models"
)

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

	// Collect errors during job cleanup instead of returning early
	var cleanupErrors []string
	abandonmentMessage := ":x: The assigned agent was disconnected, abandoning job"

	// Process each job: route cleanup based on job type
	for _, jobID := range jobs {
		// Get job directly using organization_id (optimization)
		maybeJob, err := s.jobsService.GetJobByID(ctx, jobID, client.OrganizationID)
		if err != nil {
			log.Printf("❌ Failed to get job %s for cleanup: %v", jobID, err)
			cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to get job %s: %v", jobID, err))
			continue
		}
		if !maybeJob.IsPresent() {
			log.Printf("⚠️ Job %s not found for cleanup (may have been deleted already)", jobID)
			continue // Skip this job, it may have been deleted already
		}

		job := maybeJob.MustGet()

		// Route cleanup based on job type
		switch job.JobType {
		case models.JobTypeSlack:
			if err := s.slackUseCase.CleanupFailedSlackJob(ctx, job, agent.ID, abandonmentMessage); err != nil {
				log.Printf("❌ Failed to cleanup Slack job %s: %v", jobID, err)
				cleanupErrors = append(cleanupErrors, fmt.Sprintf("failed to cleanup Slack job %s: %v", jobID, err))
				continue
			}
		// Future: case models.JobTypeEmail:
		//     err = s.emailUseCase.CleanupFailedEmailJob(ctx, job, agent.ID, abandonmentMessage)
		default:
			log.Printf("⚠️ Unknown job type %s for job %s, skipping cleanup", job.JobType, jobID)
			continue
		}

		log.Printf("✅ Cleaned up abandoned job %s", jobID)
	}

	// Delete the agent record (use organization ID since agents are organization-scoped)
	err = s.agentsService.DeleteActiveAgentByWsConnectionID(ctx, client.ID, client.OrganizationID)
	if err != nil {
		return fmt.Errorf("failed to deregister agent for client %s: %w", client.ID, err)
	}

	// Return error if there were any job cleanup failures (after agent is deleted)
	if len(cleanupErrors) > 0 {
		return fmt.Errorf(
			"agent deregistered but encountered %d job cleanup errors: %s",
			len(cleanupErrors),
			strings.Join(cleanupErrors, "; "),
		)
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
