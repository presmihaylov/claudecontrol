package agents

import (
	"context"
	"fmt"
	"log"

	"github.com/samber/mo"

	"ccbackend/core"
	"ccbackend/db"
	"ccbackend/models"
)

type AgentsService struct {
	agentsRepo *db.PostgresAgentsRepository
}

func NewAgentsService(repo *db.PostgresAgentsRepository) *AgentsService {
	return &AgentsService{agentsRepo: repo}
}

func (s *AgentsService) UpsertActiveAgent(
	ctx context.Context,
	wsConnectionID, organizationID string,
	agentID string,
) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to upsert active agent for wsConnectionID: %s, agentID: %s", wsConnectionID, agentID)
	if !core.IsValidULID(wsConnectionID) {
		return nil, fmt.Errorf("ws_connection_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}
	if !core.IsValidULID(agentID) {
		return nil, fmt.Errorf("ccagent_id must be a valid ULID")
	}

	agent := &models.ActiveAgent{
		ID:             core.NewID("ag"),
		WSConnectionID: wsConnectionID,
		OrganizationID: organizationID,
		CCAgentID:      agentID,
	}
	if err := s.agentsRepo.UpsertActiveAgent(ctx, agent); err != nil {
		return nil, fmt.Errorf("failed to upsert active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted active agent with ID: %s, ccagent_id: %v", agent.ID, agentID)
	return agent, nil
}

func (s *AgentsService) DeleteActiveAgentByWsConnectionID(
	ctx context.Context,
	wsConnectionID, organizationID string,
) error {
	log.Printf("📋 Starting to delete active agent by wsConnectionID: %s", wsConnectionID)
	if !core.IsValidULID(wsConnectionID) {
		return fmt.Errorf("ws_connection_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	// First find the agent by WebSocket connection ID
	maybeAgent, err := s.agentsRepo.GetAgentByWSConnectionID(ctx, wsConnectionID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to find agent by ws_connection_id: %w", err)
	}
	if !maybeAgent.IsPresent() {
		return core.ErrNotFound
	}
	agent := maybeAgent.MustGet()

	// Then delete by agent ID
	deleted, err := s.agentsRepo.DeleteActiveAgent(ctx, agent.ID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}
	if !deleted {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - deleted active agent with ID: %s", agent.ID)
	return nil
}

func (s *AgentsService) DeleteActiveAgent(ctx context.Context, id string, organizationID string) error {
	log.Printf("📋 Starting to delete active agent with ID: %s", id)
	if !core.IsValidULID(id) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	deleted, err := s.agentsRepo.DeleteActiveAgent(ctx, id, organizationID)
	if err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}
	if !deleted {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - deleted active agent with ID: %s", id)
	return nil
}

func (s *AgentsService) GetAgentByID(
	ctx context.Context,
	id string,
	organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	log.Printf("📋 Starting to get agent by ID: %s", id)
	if !core.IsValidULID(id) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeAgent, err := s.agentsRepo.GetAgentByID(ctx, id, organizationID)
	if err != nil {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get active agent: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("📋 Completed successfully - agent not found")
		return mo.None[*models.ActiveAgent](), nil
	}
	agent := maybeAgent.MustGet()

	log.Printf("📋 Completed successfully - retrieved agent with ID: %s", agent.ID)
	return mo.Some(agent), nil
}

func (s *AgentsService) GetAvailableAgents(
	ctx context.Context,
	organizationID string,
) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get available agents")
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	agents, err := s.agentsRepo.GetAvailableAgents(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d available agents", len(agents))
	return agents, nil
}

// GetConnectedActiveAgents returns only agents that have active WebSocket connections
func (s *AgentsService) GetConnectedActiveAgents(
	ctx context.Context,
	organizationID string,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get connected active agents")
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Get all active agents from database
	allAgents, err := s.agentsRepo.GetAllActiveAgents(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}
	log.Printf(
		"🔍 Found %d total agents in database, filtering by %d connected WebSocket clients",
		len(allAgents),
		len(connectedClientIDs),
	)

	// Create a map for faster lookup of connected client IDs
	connectedClientsMap := make(map[string]bool)
	for _, clientID := range connectedClientIDs {
		connectedClientsMap[clientID] = true
	}

	// Filter agents to only those with active WebSocket connections
	var connectedAgents []*models.ActiveAgent
	for _, agent := range allAgents {
		if connectedClientsMap[agent.WSConnectionID] {
			connectedAgents = append(connectedAgents, agent)
		}
	}

	log.Printf("📋 Completed successfully - retrieved %d connected active agents", len(connectedAgents))
	return connectedAgents, nil
}

// GetConnectedAvailableAgents returns only available agents that have active WebSocket connections
func (s *AgentsService) GetConnectedAvailableAgents(
	ctx context.Context,
	organizationID string,
	connectedClientIDs []string,
) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get connected available agents")
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	// Get all available agents from database
	availableAgents, err := s.agentsRepo.GetAvailableAgents(ctx, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	log.Printf(
		"🔍 Found %d available agents in database, filtering by %d connected WebSocket clients",
		len(availableAgents),
		len(connectedClientIDs),
	)

	// Create a map for faster lookup of connected client IDs
	connectedClientsMap := make(map[string]bool)
	for _, clientID := range connectedClientIDs {
		connectedClientsMap[clientID] = true
	}

	// Filter agents to only those with active WebSocket connections
	var connectedAvailableAgents []*models.ActiveAgent
	for _, agent := range availableAgents {
		if connectedClientsMap[agent.WSConnectionID] {
			connectedAvailableAgents = append(connectedAvailableAgents, agent)
		}
	}

	log.Printf("📋 Completed successfully - retrieved %d connected available agents", len(connectedAvailableAgents))
	return connectedAvailableAgents, nil
}

// CheckAgentHasActiveConnection verifies if an agent has an active WebSocket connection
func (s *AgentsService) CheckAgentHasActiveConnection(agent *models.ActiveAgent, connectedClientIDs []string) bool {
	log.Printf("📋 Starting to check if agent %s has active connection", agent.ID)

	// Create a map for faster lookup
	connectedClientsMap := make(map[string]bool)
	for _, clientID := range connectedClientIDs {
		connectedClientsMap[clientID] = true
	}

	hasConnection := connectedClientsMap[agent.WSConnectionID]
	log.Printf("📋 Completed check - agent %s has active connection: %t", agent.ID, hasConnection)
	return hasConnection
}

func (s *AgentsService) AssignAgentToJob(ctx context.Context, agentID, jobID string, organizationID string) error {
	log.Printf("📋 Starting to assign agent %s to job %s", agentID, jobID)
	if !core.IsValidULID(agentID) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	assignment := &models.AgentJobAssignment{
		ID:      core.NewID("aji"),
		AgentID: agentID,
		JobID:   jobID,
	}

	if err := s.agentsRepo.AssignAgentToJob(ctx, assignment); err != nil {
		return fmt.Errorf("failed to assign agent to job: %w", err)
	}

	log.Printf("📋 Completed successfully - assigned agent %s to job %s (or assignment already existed)", agentID, jobID)
	return nil
}

func (s *AgentsService) UnassignAgentFromJob(
	ctx context.Context,
	agentID, jobID string,
	organizationID string,
) error {
	log.Printf("📋 Starting to unassign agent %s from job %s", agentID, jobID)
	if !core.IsValidULID(agentID) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	unassigned, err := s.agentsRepo.UnassignAgentFromJob(ctx, agentID, jobID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}
	if !unassigned {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - unassigned agent %s from job %s", agentID, jobID)
	return nil
}

func (s *AgentsService) GetAgentByJobID(
	ctx context.Context,
	jobID string,
	organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	log.Printf("📋 Starting to get agent by job ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeAgent, err := s.agentsRepo.GetAgentByJobID(ctx, jobID, organizationID)
	if err != nil {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get agent by job ID: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("📋 Completed successfully - agent not found for job %s", jobID)
		return mo.None[*models.ActiveAgent](), nil
	}
	agent := maybeAgent.MustGet()

	log.Printf("📋 Completed successfully - retrieved agent %s for job %s", agent.ID, jobID)
	return mo.Some(agent), nil
}

func (s *AgentsService) GetAgentByWSConnectionID(
	ctx context.Context,
	wsConnectionID, organizationID string,
) (mo.Option[*models.ActiveAgent], error) {
	log.Printf("📋 Starting to get agent by WS connection ID: %s", wsConnectionID)
	if !core.IsValidULID(wsConnectionID) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("ws_connection_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("organization_id must be a valid ULID")
	}

	maybeAgent, err := s.agentsRepo.GetAgentByWSConnectionID(ctx, wsConnectionID, organizationID)
	if err != nil {
		return mo.None[*models.ActiveAgent](), fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}
	if !maybeAgent.IsPresent() {
		log.Printf("📋 Completed successfully - agent not found for WS connection")
		return mo.None[*models.ActiveAgent](), nil
	}
	agent := maybeAgent.MustGet()

	log.Printf("📋 Completed successfully - retrieved agent %s for WS connection %s", agent.ID, wsConnectionID)
	return mo.Some(agent), nil
}

func (s *AgentsService) GetActiveAgentJobAssignments(
	ctx context.Context,
	agentID string,
	organizationID string,
) ([]string, error) {
	log.Printf("📋 Starting to get active job assignments for agent %s", agentID)
	if !core.IsValidULID(agentID) {
		return nil, fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}

	jobIDs, err := s.agentsRepo.GetActiveAgentJobAssignments(ctx, agentID, organizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent job assignments: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d job assignments for agent %s", len(jobIDs), agentID)
	return jobIDs, nil
}

func (s *AgentsService) UpdateAgentLastActiveAt(ctx context.Context, wsConnectionID, organizationID string) error {
	log.Printf("📋 Starting to update last_active_at for agent with WS connection ID: %s", wsConnectionID)
	if !core.IsValidULID(wsConnectionID) {
		return fmt.Errorf("ws_connection_id must be a valid ULID")
	}
	if !core.IsValidULID(organizationID) {
		return fmt.Errorf("organization_id must be a valid ULID")
	}

	updated, err := s.agentsRepo.UpdateAgentLastActiveAt(ctx, wsConnectionID, organizationID)
	if err != nil {
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}
	if !updated {
		return core.ErrNotFound
	}

	log.Printf("📋 Completed successfully - updated last_active_at for agent with WS connection %s", wsConnectionID)
	return nil
}

func (s *AgentsService) GetInactiveAgents(
	ctx context.Context,
	organizationID string,
	inactiveThresholdMinutes int,
) ([]*models.ActiveAgent, error) {
	log.Printf(
		"📋 Starting to get inactive agents for integration %s (threshold: %d minutes)",
		organizationID,
		inactiveThresholdMinutes,
	)

	if !core.IsValidULID(organizationID) {
		return nil, fmt.Errorf("organization_id must be a valid ULID")
	}
	if inactiveThresholdMinutes <= 0 {
		return nil, fmt.Errorf("inactive threshold must be positive")
	}

	agents, err := s.agentsRepo.GetInactiveAgents(ctx, organizationID, inactiveThresholdMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive agents: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d inactive agents", len(agents))
	return agents, nil
}
