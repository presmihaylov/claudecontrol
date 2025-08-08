package usecases

import (
	"context"
	"fmt"
	"sort"

	"ccbackend/models"
	"ccbackend/services"
)

type agentWithLoad struct {
	agent *models.ActiveAgent
	load  int
}

func sortAgentsByLoad(
	ctx context.Context,
	agents []*models.ActiveAgent,
	slackIntegrationID string,
	agentsService services.AgentsService,
) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, slackIntegrationID)
		if err != nil {
			return nil, fmt.Errorf("failed to get job assignments for agent %s: %w", agent.ID, err)
		}

		jobCount := len(jobIDs)

		agentsWithLoad = append(agentsWithLoad, agentWithLoad{agent: agent, load: jobCount})
	}

	// Sort by load (ascending - least loaded first)
	sort.Slice(agentsWithLoad, func(i, j int) bool {
		return agentsWithLoad[i].load < agentsWithLoad[j].load
	})

	return agentsWithLoad, nil
}