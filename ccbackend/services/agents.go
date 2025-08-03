package services

import (
	"fmt"
	"log"

	"ccbackend/db"
	"ccbackend/models"

	"github.com/google/uuid"
)

type AgentsService struct {
	agentsRepo *db.PostgresAgentsRepository
}

func NewAgentsService(repo *db.PostgresAgentsRepository) *AgentsService {
	return &AgentsService{agentsRepo: repo}
}

func (s *AgentsService) CreateActiveAgent(wsConnectionID, slackIntegrationID string, agentID uuid.UUID) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to create active agent for wsConnectionID: %s, agentID: %s", wsConnectionID, agentID)

	if wsConnectionID == "" {
		return nil, fmt.Errorf("ws_connection_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	id := uuid.New()
	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("invalid slack_integration_id format: %w", err)
	}

	agent := &models.ActiveAgent{
		ID:                 id,
		WSConnectionID:     wsConnectionID,
		SlackIntegrationID: integrationUUID,
		AgentID:            agentID,
	}

	if err := s.agentsRepo.CreateActiveAgent(agent); err != nil {
		return nil, fmt.Errorf("failed to create active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - created active agent with ID: %s, agent_id: %v", agent.ID, agentID)
	return agent, nil
}

func (s *AgentsService) DeleteActiveAgentByWsConnectionID(wsConnectionID, slackIntegrationID string) error {
	log.Printf("📋 Starting to delete active agent by wsConnectionID: %s", wsConnectionID)

	if wsConnectionID == "" {
		return fmt.Errorf("ws_connection_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
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

func (s *AgentsService) DeleteActiveAgent(id uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to delete active agent with ID: %s", id)

	if id == uuid.Nil {
		return fmt.Errorf("agent ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.agentsRepo.DeleteActiveAgent(id, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted active agent with ID: %s", id)
	return nil
}

func (s *AgentsService) GetAgentByID(id uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by ID: %s", id)

	if id == uuid.Nil {
		return nil, fmt.Errorf("agent ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
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

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	agents, err := s.agentsRepo.GetAvailableAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d available agents", len(agents))
	return agents, nil
}

func (s *AgentsService) GetAllActiveAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get all active agents")

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	agents, err := s.agentsRepo.GetAllActiveAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved %d active agents", len(agents))
	return agents, nil
}

// GetConnectedActiveAgents returns only agents that have active WebSocket connections
func (s *AgentsService) GetConnectedActiveAgents(slackIntegrationID string, connectedClientIDs []string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get connected active agents")

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
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

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
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

// GetStaleAgents returns agents that don't have active WebSocket connections
func (s *AgentsService) GetStaleAgents(slackIntegrationID string, connectedClientIDs []string) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get stale agents")

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	// Get all active agents from database
	allAgents, err := s.agentsRepo.GetAllActiveAgents(slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	log.Printf("🔍 Found %d total agents in database, checking against %d connected WebSocket clients", len(allAgents), len(connectedClientIDs))

	// Create a map for faster lookup of connected client IDs
	connectedClientsMap := make(map[string]bool)
	for _, clientID := range connectedClientIDs {
		connectedClientsMap[clientID] = true
	}

	// Filter agents to only those WITHOUT active WebSocket connections
	var staleAgents []*models.ActiveAgent
	for _, agent := range allAgents {
		if !connectedClientsMap[agent.WSConnectionID] {
			staleAgents = append(staleAgents, agent)
		}
	}

	log.Printf("📋 Completed successfully - found %d stale agents", len(staleAgents))
	return staleAgents, nil
}

func (s *AgentsService) DeleteAllActiveAgents() error {
	log.Printf("📋 Starting to delete all active agents")

	if err := s.agentsRepo.DeleteAllActiveAgents(); err != nil {
		return fmt.Errorf("failed to delete all active agents: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted all active agents")
	return nil
}

func (s *AgentsService) AssignAgentToJob(agentID, jobID uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to assign agent %s to job %s", agentID, jobID)

	if agentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be nil")
	}

	if jobID == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	integrationUUID, err := uuid.Parse(slackIntegrationID)
	if err != nil {
		return fmt.Errorf("invalid slack_integration_id format: %w", err)
	}

	assignment := &models.AgentJobAssignment{
		ID:                 uuid.New(),
		AgentID:            agentID,
		JobID:              jobID,
		SlackIntegrationID: integrationUUID,
	}

	if err := s.agentsRepo.AssignAgentToJob(assignment); err != nil {
		return fmt.Errorf("failed to assign agent to job: %w", err)
	}

	log.Printf("📋 Completed successfully - assigned agent %s to job %s", agentID, jobID)
	return nil
}

func (s *AgentsService) UnassignAgentFromJob(agentID, jobID uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to unassign agent %s from job %s", agentID, jobID)

	if agentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be nil")
	}

	if jobID == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.agentsRepo.UnassignAgentFromJob(agentID, jobID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}

	log.Printf("📋 Completed successfully - unassigned agent %s from job %s", agentID, jobID)
	return nil
}

func (s *AgentsService) GetAgentByJobID(jobID uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by job ID: %s", jobID)

	if jobID == uuid.Nil {
		return nil, fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	agent, err := s.agentsRepo.GetAgentByJobID(jobID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by job ID: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved agent %s for job %s", agent.ID, jobID)
	return agent, nil
}

func (s *AgentsService) GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get agent by WS connection ID: %s", wsConnectionID)

	if wsConnectionID == "" {
		return nil, fmt.Errorf("ws_connection_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
	}

	agent, err := s.agentsRepo.GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get agent by WS connection ID: %w", err)
	}

	log.Printf("📋 Completed successfully - retrieved agent %s for WS connection %s", agent.ID, wsConnectionID)
	return agent, nil
}

func (s *AgentsService) GetActiveAgentJobAssignments(agentID uuid.UUID, slackIntegrationID string) ([]uuid.UUID, error) {
	log.Printf("📋 Starting to get active job assignments for agent %s", agentID)

	if agentID == uuid.Nil {
		return nil, fmt.Errorf("agent ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
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

	if wsConnectionID == "" {
		return fmt.Errorf("ws_connection_id cannot be empty")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.agentsRepo.UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	log.Printf("📋 Completed successfully - updated last_active_at for agent with WS connection %s", wsConnectionID)
	return nil
}

func (s *AgentsService) GetInactiveAgents(slackIntegrationID string, inactiveThresholdMinutes int) ([]*models.ActiveAgent, error) {
	log.Printf("📋 Starting to get inactive agents for integration %s (threshold: %d minutes)", slackIntegrationID, inactiveThresholdMinutes)

	if slackIntegrationID == "" {
		return nil, fmt.Errorf("slack_integration_id cannot be empty")
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
