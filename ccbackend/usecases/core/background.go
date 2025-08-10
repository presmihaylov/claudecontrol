package core

import (
	"context"
	"fmt"
	"log"

	"ccbackend/core"
	"ccbackend/models"
)

// DefaultIdleJobTimeoutMinutes defines how long a job can be idle before being assigned to an available agent
const DefaultIdleJobTimeoutMinutes = 5

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

// GetActiveOrganizations returns organizations that have available agents
func (s *CoreUseCase) GetActiveOrganizations(ctx context.Context) ([]*models.Organization, error) {
	log.Printf("📋 Starting to get active organizations with available agents")
	
	organizations, err := s.organizationsService.GetAllOrganizations(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get organizations: %w", err)
	}

	connectedClientIDs := s.wsClient.GetClientIDs()
	var activeOrganizations []*models.Organization

	for _, organization := range organizations {
		availableAgents, err := s.agentsService.GetConnectedAvailableAgents(ctx, organization.ID, connectedClientIDs)
		if err != nil {
			return nil, fmt.Errorf("failed to get available agents for organization %s: %w", organization.ID, err)
		}
		
		if len(availableAgents) > 0 {
			activeOrganizations = append(activeOrganizations, organization)
		}
	}

	log.Printf("📋 Completed successfully - found %d active organizations", len(activeOrganizations))
	return activeOrganizations, nil
}

// AssignJobs assigns idle jobs to available agents for the given organization
func (s *CoreUseCase) AssignJobs(ctx context.Context, organization *models.Organization) error {
	log.Printf("📋 Starting to assign jobs for organization: %s", organization.ID)
	
	connectedClientIDs := s.wsClient.GetClientIDs()
	availableAgents, err := s.agentsService.GetConnectedAvailableAgents(ctx, organization.ID, connectedClientIDs)
	if err != nil {
		return fmt.Errorf("failed to get available agents for organization %s: %w", organization.ID, err)
	}

	if len(availableAgents) == 0 {
		log.Printf("📋 No available agents for organization %s", organization.ID)
		return nil
	}

	idleJobs, err := s.jobsService.GetIdleJobs(ctx, DefaultIdleJobTimeoutMinutes, organization.ID)
	if err != nil {
		return fmt.Errorf("failed to get idle jobs for organization %s: %w", organization.ID, err)
	}

	if len(idleJobs) == 0 {
		log.Printf("📋 No idle jobs found for organization %s", organization.ID)
		return nil
	}

	assignmentCount := 0
	for i, job := range idleJobs {
		if i >= len(availableAgents) {
			break // More jobs than agents, assign what we can
		}
		
		agent := availableAgents[i]
		if err := s.agentsService.AssignAgentToJob(ctx, agent.ID, job.ID, organization.ID); err != nil {
			return fmt.Errorf("failed to assign agent %s to job %s: %w", agent.ID, job.ID, err)
		}
		assignmentCount++
		log.Printf("📤 Assigned agent %s to job %s", agent.ID, job.ID)
	}

	log.Printf("📋 Completed successfully - assigned %d jobs for organization %s", assignmentCount, organization.ID)
	return nil
}
