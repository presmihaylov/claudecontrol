package db

import (
	"database/sql"
	"fmt"

	"ccbackend/models"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	// necessary import to wire up the postgres driver
	_ "github.com/lib/pq"
)

type PostgresAgentsRepository struct {
	db     *sqlx.DB
	schema string
}

func NewPostgresAgentsRepository(db *sqlx.DB, schema string) *PostgresAgentsRepository {
	return &PostgresAgentsRepository{db: db, schema: schema}
}

func (r *PostgresAgentsRepository) UpsertActiveAgent(agent *models.ActiveAgent) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.active_agents (id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at) 
		VALUES ($1, $2, $3, $4, NOW(), NOW(), NOW()) 
		ON CONFLICT (slack_integration_id, agent_id) 
		DO UPDATE SET 
			ws_connection_id = EXCLUDED.ws_connection_id,
			updated_at = NOW(),
			last_active_at = NOW()
		RETURNING id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at`, r.schema)

	err := r.db.QueryRowx(query, agent.ID, agent.WSConnectionID, agent.SlackIntegrationID, agent.AgentID).StructScan(agent)
	if err != nil {
		return fmt.Errorf("failed to upsert active agent: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) DeleteActiveAgent(id uuid.UUID, slackIntegrationID string) error {
	query := fmt.Sprintf("DELETE FROM %s.active_agents WHERE id = $1 AND slack_integration_id = $2", r.schema)

	result, err := r.db.Exec(query, id, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to delete active agent: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active agent with id %s not found", id)
	}

	return nil
}

func (r *PostgresAgentsRepository) GetAgentByID(id uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, id, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with id %s not found", id)
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAgentByWSConnectionID(wsConnectionID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE ws_connection_id = $1 AND slack_integration_id = $2`, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, wsConnectionID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with ws_connection_id %s not found", wsConnectionID)
		}
		return nil, fmt.Errorf("failed to get active agent: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) GetAvailableAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT a.id, a.ws_connection_id, a.slack_integration_id, a.agent_id, a.created_at, a.updated_at, a.last_active_at 
		FROM %s.active_agents a
		LEFT JOIN %s.agent_job_assignments aja ON a.id = aja.agent_id
		WHERE aja.agent_id IS NULL AND a.slack_integration_id = $1
		ORDER BY a.created_at ASC`, r.schema, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get available agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAllActiveAgents(slackIntegrationID string) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE slack_integration_id = $1
		ORDER BY created_at ASC`, r.schema)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get all active agents: %w", err)
	}

	return agents, nil
}

func (r *PostgresAgentsRepository) GetAgentByJobID(jobID uuid.UUID, slackIntegrationID string) (*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT a.id, a.ws_connection_id, a.slack_integration_id, a.agent_id, a.created_at, a.updated_at, a.last_active_at 
		FROM %s.active_agents a
		JOIN %s.agent_job_assignments aja ON a.id = aja.agent_id
		WHERE aja.job_id = $1 AND aja.slack_integration_id = $2
		LIMIT 1`, r.schema, r.schema)

	agent := &models.ActiveAgent{}
	err := r.db.Get(agent, query, jobID, slackIntegrationID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("active agent with assigned_job_id %s not found", jobID)
		}
		return nil, fmt.Errorf("failed to get active agent by job ID: %w", err)
	}

	return agent, nil
}

func (r *PostgresAgentsRepository) AssignAgentToJob(assignment *models.AgentJobAssignment) error {
	query := fmt.Sprintf(`
		INSERT INTO %s.agent_job_assignments (id, agent_id, job_id, slack_integration_id, assigned_at) 
		VALUES ($1, $2, $3, $4, NOW()) 
		RETURNING id, agent_id, job_id, slack_integration_id, assigned_at`, r.schema)

	err := r.db.QueryRowx(query, assignment.ID, assignment.AgentID, assignment.JobID, assignment.SlackIntegrationID).StructScan(assignment)
	if err != nil {
		return fmt.Errorf("failed to assign agent to job: %w", err)
	}

	return nil
}

func (r *PostgresAgentsRepository) UnassignAgentFromJob(agentID, jobID uuid.UUID, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		DELETE FROM %s.agent_job_assignments 
		WHERE agent_id = $1 AND job_id = $2 AND slack_integration_id = $3`, r.schema)

	result, err := r.db.Exec(query, agentID, jobID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to unassign agent from job: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("agent-job assignment not found")
	}

	return nil
}

func (r *PostgresAgentsRepository) GetActiveAgentJobAssignments(agentID uuid.UUID, slackIntegrationID string) ([]uuid.UUID, error) {
	query := fmt.Sprintf(`
		SELECT job_id 
		FROM %s.agent_job_assignments
		WHERE agent_id = $1 AND slack_integration_id = $2
		ORDER BY assigned_at ASC`, r.schema)

	var jobIDs []uuid.UUID
	err := r.db.Select(&jobIDs, query, agentID, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get active agent job assignments: %w", err)
	}

	return jobIDs, nil
}

func (r *PostgresAgentsRepository) UpdateAgentLastActiveAt(wsConnectionID, slackIntegrationID string) error {
	query := fmt.Sprintf(`
		UPDATE %s.active_agents 
		SET last_active_at = NOW() 
		WHERE ws_connection_id = $1 AND slack_integration_id = $2`, r.schema)

	result, err := r.db.Exec(query, wsConnectionID, slackIntegrationID)
	if err != nil {
		return fmt.Errorf("failed to update agent last_active_at: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("active agent with ws_connection_id %s not found", wsConnectionID)
	}

	return nil
}

func (r *PostgresAgentsRepository) GetInactiveAgents(slackIntegrationID string, inactiveThresholdMinutes int) ([]*models.ActiveAgent, error) {
	query := fmt.Sprintf(`
		SELECT id, ws_connection_id, slack_integration_id, agent_id, created_at, updated_at, last_active_at 
		FROM %s.active_agents 
		WHERE slack_integration_id = $1 AND last_active_at < NOW() - INTERVAL '%d minutes'
		ORDER BY last_active_at ASC`, r.schema, inactiveThresholdMinutes)

	var agents []*models.ActiveAgent
	err := r.db.Select(&agents, query, slackIntegrationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get inactive agents: %w", err)
	}

	return agents, nil
}
