package agents

import (
	"context"
	"fmt"
	"log"
	"slices"
	"sort"

	"ccbackend/clients"
	"ccbackend/models"
	"ccbackend/services"
)

// AgentsUseCase handles agent-job assignment logic
type AgentsUseCase struct {
	wsClient      clients.SocketIOClient
	agentsService services.AgentsService
}

// NewAgentsUseCase creates a new instance of AgentsUseCase
func NewAgentsUseCase(
	wsClient clients.SocketIOClient,
	agentsService services.AgentsService,
) *AgentsUseCase {
	return &AgentsUseCase{
		wsClient:      wsClient,
		agentsService: agentsService,
	}
}

// GetOrAssignAgentForJob gets an existing agent assignment or assigns a new agent to a job
func (s *AgentsUseCase) GetOrAssignAgentForJob(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	// Check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, job.ID, organizationID)
	if err != nil {
		// Some error occurred
		log.Printf("❌ Failed to check for existing agent assignment: %v", err)
		return "", fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if !maybeExistingAgent.IsPresent() {
		// Job not assigned to any agent yet - need to assign to an available agent
		return s.AssignJobToAvailableAgent(ctx, job, threadTS, organizationID)
	}

	existingAgent := maybeExistingAgent.MustGet()

	// Job is already assigned to an agent - verify it still has an active connection
	connectedClientIDs := s.wsClient.GetClientIDs()
	if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
		log.Printf(
			"🔄 Job %s already assigned to agent %s with active connection, routing message to existing agent",
			job.ID,
			existingAgent.ID,
		)
		return existingAgent.WSConnectionID, nil
	}

	// Existing agent doesn't have active connection - return error to signal no available agents
	log.Printf("⚠️ Job %s assigned to agent %s but no active WebSocket connection found", job.ID, existingAgent.ID)
	return "", fmt.Errorf("no active agents available for job assignment")
}

// AssignJobToAvailableAgent attempts to assign a job to the least loaded available agent
// Returns the WebSocket client ID if successful, empty string if no agents available, or error on failure
func (s *AgentsUseCase) AssignJobToAvailableAgent(
	ctx context.Context,
	job *models.Job,
	threadTS, organizationID string,
) (string, error) {
	log.Printf("📝 Job %s not yet assigned, looking for any active agent", job.ID)

	clientID, assigned, err := s.TryAssignJobToAgent(ctx, job.ID, organizationID)
	if err != nil {
		return "", err
	}

	if !assigned {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", fmt.Errorf("no agents with active WebSocket connections available for job assignment")
	}

	log.Printf("✅ Assigned job %s to agent for slack thread %s (agent can handle multiple jobs)", job.ID, threadTS)
	return clientID, nil
}

// TryAssignJobToAgent is a reusable function that attempts to assign a job to the least loaded available agent
// Returns (clientID, wasAssigned, error) where:
// - clientID: WebSocket connection ID of assigned agent (empty if not assigned)
// - wasAssigned: true if job was successfully assigned to an agent, false if no agents available
// - error: any error that occurred during the assignment process
func (s *AgentsUseCase) TryAssignJobToAgent(
	ctx context.Context,
	jobID string,
	organizationID string,
) (string, bool, error) {
	// First check if this job is already assigned to an agent
	maybeExistingAgent, err := s.agentsService.GetAgentByJobID(ctx, jobID, organizationID)
	if err != nil {
		return "", false, fmt.Errorf("failed to check for existing agent assignment: %w", err)
	}

	if maybeExistingAgent.IsPresent() {
		existingAgent := maybeExistingAgent.MustGet()
		// Job is already assigned - check if agent still has active connection
		connectedClientIDs := s.wsClient.GetClientIDs()
		if s.agentsService.CheckAgentHasActiveConnection(existingAgent, connectedClientIDs) {
			log.Printf("🔄 Job %s already assigned to agent %s with active connection", jobID, existingAgent.ID)
			return existingAgent.WSConnectionID, true, nil
		}
		// Agent no longer has active connection - job remains assigned but can't process
		log.Printf("⚠️ Job %s assigned to agent %s but no active connection", jobID, existingAgent.ID)
		return "", false, nil
	}

	// Job not assigned - proceed with assignment
	// Get active WebSocket connections first
	connectedClientIDs := s.wsClient.GetClientIDs()
	log.Printf("🔍 Found %d connected WebSocket clients", len(connectedClientIDs))

	// Get only agents with active connections using centralized service method
	connectedAgents, err := s.agentsService.GetConnectedActiveAgents(ctx, organizationID, connectedClientIDs)
	if err != nil {
		log.Printf("❌ Failed to get connected active agents: %v", err)
		return "", false, fmt.Errorf("failed to get connected active agents: %w", err)
	}

	if len(connectedAgents) == 0 {
		log.Printf("⚠️ No agents have active WebSocket connections")
		return "", false, nil
	}

	// Sort agents by load (number of assigned jobs) to select the least loaded agent
	sortedAgents, err := s.sortAgentsByLoad(ctx, connectedAgents, organizationID)
	if err != nil {
		log.Printf("❌ Failed to sort agents by load: %v", err)
		return "", false, fmt.Errorf("failed to sort agents by load: %w", err)
	}

	selectedAgent := sortedAgents[0].agent
	log.Printf("🎯 Selected agent %s with %d active jobs (least loaded)", selectedAgent.ID, sortedAgents[0].load)

	// Assign the job to the selected agent (agents can now handle multiple jobs simultaneously)
	if err := s.agentsService.AssignAgentToJob(ctx, selectedAgent.ID, jobID, organizationID); err != nil {
		log.Printf("❌ Failed to assign job %s to agent %s: %v", jobID, selectedAgent.ID, err)
		return "", false, fmt.Errorf("failed to assign job to agent: %w", err)
	}

	log.Printf("✅ Assigned job %s to agent %s", jobID, selectedAgent.ID)
	return selectedAgent.WSConnectionID, true, nil
}

// ValidateJobBelongsToAgent checks if a job is assigned to the specified agent
func (s *AgentsUseCase) ValidateJobBelongsToAgent(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	agentJobs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agentID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to get jobs for agent: %w", err)
	}
	if slices.Contains(agentJobs, jobID) {
		log.Printf("✅ Agent %s is assigned to job %s", agentID, jobID)
		return nil
	}

	log.Printf("❌ Agent %s is not assigned to job %s", agentID, jobID)
	return fmt.Errorf("agent %s is not assigned to job %s", agentID, jobID)
}

type agentWithLoad struct {
	agent *models.ActiveAgent
	load  int
}

// sortAgentsByLoad sorts agents by their current job load (ascending - least loaded first)
func (s *AgentsUseCase) sortAgentsByLoad(
	ctx context.Context,
	agents []*models.ActiveAgent,
	organizationID string,
) ([]agentWithLoad, error) {
	agentsWithLoad := make([]agentWithLoad, 0, len(agents))

	for _, agent := range agents {
		// Get job IDs assigned to this agent
		jobIDs, err := s.agentsService.GetActiveAgentJobAssignments(ctx, agent.ID, organizationID)
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
