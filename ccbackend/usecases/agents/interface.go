package agents

import (
	"context"

	"ccbackend/models"
)

// AgentsUseCaseInterface defines the interface for agent-job assignment operations
type AgentsUseCaseInterface interface {
	// GetOrAssignAgentForJob gets an existing agent assignment or assigns a new agent to a job
	GetOrAssignAgentForJob(
		ctx context.Context,
		job *models.Job,
		threadTS, organizationID string,
	) (string, error)

	// TryAssignJobToAgent attempts to assign a job to the least loaded available agent
	// Returns (clientID, wasAssigned, error) where:
	// - clientID: WebSocket connection ID of assigned agent (empty if not assigned)
	// - wasAssigned: true if job was successfully assigned to an agent, false if no agents available
	// - error: any error that occurred during the assignment process
	TryAssignJobToAgent(
		ctx context.Context,
		jobID string,
		organizationID string,
	) (string, bool, error)

	// ValidateJobBelongsToAgent checks if a job is assigned to the specified agent
	ValidateJobBelongsToAgent(
		ctx context.Context,
		agentID, jobID string,
		organizationID string,
	) error
}
