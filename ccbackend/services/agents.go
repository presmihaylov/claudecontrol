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

func (s *AgentsService) CreateActiveAgent(wsConnectionID, slackIntegrationID string, assignedJobID *uuid.UUID) (*models.ActiveAgent, error) {
	log.Printf("📋 Starting to create active agent for wsConnectionID: %s", wsConnectionID)

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
		AssignedJobID:      assignedJobID,
		WSConnectionID:     wsConnectionID,
		SlackIntegrationID: integrationUUID,
	}

	if err := s.agentsRepo.CreateActiveAgent(agent); err != nil {
		return nil, fmt.Errorf("failed to create active agent: %w", err)
	}

	log.Printf("📋 Completed successfully - created active agent with ID: %s", agent.ID)
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

func (s *AgentsService) DeleteAllActiveAgents() error {
	log.Printf("📋 Starting to delete all active agents")

	if err := s.agentsRepo.DeleteAllActiveAgents(); err != nil {
		return fmt.Errorf("failed to delete all active agents: %w", err)
	}

	log.Printf("📋 Completed successfully - deleted all active agents")
	return nil
}

func (s *AgentsService) AssignJobToAgent(agentID, jobID uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to assign job %s to agent %s", jobID, agentID)

	if agentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be nil")
	}

	if jobID == uuid.Nil {
		return fmt.Errorf("job ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.agentsRepo.UpdateAgentJobAssignment(agentID, &jobID, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to assign job to agent: %w", err)
	}

	log.Printf("📋 Completed successfully - assigned job %s to agent %s", jobID, agentID)
	return nil
}

func (s *AgentsService) UnassignJobFromAgent(agentID uuid.UUID, slackIntegrationID string) error {
	log.Printf("📋 Starting to unassign job from agent %s", agentID)

	if agentID == uuid.Nil {
		return fmt.Errorf("agent ID cannot be nil")
	}

	if slackIntegrationID == "" {
		return fmt.Errorf("slack_integration_id cannot be empty")
	}

	if err := s.agentsRepo.UpdateAgentJobAssignment(agentID, nil, slackIntegrationID); err != nil {
		return fmt.Errorf("failed to unassign job from agent: %w", err)
	}

	log.Printf("📋 Completed successfully - unassigned job from agent %s", agentID)
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

