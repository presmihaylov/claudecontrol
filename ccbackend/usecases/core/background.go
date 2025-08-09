package core

import (
	"context"
	"fmt"
	"log"

	"ccbackend/core"
	"ccbackend/models"
)

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
