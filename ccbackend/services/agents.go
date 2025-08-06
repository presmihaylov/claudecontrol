package services

import (
	"fmt"
	"log"
	"strings"

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

func (s *AgentsService) UpsertActiveAgent(wsConnectionID, slackIntegrationID string, agentID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to upsert active agent for wsConnectionID: %s, agentID: %s", wsConnectionID, agentID)

	if !core.IsValidULID(wsConnectionID) {
		return nil, fmt.Errorf("ws_connection_id must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	if !core.IsValidULID(agentID) {
		return nil, fmt.Errorf("ccagent_id must be a valid ULID")
	}

	agent := &models.ActiveAgent{
		ID:                 core.NewID("ag"),
		WSConnectionID:     wsConnectionID,
		SlackIntegrationID: slackIntegrationID,
		CcagentID:          agentID,
	}

	if err := s.agentsRepo.UpsertActiveAgent(agent); err != nil {
		return nil, fmt.Errorf("failed to upsert active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - upserted active agent with ID: %s, ccagent_id: %v", agent.ID, agentID)
	return agent, nil
}

func (s *AgentsService) DeleteActiveAgentByWsConnectionID(wsConnectionID, slackIntegrationID string) error {
	log.Printf("📋 Starting to delete active agent by wsConnectionID: %s", wsConnectionID)

	if !core.IsValidULID(wsConnectionID) {
		return fmt.Errorf("ws_connection_id must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	// First find the agent by WebSocket connection ID
	agent, err := s.agentsRepo.GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to find agent by ws_connection_id: %w", err)
	}

	// Then delete by agent ID
	if err := s.agentsRepo.DeleteActiveAgent(agent.ID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted active agent with ID: %s", agent.ID)
	return nil
}

func (s *AgentsService) DeleteActiveAgent(id string, slackIntegrationID string) error {
	log.Printf("📋 Starting to delete active agent with ID: %s", id)

	if !core.IsValidULID(id) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	if err := s.agentsRepo.DeleteActiveAgent(id, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted active agent with ID: %s", id)
	return nil
}

func (s *AgentsService) GetAgentByID(id string, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by ID: %s", id)

	if !core.IsValidULID(id) {
		return nil, fmt.Errorf("agent ID must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	agent, err := s.agentsRepo.GetAgentByID(id, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved agent with ID: %s", agent.ID)
	return agent, nil
}

func (s *AgentsService) GetAvailableAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get available agents")

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	agents, err := s.agentsRepo.GetAvailableAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d available agents", len(agents))
	return agents, nil
}

// GetConnectedActiveAgents returns only agents that have active WebSocket connections
func (s *AgentsService) GetConnectedActiveAgents(slackIntegrationID string, connectedClientIDs []string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get connected active agents")

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	// Get all active agents from database
	allAgents, err := s.agentsRepo.GetAllActiveAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	log.Printf("🔍 Found %d total agents in database, filtering by %d connected WebSocket clients", len(allAgents), len(connectedClientIDs))

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
func (s *AgentsService) GetConnectedAvailableAgents(slackIntegrationID string, connectedClientIDs []string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get connected available agents")

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	// Get all available agents from database
	availableAgents, err := s.agentsRepo.GetAvailableAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	log.Printf("🔍 Found %d available agents in database, filtering by %d connected WebSocket clients", len(availableAgents), len(connectedClientIDs))

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

func (s *AgentsService) AssignAgentToJob(agentID, jobID string, slackIntegrationID string) error {
	log.Printf("📋 Starting to assign agent %s to job %s", agentID, jobID)
	if !core.IsValidULID(agentID) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}
	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	assignment := &models.AgentJobAssignment{
		ID:                 core.NewID("aji"),
		CcagentID:          agentID,
		JobID:              jobID,
		SlackIntegrationID: slackIntegrationID,
	}

	if err := s.agentsRepo.AssignAgentToJob(assignment); err != nil {
		return fmt.Errorf("failed to assign agent to job: %w", err)
	}

	log.Printf("📋 Completed successfully - assigned agent %s to job %s (or assignment already existed)", agentID, jobID)
	return nil
}

func (s *AgentsService) UnassignAgentFromJob(agentID, jobID string, slackIntegrationID string) error {
	log.Printf("📋 Starting to unassign agent %s from job %s", agentID, jobID)

	if !core.IsValidULID(agentID) {
		return fmt.Errorf("agent ID must be a valid ULID")
	}

	if !core.IsValidULID(jobID) {
		return fmt.Errorf("job ID must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	if err := s.agentsRepo.UnassignAgentFromJob(agentID, jobID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}

	log.Printf("📋 Completed successfully - unassigned agent %s from job %s", agentID, jobID)
	return nil
}

func (s *AgentsService) GetAgentByJobID(jobID string, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by job ID: %s", jobID)
	if !core.IsValidULID(jobID) {
		return nil, fmt.Errorf("job ID must be a valid ULID")
	}
	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	agent, err := s.agentsRepo.GetAgentByJobID(jobID, slackIntegrationID)
	if err != nil {
		// Check if it's a not found error
		if strings.Contains(err.Error(), "not found") {
			log.Printf("📋 No agent found for job %s", jobID)
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get agent by job ID: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved agent %s for job %s", agent.ID, jobID)
	return agent, nil
}

func (s *AgentsService) GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by WS connection ID: %s", wsConnectionID)

	if !core.IsValidULID(wsConnectionID) {
		return nil, fmt.Errorf("ws_connection_id must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	agent, err := s.agentsRepo.GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved agent %s for WS connection %s", agent.ID, wsConnectionID)
	return agent, nil
}

func (s *AgentsService) GetActiveAgentJobAssignments(agentID string, slackIntegrationID string) ([]string, error) {
	log.Printf("📋 Starting to get active job assignments for agent %s", agentID)

	if !core.IsValidULID(agentID) {
		return nil, fmt.Errorf("agent ID must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	jobIDs, err := s.agentsRepo.GetActiveAgentJobAssignments(agentID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent job assignments: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d job assignments for agent %s", len(jobIDs), agentID)
	return jobIDs, nil
}

func (s *AgentsService) UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID string) error {
	log.Printf("📋 Starting to update last_active_at for agent with WS connection ID: %s", wsConnectionID)

	if !core.IsValidULID(wsConnectionID) {
		return fmt.Errorf("ws_connection_id must be a valid ULID")
	}

	if !core.IsValidULID(slackIntegrationID) {
		return fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	if err := s.agentsRepo.UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated last_active_at for agent with WS connection %s", wsConnectionID)
	return nil
}

func (s *AgentsService) GetInactiveAgents(slackIntegrationID string, inactiveThresholdMinutes int) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get inactive agents for integration %s (threshold: %d minutes)", slackIntegrationID, inactiveThresholdMinutes)

	if !core.IsValidULID(slackIntegrationID) {
		return nil, fmt.Errorf("slack_integration_id must be a valid ULID")
	}

	if inactiveThresholdMinutes <= 0 {
		return nil, fmt.Errorf("inactive threshold must be positive")
	}

	agents, err := s.agentsRepo.GetInactiveAgents(slackIntegrationID, inactiveThresholdMinutes)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive agents: %w", err)
	}

	log.Printf("📋 Completed successfully - found %d inactive agents", len(agents))
	return agents, nil
}
